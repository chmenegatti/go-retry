// Package retry provides a simple and composable retry mechanism for Go.
package retry

import (
	"math/rand"
	"time"
)

// Backoff is a function that, given an attempt number (1-indexed), returns
// the duration to wait before the next retry.
type Backoff func(attempt int) time.Duration

// Constant returns a Backoff strategy that always waits the same duration
// between attempts, regardless of the attempt number.
//
// Example:
//
//	r := retry.New().Backoff(retry.Constant(2 * time.Second))
func Constant(d time.Duration) Backoff {
	return func(int) time.Duration {
		return d
	}
}

// Linear returns a Backoff strategy where the wait duration grows linearly
// with the attempt number: delay = attempt * base.
//
// Example:
//
//	r := retry.New().Backoff(retry.Linear(500 * time.Millisecond))
//	// attempt 1 → 500ms, attempt 2 → 1s, attempt 3 → 1.5s
func Linear(base time.Duration) Backoff {
	return func(attempt int) time.Duration {
		return time.Duration(attempt) * base
	}
}

// Exponential returns a Backoff strategy where the wait duration doubles
// with each attempt: delay = base * 2^(attempt-1).
//
// Example:
//
//	r := retry.New().Backoff(retry.Exponential(100 * time.Millisecond))
//	// attempt 1 → 100ms, attempt 2 → 200ms, attempt 3 → 400ms
func Exponential(base time.Duration) Backoff {
	return func(attempt int) time.Duration {
		if attempt <= 0 {
			return base
		}
		return base * time.Duration(1<<uint(attempt-1))
	}
}

// ExponentialJitter returns a Backoff strategy similar to Exponential, but
// adds a random jitter in the range [d/2, d] to avoid the thundering herd
// problem when many clients retry simultaneously.
//
// Example:
//
//	r := retry.New().Backoff(retry.ExponentialJitter(100 * time.Millisecond))
func ExponentialJitter(base time.Duration) Backoff {
	return func(attempt int) time.Duration {
		if attempt <= 0 {
			return base
		}
		d := base * time.Duration(1<<uint(attempt-1))
		half := d / 2
		if half <= 0 {
			return d
		}
		jitter := time.Duration(rand.Int63n(int64(half)))
		return half + jitter
	}
}
