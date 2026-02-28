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
			return nil // Success
		}

		// If this is the last attempt, don't sleep, just return the error
		if attempt == r.cfg.MaxRetries {
			break
		}

		// Calculate backoff
		nextDelay := ExponentialBackoff(attempt, r.cfg)

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
