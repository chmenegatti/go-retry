package retry

import "time"

// Config holds the configuration for a retry operation.
type Config struct {
	// MaxRetries is the maximum number of retries before giving up.
	// A value of 0 means do not retry. A negative value means retry indefinitely (if supported, although usually we limit).
	// For this lib, let's use 0 as "no retry", and N for N retries.
	MaxRetries int

	// InitialDelay is the starting delay for the first retry.
	InitialDelay time.Duration

	// MaxDelay is the maximum possible delay between retries.
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases each attempt.
	// For example, 2.0 means the delay doubles each time.
	Multiplier float64

	// Jitter enables "Full Jitter" to prevent thundering herd problems.
	Jitter bool

	// OnRetry is a hook called before a retry occurs.
	// attempt is zero-indexed based on the retry attempts (0 means first retry).
	OnRetry func(attempt int, err error, nextDelay time.Duration)
}

// Option represents a functional option for configuring a Retryer.
type Option func(*Config)

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Jitter:       true, // Full jitter is recommended best practice
		OnRetry:      func(attempt int, err error, nextDelay time.Duration) {},
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithInitialDelay sets the initial backoff delay.
func WithInitialDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.InitialDelay = delay
	}
}

// WithMaxDelay sets the maximum backoff delay.
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(c *Config) {
		c.MaxDelay = maxDelay
	}
}

// WithMultiplier sets the exponential backoff multiplier.
func WithMultiplier(multiplier float64) Option {
	return func(c *Config) {
		c.Multiplier = multiplier
	}
}

// WithJitter enables or disables full jitter.
func WithJitter(jitter bool) Option {
	return func(c *Config) {
		c.Jitter = jitter
	}
}

// WithOnRetry sets a callback to be executed before each retry attempt.
func WithOnRetry(onRetry func(attempt int, err error, nextDelay time.Duration)) Option {
	return func(c *Config) {
		if onRetry != nil {
			c.OnRetry = onRetry
		}
	}
}
