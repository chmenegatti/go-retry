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
		retry.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			log.Printf("[Retry Hook] Attempt %d failed with error: %v. Retrying in %v...\n", attempt+1, err, nextDelay)
		}),
	)

	// A sample URL that might fail or succeed
	// In a real scenario, this could be a flaky endpoint
	url := "https://httpbin.org/status/500,200"

	log.Println("Starting HTTP GET request...")

	// Execute the operation wrapped in the retryer
	err := retryer.Run(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err // Network error, triggering a retry
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			// Simulate a recoverable server error
			return fmt.Errorf("server returned status code: %d", resp.StatusCode)
		}
		if resp.StatusCode >= 400 {
			// Usually we don't retry 4xx errors as they are client errors
			// For demonstration, let's say we don't retry by returning a special wrapped error,
			// or we can just return it and let the retryer retry anyway if we want.
			// Let's keep it simple here.
			return fmt.Errorf("client error: %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		log.Printf("Request successful! Response: %s\n", string(body[:min(len(body), 50)]))
		return nil
	})

	if err != nil {
		log.Fatalf("Operation failed completely after retries: %v\n", err)
	} else {
		log.Println("Operation completed successfully!")
	}
}
