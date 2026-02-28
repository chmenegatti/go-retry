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
