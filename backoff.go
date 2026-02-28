package retry

import (
	"math/rand"
	"time"
)

type Backoff func(attempt int) time.Duration

func Constant(d time.Duration) Backoff {
	return func(int) time.Duration {
		return d
	}
}

func Linear(base time.Duration) Backoff {
	return func(attempt int) time.Duration {
		return time.Duration(attempt) * base
	}
}

func Exponential(base time.Duration) Backoff {
	return func(attempt int) time.Duration {
		return base * time.Duration(1<<uint(attempt-1))
	}
}

func ExponentialJitter(base time.Duration) Backoff {

	return func(attempt int) time.Duration {

		d := base * time.Duration(1<<uint(attempt-1))

		jitter := time.Duration(rand.Int63n(int64(d / 2)))

		return d/2 + jitter
	}
}
