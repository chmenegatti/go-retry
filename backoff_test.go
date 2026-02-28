package retry

import (
	"math"
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Jitter = false // Disable jitter for deterministic tests
	cfg.InitialDelay = 100 * time.Millisecond
	cfg.Multiplier = 2.0
	cfg.MaxDelay = 5 * time.Second

	tests := []struct {
		attempt       int
		expectedDelay time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
		{5, 3200 * time.Millisecond},
		{6, 5000 * time.Millisecond}, // Capped at MaxDelay
	}

	for _, tc := range tests {
		delay := ExponentialBackoff(cfg)(tc.attempt)
		if delay != tc.expectedDelay {
			t.Errorf("expected attempt %d to have delay %v, got %v", tc.attempt, tc.expectedDelay, delay)
		}
	}
}

func TestExponentialBackoff_Jitter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Jitter = true
	cfg.InitialDelay = 100 * time.Millisecond
	cfg.Multiplier = 2.0
	cfg.MaxDelay = 5 * time.Second

	// Due to randomness, we test bounds
	for attempt := 0; attempt < 10; attempt++ {
		delay := ExponentialBackoff(cfg)(attempt)

		maxExpectedFloat := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))
		maxExpected := time.Duration(maxExpectedFloat)
		if maxExpected > cfg.MaxDelay {
			maxExpected = cfg.MaxDelay
		}

		if delay < 0 || delay > maxExpected {
			t.Errorf("attempt %d: jittered delay %v is outside [0, %v]", attempt, delay, maxExpected)
		}
	}
}

func TestConstantBackoff(t *testing.T) {
	delay := 5 * time.Second
	backoff := ConstantBackoff(delay)

	for attempt := 0; attempt < 5; attempt++ {
		if got := backoff(attempt); got != delay {
			t.Errorf("expected attempt %d to have constant delay %v, got %v", attempt, delay, got)
		}
	}
}
