package retry

import (
	"context"
	"time"
)

// timerAbstract is an interface for time operations to allow mocking in tests.
type timerAbstract interface {
	After(d time.Duration) <-chan time.Time
}

// realTimer implements timerAbstract using the standard time package.
type realTimer struct{}

func (rt *realTimer) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// Retryer is responsible for executing functions with retry logic.
type Retryer struct {
	cfg   *Config
	timer timerAbstract
}

// New creates a new Retryer with the provided options.
func New(opts ...Option) *Retryer {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return &Retryer{
		cfg:   cfg,
		timer: &realTimer{},
	}
}

// Run executes the provided function. If the function returns an error, it will
// be retried according to the Retryer's configuration.
// It respects the context cancellation immediately.
func (r *Retryer) Run(ctx context.Context, fn func(ctx context.Context) error) error {
	var err error

	// Pre-check context
	if ctx.Err() != nil {
		return ctx.Err()
	}

	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		// Run the user function
		err = fn(ctx)
		if err == nil {
			if attempt > 0 && r.cfg.OnSuccess != nil {
				r.cfg.OnSuccess(attempt)
			}
			return nil // Success
		}

		// Check if we should retry this error
		if r.cfg.RetryIf != nil && !r.cfg.RetryIf(err) {
			return err
		}

		// If this is the last attempt, don't sleep, just return the error
		if attempt == r.cfg.MaxRetries {
			if r.cfg.OnDrop != nil {
				r.cfg.OnDrop(attempt, err)
			}
			break
		}

		// Calculate backoff
		var nextDelay time.Duration
		if r.cfg.Backoff != nil {
			nextDelay = r.cfg.Backoff(attempt)
		} else {
			// Fallback if somehow Backoff is nil, although DefaultConfig sets it
			nextDelay = ExponentialBackoff(r.cfg)(attempt)
		}

		// Call the OnRetry hook
		if r.cfg.OnRetry != nil {
			r.cfg.OnRetry(attempt, err, nextDelay)
		}

		// Wait for either the delay to pass or the context to be cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.timer.After(nextDelay):
			// Proceed to the next attempt
		}
	}

	return err
}

// Do is a generic package-level helper that executes a function returning a value of type T
// and an error. It wraps the execution in the provided Retryer.
func Do[T any](ctx context.Context, r *Retryer, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := r.Run(ctx, func(c context.Context) error {
		v, err := fn(c)
		if err != nil {
			return err
		}
		result = v
		return nil
	})
	return result, err
}
