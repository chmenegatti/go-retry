package retry

import (
	"context"
	"time"
)

// Retry holds the configuration for a retry operation.
// Use New() to construct a Retry and chain the builder methods
// (Attempts, Backoff, OnRetry) before calling Do.
type Retry struct {
	attempts int
	backoff  Backoff
	onRetry  func(int, error)
}

// New creates a new Retry with sensible defaults:
// 3 attempts with a 100ms constant backoff.
func New() *Retry {
	return &Retry{
		attempts: 3,
		backoff:  Constant(100 * time.Millisecond),
	}
}

// Attempts sets the total number of times the function will be called
// (including the first attempt). Values less than 1 are treated as 1.
//
// Example:
//
//	retry.New().Attempts(5).Do(ctx, fn)
func (r *Retry) Attempts(n int) *Retry {
	if n < 1 {
		n = 1
	}
	r.attempts = n
	return r
}

// Backoff sets the strategy used to calculate the wait duration between
// attempts. Use the built-in strategies: Constant, Linear, Exponential,
// or ExponentialJitter. You may also provide a custom Backoff function.
//
// Example:
//
//	retry.New().Backoff(retry.ExponentialJitter(200 * time.Millisecond)).Do(ctx, fn)
func (r *Retry) Backoff(b Backoff) *Retry {
	r.backoff = b
	return r
}

// OnRetry sets a callback that is invoked before each retry sleep.
// The callback receives the current attempt number (1-indexed) and the
// error returned by the last execution. It is NOT called on the final failure.
//
// Example:
//
//	retry.New().OnRetry(func(attempt int, err error) {
//	    log.Printf("attempt %d failed: %v", attempt, err)
//	}).Do(ctx, fn)
func (r *Retry) OnRetry(fn func(int, error)) *Retry {
	r.onRetry = fn
	return r
}

// Do executes fn up to the configured number of Attempts. Between each
// failed attempt, it waits for the duration returned by the Backoff
// strategy or until ctx is cancelled — whichever comes first.
//
// Do returns nil on the first successful execution, or the last error
// returned by fn if all attempts fail. If the context is cancelled while
// waiting between attempts, Do returns ctx.Err() immediately.
//
// Note: fn is always called at least once, even if ctx is already cancelled,
// to give the caller a chance to detect the context state inside fn.
func (r *Retry) Do(ctx context.Context, fn func() error) error {
	var err error

	for attempt := 1; attempt <= r.attempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		// No sleep after the final attempt.
		if attempt == r.attempts {
			break
		}

		if r.onRetry != nil {
			r.onRetry(attempt, err)
		}

		delay := r.backoff(attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return err
}
