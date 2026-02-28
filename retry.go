package retry

import (
	"context"
	"time"
)

type Retry struct {
	attempts int
	backoff  Backoff
	onRetry  func(int, error)
}

func New() *Retry {
	return &Retry{
		attempts: 3,
		backoff:  Constant(100 * time.Millisecond),
	}
}

func (r *Retry) Attempts(n int) *Retry {
	if n < 1 {
		n = 1
	}

	r.attempts = n
	return r
}

func (r *Retry) Backoff(b Backoff) *Retry {
	r.backoff = b
	return r
}

func (r *Retry) OnRetry(fn func(int, error)) *Retry {
	r.onRetry = fn
	return r
}

func (r *Retry) Do(ctx context.Context, fn func() error) error {

	var err error

	for attempt := 1; attempt <= r.attempts; attempt++ {

		err = fn()
		if err == nil {
			return nil
		}

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
