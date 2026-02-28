package retry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockTimer struct allows us to simulate time passing instantly.
type mockTimer struct {
	mu     sync.Mutex
	timers []time.Duration
}

func (m *mockTimer) After(d time.Duration) <-chan time.Time {
	m.mu.Lock()
	m.timers = append(m.timers, d)
	m.mu.Unlock()

	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func TestRetryer_SuccessFirstTry(t *testing.T) {
	r := New()
	attempts := 0
	err := r.Run(context.Background(), func(ctx context.Context) error {
		attempts++
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryer_SuccessAfterFailures(t *testing.T) {
	callbackAttempts := []int{}
	var mu sync.Mutex

	r := New(
		WithMaxRetries(3),
		WithJitter(false),
		WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			mu.Lock()
			callbackAttempts = append(callbackAttempts, attempt)
			mu.Unlock()
		}),
	)

	// Inject mock timer
	mTimer := &mockTimer{}
	r.timer = mTimer

	attempts := 0
	err := r.Run(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts <= 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}

	mu.Lock()
	if len(callbackAttempts) != 2 {
		t.Fatalf("expected OnRetry to be called 2 times, got %d", len(callbackAttempts))
	}
	if callbackAttempts[0] != 0 || callbackAttempts[1] != 1 {
		t.Fatalf("expected attempts 0 and 1 in callback, got %v", callbackAttempts)
	}
	mu.Unlock()
}

func TestRetryer_MaxRetriesExceeded(t *testing.T) {
	r := New(WithMaxRetries(2))
	r.timer = &mockTimer{}

	expectedErr := errors.New("permanent error")
	attempts := 0
	err := r.Run(context.Background(), func(ctx context.Context) error {
		attempts++
		return expectedErr
	})

	if err != expectedErr {
		t.Fatalf("expected returned error to be %v, got %v", expectedErr, err)
	}

	// Initial attempt + 2 retries = 3 attempts total
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryer_ContextCancellation(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	attempts := 0

	go func() {
		defer wg.Done()
		err := r.Run(ctx, func(ctx context.Context) error {
			attempts++
			cancel() // cancel context during the first execution
			return errors.New("some error to trigger retry loop")
		})

		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	}()

	wg.Wait()

	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt before cancellation was caught, got %d", attempts)
	}
}

func TestRetryer_PreCanceledContext(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	err := r.Run(ctx, func(ctx context.Context) error {
		attempts++
		return nil
	})

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if attempts != 0 {
		t.Fatalf("expected 0 attempts for pre-canceled context, got %d", attempts)
	}
}

func TestRetryer_RetryIf(t *testing.T) {
	errDoNotRetry := errors.New("do not retry me")
	errRetry := errors.New("retry me")

	r := New(
		WithMaxRetries(3),
		WithRetryIf(func(err error) bool {
			return err != errDoNotRetry
		}),
	)
	r.timer = &mockTimer{}

	// Test 1: Error that should NOT be retried
	attempts1 := 0
	err := r.Run(context.Background(), func(ctx context.Context) error {
		attempts1++
		return errDoNotRetry
	})
	if err != errDoNotRetry {
		t.Fatalf("expected errDoNotRetry, got %v", err)
	}
	if attempts1 != 1 {
		t.Fatalf("expected exactly 1 attempt due to RetryIf abort, got %d", attempts1)
	}

	// Test 2: Error that SHOULD be retried
	attempts2 := 0
	err = r.Run(context.Background(), func(ctx context.Context) error {
		attempts2++
		return errRetry
	})
	if err != errRetry {
		t.Fatalf("expected errRetry, got %v", err)
	}
	if attempts2 != 4 { // 1 initial + 3 retries
		t.Fatalf("expected exactly 4 attempts, got %d", attempts2)
	}
}

func TestDo_Success(t *testing.T) {
	r := New(WithMaxRetries(2))
	r.timer = &mockTimer{}

	attempts := 0
	val, err := Do(context.Background(), r, func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("temporary error")
		}
		return "success_value", nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != "success_value" {
		t.Fatalf("expected 'success_value', got %v", val)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDo_Failure(t *testing.T) {
	r := New(WithMaxRetries(1))
	r.timer = &mockTimer{}

	expectedErr := errors.New("permanent error")
	val, err := Do(context.Background(), r, func(ctx context.Context) (int, error) {
		return 42, expectedErr
	})

	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if val != 0 {
		t.Fatalf("expected zero value, got %v", val)
	}
}

func TestRetryer_OnSuccess(t *testing.T) {
	successCalled := false
	var successAttempt int

	r := New(
		WithMaxRetries(3),
		WithOnSuccess(func(attempt int) {
			successCalled = true
			successAttempt = attempt
		}),
	)
	r.timer = &mockTimer{}

	attempts := 0
	err := r.Run(context.Background(), func(ctx context.Context) error {
		attempts++
		if attempts < 3 { // Fail on attempt 0 and 1
			return errors.New("temp error")
		}
		return nil // Success on attempt 2
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !successCalled {
		t.Fatal("expected OnSuccess to be called")
	}
	if successAttempt != 2 {
		t.Fatalf("expected OnSuccess to be called with attempt 2, got %d", successAttempt)
	}
}

func TestRetryer_OnDrop(t *testing.T) {
	dropCalled := false
	var dropAttempt int
	var dropErr error
	expectedErr := errors.New("permanent failure")

	r := New(
		WithMaxRetries(2),
		WithOnDrop(func(attempt int, err error) {
			dropCalled = true
			dropAttempt = attempt
			dropErr = err
		}),
	)
	r.timer = &mockTimer{}

	err := r.Run(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if !dropCalled {
		t.Fatal("expected OnDrop to be called")
	}
	if dropAttempt != 2 { // MaxRetries
		t.Fatalf("expected OnDrop attempt to be 2, got %d", dropAttempt)
	}
	if dropErr != expectedErr {
		t.Fatalf("expected OnDrop err to be %v, got %v", expectedErr, dropErr)
	}
}

func TestDoFunc(t *testing.T) {
	attempts := 0
	err := DoFunc(
		context.Background(),
		func(ctx context.Context) error {
			attempts++
			if attempts < 2 {
				return errors.New("temp error")
			}
			return nil // Success
		},
		WithMaxRetries(2),
		WithJitter(false),
		WithInitialDelay(1*time.Millisecond),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDoValue(t *testing.T) {
	attempts := 0
	val, err := DoValue(
		context.Background(),
		func(ctx context.Context) (int, error) {
			attempts++
			if attempts < 2 {
				return 0, errors.New("temp error")
			}
			return 42, nil // Success
		},
		WithMaxRetries(2),
		WithJitter(false),
		WithInitialDelay(1*time.Millisecond),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != 42 {
		t.Fatalf("expected value 42, got %v", val)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
