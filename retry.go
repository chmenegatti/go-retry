package retry

import (
	"context"
	"time"
)

// Retry holds the configuration for a retry operation.
// Use New() to construct a Retry and chain builder methods before calling Do.
type Retry struct {
	attempts int
	backoff  Backoff
	maxDelay time.Duration
	retryIf  func(error) bool
	onRetry  func(int, error)
}

// New creates a new Retry with sensible defaults:
// 3 total attempts with a 100ms constant backoff.
func New() *Retry {
	return &Retry{
		attempts: 3,
		backoff:  Constant(100 * time.Millisecond),
	}
}

// Attempts sets the total number of times fn will be called (including the
// first attempt). Values less than 1 are treated as 1.
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

// Backoff sets the strategy used to compute the wait duration between
// attempts. Use the built-in strategies (Constant, Linear, Exponential,
// ExponentialJitter) or provide a custom func(attempt int) time.Duration.
//
// Example:
//
//	retry.New().Backoff(retry.ExponentialJitter(200 * time.Millisecond))
func (r *Retry) Backoff(b Backoff) *Retry {
	r.backoff = b
	return r
}

// MaxDelay caps the delay returned by the backoff strategy. Any computed
// delay exceeding MaxDelay will be clamped to MaxDelay. This is especially
// useful with exponential backoffs to prevent excessively long waits.
//
// Example:
//
//	retry.New().
//	    Backoff(retry.Exponential(100*time.Millisecond)).
//	    MaxDelay(5 * time.Second)
func (r *Retry) MaxDelay(d time.Duration) *Retry {
	r.maxDelay = d
	return r
}

// RetryIf sets a predicate that controls whether a given error should trigger
// a retry. If the predicate returns false, Do returns the error immediately
// without further attempts. If unset, all errors are retried up to Attempts.
//
// Note: errors wrapped with Permanent always stop the loop, regardless of RetryIf.
//
// Example:
//
//	retry.New().RetryIf(func(err error) bool {
//	    return errors.Is(err, io.ErrUnexpectedEOF)
//	})
func (r *Retry) RetryIf(fn func(error) bool) *Retry {
	r.retryIf = fn
	return r
}

// OnRetry registers a callback invoked before each sleep between attempts.
// The callback receives the current attempt number (1-indexed) and the error
// from the last execution. It is NOT called after the final failing attempt.
//
// Example:
//
//	retry.New().OnRetry(func(attempt int, err error) {
//	    log.Printf("attempt %d failed: %v", attempt, err)
//	})
func (r *Retry) OnRetry(fn func(int, error)) *Retry {
	r.onRetry = fn
	return r
}

// Do executes fn up to the configured number of Attempts. Between each
// failed attempt, Do sleeps for the duration returned by the Backoff
// strategy (capped by MaxDelay if set), or until ctx is cancelled —
// whichever comes first.
//
// Do returns nil on the first successful execution of fn.
// Do returns the last error from fn if all attempts fail.
// Do returns ctx.Err() if the context is cancelled while waiting.
// Do returns the error immediately (without further retries) if:
//   - the error was wrapped with Permanent, or
//   - a RetryIf predicate returns false for that error.
func (r *Retry) Do(ctx context.Context, fn func() error) error {
	var err error

	for attempt := 1; attempt <= r.attempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Stop immediately for permanent errors.
		if IsPermanent(err) {
			return err
		}

		// Stop if the caller's predicate says don't retry.
		if r.retryIf != nil && !r.retryIf(err) {
			return err
		}

		// No sleep after the final attempt.
		if attempt == r.attempts {
			break
		}

		if r.onRetry != nil {
			r.onRetry(attempt, err)
		}

		delay := r.backoff(attempt)
		if r.maxDelay > 0 && delay > r.maxDelay {
			delay = r.maxDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return err
}
