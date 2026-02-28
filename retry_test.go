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

	attempts := 0

	cancel()

	err := New().
		Attempts(5).
		Do(ctx, func() error {

			attempts++
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

	duration := time.Since(start)

	if duration < 200*time.Millisecond {
		t.Fatal("backoff not applied")
	}
}

func TestDoValue(t *testing.T) {

	ctx := context.Background()

	attempts := 0

	result, err := DoValue(
		ctx,
		New().Attempts(5),
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
		t.Fatal("unexpected result")
	}

	if attempts != 3 {
		t.Fatal("unexpected attempts")
	}
}
