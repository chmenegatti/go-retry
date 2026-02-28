# goretry

`goretry` is a modern, idiomatic, and performant (low allocation) retry library for Go (Golang). It focuses on an exceptional developer experience (DX), offering full `context.Context` awareness, an easy-to-use Functional Options Pattern for configuration, and an implemented Exponential Backoff with Full Jitter.

## Features

- **Context First**: Fully respects `context.Context` cancellation immediately avoiding unnecessary delays.
- **Fluent API**: Highly configurable with the Functional Options Pattern.
- **Generics Support**: Native `Do[T]` function properly returns values avoiding clunky external variable closures.
- **Retry Filtering**: `RetryIf` evaluates which errors should be ignored and bypass retrying.
- **Advanced Backoff Algorithm**: Ships with Exponential Backoff + Full Jitter to prevent Thundering Herd problems.
- **ObservabilityHooks**: Supports an `OnRetry` hook for lifecycle logging/metrics.
- **Thread-Safe**: Safely use the initialized `Retryer` instances across multiple goroutines.
- **Zero Dependencies**: Uses only the Go Standard Library (`stdlib`).

## Installation

```sh
go get github.com/chmenegatti/go-retry
```

## Quick Start

Here is a typical usage example of `goretry` wrapping a flaky HTTP request:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chmenegatti/go-retry"
)

func main() {
	ctx := context.Background()

	// 1. Initialize the Retryer
	retryer := retry.New(
		retry.WithMaxRetries(3),
		retry.WithInitialDelay(100 * time.Millisecond),
		retry.WithMaxDelay(5 * time.Second),
		retry.WithMultiplier(2.0),
		retry.WithJitter(true),
		retry.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
			log.Printf("Attempt %d failed: %v. Retrying in %v...\n", attempt+1, err, nextDelay)
		}),
	)

	// 2. Wrap your risky operation and return a byte slice using the generic Do function
	body, err := retry.Do(ctx, retryer, func(c context.Context) ([]byte, error) {
		req, err := http.NewRequestWithContext(c, http.MethodGet, "https://api.example.com/data", nil)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error: %d", resp.StatusCode) // Will trigger retry
		}

		log.Println("Request succeeded!")
		return io.ReadAll(resp.Body)
	})

	if err != nil {
		log.Fatalf("Operation failed entirely: %v", err)
	}
	log.Printf("Data: %s\n", string(body))
}
```

## Execution Flow

```text
Run(ctx, func)
   |
   +--> Execute Func
   |      |
   |      +-- Success ---> Done
   |      |
   |      +-- Error
   |            |
   |            v
   |       Calculate Backoff (Exponential + Jitter)
   |            |
   |            v
   |       Trigger OnRetry Hook
   |            |
   |            v
   +<----- Wait NextDelay || Select <-ctx.Done()
```

## Configuration Options

- `WithMaxRetries(int)`: Maximum number of retry attempts. Default `3`.
- `WithInitialDelay(time.Duration)`: Starting sleep duration. Default `100ms`.
- `WithMaxDelay(time.Duration)`: Capping sleep duration. Default `10s`.
- `WithMultiplier(float64)`: The exponential growth factor. Default `2.0`.
- `WithJitter(bool)`: Toggles Full Jitter variation. Default `true`.
- `WithOnRetry(func(attempt int, err error, nextDelay time.Duration))`: Hook injected right before sleeping.
- `WithRetryIf(func(err error) bool)`: A filter to intercept errors and decide if it's worth retrying or bypass immediately.
