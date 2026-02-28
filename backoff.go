package retry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffAlgorithm defines the signature for a backoff strategy.
// attempt is the number of retries already performed (0-indexed).
type BackoffAlgorithm func(attempt int) time.Duration

// ExponentialBackoff returns a BackoffAlgorithm that calculates the backoff delay
// with exponential growth and optional full jitter based on the provided config.
func ExponentialBackoff(cfg *Config) BackoffAlgorithm {
	return func(attempt int) time.Duration {
		if cfg.InitialDelay <= 0 {
			return 0
		}

		// Exponential part: initial * (multiplier ^ attempt)
		delayF := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))

		// Cap at MaxDelay
		if cfg.MaxDelay > 0 && delayF > float64(cfg.MaxDelay) {
			delayF = float64(cfg.MaxDelay)
		}

		delay := time.Duration(delayF)

		// Full Jitter
		if cfg.Jitter {
			if delay > 0 {
				delay = time.Duration(rand.Int63n(int64(delay)))
			}
		}

		return delay
	}
}

// ConstantBackoff returns a BackoffAlgorithm that always returns the same delay.
func ConstantBackoff(delay time.Duration) BackoffAlgorithm {
	return func(attempt int) time.Duration {
		return delay
	}
}
