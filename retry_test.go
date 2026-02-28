package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySuccess(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := New().
		Attempts(5).
		Do(ctx, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("fail")
			}
			return nil
		})

	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts got %d", attempts)
	}
}

func TestRetryFailure(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := New().
		Attempts(3).
		Do(ctx, func() error {
			attempts++
			return errors.New("fail")
		})

	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts got %d", attempts)
	}
}

func TestRetryContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := New().
		Attempts(5).
		Do(ctx, func() error {
			return errors.New("fail")
		})

	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestRetryBackoff(t *testing.T) {
	ctx := context.Background()
	start := time.Now()

	New().
		Attempts(2).
		Backoff(Constant(200*time.Millisecond)).
		Do(ctx, func() error {
			return errors.New("fail")
		})

	if time.Since(start) < 200*time.Millisecond {
		t.Fatal("backoff not applied")
	}
}

func TestDoValue(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	result, err := DoValue(ctx,
		New().Attempts(5).Backoff(Constant(time.Millisecond)),
		func() (int, error) {
			attempts++
			if attempts < 3 {
				return 0, errors.New("fail")
			}
			return 42, nil
		},
	)

	if err != nil {
		t.Fatal(err)
	}
	if result != 42 {
		t.Fatalf("expected 42, got %v", result)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

// --- RetryIf ---

func TestRetryIf_Allowed(t *testing.T) {
	ctx := context.Background()
	transient := errors.New("transient")
	attempts := 0

	err := New().
		Attempts(3).
		Backoff(Constant(time.Millisecond)).
		RetryIf(func(err error) bool { return errors.Is(err, transient) }).
		Do(ctx, func() error {
			attempts++
			return transient
		})

	// All 3 attempts should have been made.
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if !errors.Is(err, transient) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryIf_Denied(t *testing.T) {
	ctx := context.Background()
	fatal := errors.New("fatal")
	attempts := 0

	err := New().
		Attempts(5).
		RetryIf(func(err error) bool { return false }). // always deny
		Do(ctx, func() error {
			attempts++
			return fatal
		})

	// Must stop after first attempt.
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	if !errors.Is(err, fatal) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Permanent ---

func TestPermanentError_StopsImmediately(t *testing.T) {
	ctx := context.Background()
	base := errors.New("original error")
	attempts := 0

	err := New().Attempts(5).Do(ctx, func() error {
		attempts++
		return Permanent(base)
	})

	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
	if !errors.Is(err, base) {
		t.Fatalf("expected base error to be preserved, got %v", err)
	}
	if !IsPermanent(err) {
		t.Fatal("expected IsPermanent to return true")
	}
}

func TestPermanentNil(t *testing.T) {
	if Permanent(nil) != nil {
		t.Fatal("Permanent(nil) should return nil")
	}
}

func TestIsPermanent_False(t *testing.T) {
	if IsPermanent(errors.New("regular")) {
		t.Fatal("expected false for non-permanent error")
	}
}

// --- MaxDelay ---

func TestMaxDelay_Caps(t *testing.T) {
	ctx := context.Background()
	start := time.Now()

	// Exponential base=1s, attempt=1 → 1s, but MaxDelay=50ms
	New().
		Attempts(2).
		Backoff(Exponential(time.Second)).
		MaxDelay(50*time.Millisecond).
		Do(ctx, func() error {
			return errors.New("fail")
		})

	elapsed := time.Since(start)
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("MaxDelay not applied, elapsed: %v", elapsed)
	}
}
