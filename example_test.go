package retry_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	retry "github.com/chmenegatti/go-retry"
)

// ExampleNew demonstrates a basic retry with the default configuration.
func ExampleNew() {
	ctx := context.Background()

	err := retry.New().
		Attempts(3).
		Do(ctx, func() error {
			// Simulate a transient failure.
			return errors.New("service unavailable")
		})

	if err != nil {
		log.Printf("all attempts failed: %v", err)
	}
	// Output:
}

// ExampleDoValue demonstrates returning a typed result from the retry loop.
func ExampleDoValue() {
	ctx := context.Background()

	attempts := 0
	result, err := retry.DoValue(ctx,
		retry.New().Attempts(3).Backoff(retry.Constant(time.Millisecond)),
		func() (string, error) {
			attempts++
			if attempts < 3 {
				return "", errors.New("not ready yet")
			}
			return "ok", nil
		},
	)

	if err != nil {
		log.Printf("failed: %v", err)
		return
	}
	fmt.Println(result)
	// Output: ok
}

// ExampleRetry_RetryIf shows how to skip retrying for known fatal errors.
func ExampleRetry_RetryIf() {
	ctx := context.Background()

	errFatal := errors.New("fatal: invalid credentials")

	err := retry.New().
		Attempts(5).
		RetryIf(func(err error) bool {
			// Only retry transient errors, not authentication failures.
			return !errors.Is(err, errFatal)
		}).
		Do(ctx, func() error {
			return errFatal // stopped after 1 attempt
		})

	fmt.Println(err)
	// Output: fatal: invalid credentials
}

// ExamplePermanent shows how to instantly stop retrying from inside fn.
func ExamplePermanent() {
	ctx := context.Background()

	var ErrBadRequest = errors.New("400 bad request")

	err := retry.New().Attempts(5).Do(ctx, func() error {
		// Wrapping with Permanent signals "don't retry this".
		return retry.Permanent(ErrBadRequest)
	})

	// Unwrap to get the original error.
	fmt.Println(errors.Is(err, ErrBadRequest))
	// Output: true
}

// ExampleRetry_MaxDelay shows capping exponential backoff at a fixed ceiling.
func ExampleRetry_MaxDelay() {
	ctx := context.Background()

	retry.New().
		Attempts(10).
		Backoff(retry.Exponential(100*time.Millisecond)).
		MaxDelay(2*time.Second).
		Do(ctx, func() error {
			return errors.New("fail")
		})
}
