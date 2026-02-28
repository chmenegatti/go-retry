package retry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffFunc defines the signature for a backoff strategy.
// attempt is the number of retries already performed (0-indexed).
type BackoffFunc func(attempt int, cfg *Config) time.Duration

// ExponentialBackoff calculates the backoff delay with exponential growth
// and optional full jitter.
func ExponentialBackoff(attempt int, cfg *Config) time.Duration {
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

	// Full Jitter: picking a uniform random duration between 0 and delay
	if cfg.Jitter {
		// math/rand is not thread-safe and usually requires a lock for a custom source,
		// but since Go 1.20 rand handles global seeding automatically and thread-safety
		// improvements have been introduced, wait, global math/rand is thread-safe.
		// Let's use it or instantiate a local source.
		// Actually, rand.Int64n is thread-safe on the global source.
		// Since delay must be > 0 for rand.Int63n
		if delay > 0 {
			delay = time.Duration(rand.Int63n(int64(delay)))
		}
	}

	return delay
}
