package retry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestExponentialJitterRandomness(t *testing.T) {

	backoff := ExponentialJitter(100 * time.Millisecond)

	values := make(map[time.Duration]bool)

	for i := 0; i < 100; i++ {

		d := backoff(3)

		values[d] = true
	}

	if len(values) < 5 {
		t.Fatal("jitter does not appear random")
	}
}

func TestExponentialJitterBounds(t *testing.T) {

	base := 100 * time.Millisecond

	backoff := ExponentialJitter(base)

	for i := 0; i < 100; i++ {

		d := backoff(3)

		exp := base * 4

		if d < exp/2 || d > exp {
			t.Fatalf("jitter out of bounds: %v", d)
		}
	}
}

func TestRetryConcurrentCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	go func() {

		time.Sleep(50 * time.Millisecond)
		cancel()

	}()

	err := New().
		Attempts(10).
		Backoff(Constant(100*time.Millisecond)).
		Do(ctx, func() error {

			return errors.New("fail")

		})

	if err == nil {
		t.Fatal("expected context cancellation")
	}
}

func TestRetryConcurrentUsage(t *testing.T) {

	ctx := context.Background()

	r := New().
		Attempts(3).
		Backoff(Constant(time.Millisecond))

	wg := sync.WaitGroup{}

	for i := 0; i < 50; i++ {

		wg.Add(1)

		go func() {

			defer wg.Done()

			r.Do(ctx, func() error {

				return errors.New("fail")

			})

		}()

	}

	wg.Wait()
}

func TestExponentialBackoffValues(t *testing.T) {

	base := 100 * time.Millisecond

	backoff := Exponential(base)

	tests := []struct {
		attempt int
		expect  time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
	}

	for _, tt := range tests {

		got := backoff(tt.attempt)

		if got != tt.expect {
			t.Fatalf("expected %v got %v", tt.expect, got)
		}
	}
}

func TestZeroAttempts(t *testing.T) {

	ctx := context.Background()

	err := New().
		Attempts(0).
		Do(ctx, func() error {

			return errors.New("fail")

		})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOnRetryHook(t *testing.T) {

	ctx := context.Background()

	count := 0

	New().
		Attempts(3).
		OnRetry(func(int, error) {

			count++

		}).
		Do(ctx, func() error {

			return errors.New("fail")

		})

	if count != 2 {
		t.Fatalf("expected 2 retries got %d", count)
	}
}

func TestLinearBackoff(t *testing.T) {
	b := Linear(100 * time.Millisecond)
	for _, tt := range []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 300 * time.Millisecond},
	} {
		if got := b(tt.attempt); got != tt.want {
			t.Fatalf("Linear(%d): got %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestExponential_AttemptZero(t *testing.T) {
	b := Exponential(100 * time.Millisecond)
	if d := b(0); d != 100*time.Millisecond {
		t.Fatalf("Exponential(0) should return base, got %v", d)
	}
}

func TestExponentialJitter_AttemptZero(t *testing.T) {
	b := ExponentialJitter(100 * time.Millisecond)
	if d := b(0); d != 100*time.Millisecond {
		t.Fatalf("ExponentialJitter(0) should return base, got %v", d)
	}
}

func TestExponentialJitter_AttemptOne(t *testing.T) {
	b := ExponentialJitter(100 * time.Millisecond)
	d := b(1)
	if d < 50*time.Millisecond || d > 100*time.Millisecond {
		t.Fatalf("ExponentialJitter(1) out of range [50ms,100ms]: %v", d)
	}
}

func TestPermanentError_ErrorString(t *testing.T) {
	pErr := Permanent(errors.New("something went wrong"))
	if pErr.Error() != "something went wrong" {
		t.Fatalf("unexpected error string: %q", pErr.Error())
	}
}
