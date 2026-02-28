package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/chmenegatti/go-retry"
)

func main() {
	// Let's create a context with timeout to demonstrate context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize the Retryer with custom options
	retryer := retry.New(
		retry.WithMaxRetries(5),
		retry.WithInitialDelay(500*time.Millisecond),
		retry.WithMaxDelay(3*time.Second),
		retry.WithMultiplier(2.0),
		retry.WithJitter(true),
		retry.WithBackoff(retry.ConstantBackoff(2*time.Second)), // Overriding default Exponential
		retry.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			log.Printf("[Retry Hook] Attempt %d failed with error: %v. Retrying in %v...\n", attempt+1, err, nextDelay)
		}),
		retry.WithOnSuccess(func(attempt int) {
			log.Printf("✅ Success! Recovered after %d attempts.\n", attempt)
		}),
		retry.WithOnDrop(func(attempt int, err error) {
			log.Printf("❌ Dropped! Too many failures (%d). Final Error: %v\n", attempt, err)
		}),
	)

	// A sample URL that might fail or succeed
	// In a real scenario, this could be a flaky endpoint
	url := "https://httpbin.org/status/500,200"

	log.Println("Starting HTTP GET request...")

	// Execute the operation wrapped in the retryer to return a byte slice
	body, err := retry.Do(ctx, retryer, func(ctx context.Context) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err // Network error, triggering a retry
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			// Simulate a recoverable server error
			return nil, fmt.Errorf("server returned status code: %d", resp.StatusCode)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("client error: %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	})

	if err != nil {
		log.Fatalf("Operation failed completely after retries: %v\n", err)
	} else {
		log.Printf("Operation completed successfully! Response: %s\n", string(body[:min(len(body), 50)]))
	}
}
