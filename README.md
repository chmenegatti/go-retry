<div align="center">
  <img src="img/logo.jpg" alt="goretry logo" width="300" />
  <h1>🚀 goretry</h1>
  <p><b>A modern, robust, and DX-focused retry library for Go.</b></p>
  
  [![Go Reference](https://pkg.go.dev/badge/github.com/chmenegatti/go-retry.svg)](https://pkg.go.dev/github.com/chmenegatti/go-retry)
  [![Go Report Card](https://goreportcard.com/badge/github.com/chmenegatti/go-retry)](https://goreportcard.com/report/github.com/chmenegatti/go-retry)
  [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
</div>

<br/>

`goretry` was built from the ground up to solve the headaches of retrying operations in Go. It embraces modern Go paradigms like **Generics** and **context.Context**, shipping with enterprise-grade defaults like **Exponential Backoff** and **Full Jitter** to prevent melting your servers under load.

All of this with **Zero Dependencies**, fully relying on the Go standard library.

## ✨ Why `goretry`?

* 🎯 **Context First:** Cancellation (`ctx.Done()`) is respected immediately. No waiting for a 10-second sleep to finish before your goroutine can exit.
* 🧬 **Generics Support:** Read your returned objects straight from the retry loop. No more awkward variable closures.
* 🛑 **Smart Filtering:** Ignore specific errors (e.g., `400 Bad Request`) using `WithRetryIf` to save computing time.
* 📈 **Sensible Defaults:** Exponential backoff with Full Jitter activated by default, preventing [Thundering Herds](https://en.wikipedia.org/wiki/Thundering_herd_problem). **Pluggable** with `WithBackoff`.
* 🪝 **Observability Ready:** Use hooks like `OnRetry`, `OnSuccess` and `OnDrop` to log or ship metrics precisely when failures and recoveries happen.
* 🪶 **Featherweight:** 0 external dependencies. `stdlib` only!

---

## 📦 Installation

```bash
go get github.com/chmenegatti/go-retry
```

*(Requires Go 1.18+ for Generics support)*

---

## 💻 Quick Start

### The Modern Way (Returning Values with Generics)

No more capturing variables outside the closure. Use `retry.Do` to smoothly return your fetched data:

```go
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chmenegatti/go-retry"
)

func main() {
	ctx := context.Background()

	// 1. Setup your rules
	retryer := retry.New(
		retry.WithMaxRetries(3),
		retry.WithInitialDelay(100 * time.Millisecond),
	)

	// 2. Wrap the risky operation
	body, err := retry.Do(ctx, retryer, func(c context.Context) ([]byte, error) {
		resp, err := http.Get("https://api.example.com/data")
		if err != nil {
			return nil, err // Network failure -> retry!
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error: %d", resp.StatusCode) // Server error -> retry!
		}

		// Success!
		return io.ReadAll(resp.Body) 
	})

	if err != nil {
		fmt.Printf("Dead after all retries: %v\n", err)
		return
	}
	
	fmt.Printf("Successfully fetched %d bytes!\n", len(body))
}
```

---

## 📖 Advanced Usage & Examples

### Using `RetryIf` to avoid retrying fatal errors
Sometimes, an error means *"Game Over"* (e.g., `401 Unauthorized` or `sql.ErrNoRows`). Retrying won't change the outcome. Use `RetryIf` to abort early:

```go
retryer := retry.New(
	retry.WithMaxRetries(5),
	retry.WithRetryIf(func(err error) bool {
		// Do NOT retry if it's a 4xx Client Error
		if apiErr, ok := err.(APIError); ok && apiErr.Code >= 400 && apiErr.Code < 500 {
			return false // Abort!
		}
		// Otherwise, keep retrying
		return true
	}),
)
```

### Hooking into the lifecycle for Observability
Need to plug this into Prometheus, DataDog, or just `log`? Use the `OnRetry` hook. It triggers right *before* the sleep occurs.

```go
retryer := retry.New(
	retry.WithMaxRetries(10),
	retry.WithOnRetry(func(attempt int, err error, nextDelay time.Duration) {
		log.Printf("⚠️ Attempt %d failed: %v. Resting for %v...", attempt+1, err, nextDelay)
		// metrics.IncrementCounter("retry_events")
	}),
)
```

### Pure Error Handling (Without Generics)
If your function just does side-effects (like deleting a file or writing to a socket) and returns `error`, just use `.Run()`:

```go
err := retryer.Run(ctx, func(c context.Context) error {
	return db.ExecContext(c, "DELETE FROM users WHERE id = ?", userID)
})
```

---

## ⚙️ Configuration Reference (Options)

By default, calling `retry.New()` gives you: 3 retries, starting at 100ms, capping at 10s, utilizing `x2` multiplications with jitter ON.

You can tweak everything using the Functional Options:

| Option | Default | Description |
|--------|---------|-------------|
| `WithMaxRetries(int)` | `3` | Maximum number of **retries** before returning the error. |
| `WithInitialDelay(Duration)` | `100ms` | The base backoff delay. |
| `WithMultiplier(float64)` | `2.0` | Exponential factor. e.g: `100ms` -> `200ms` -> `400ms`. |
| `WithMaxDelay(Duration)` | `10s` | Hard cap to prevent sleeps from taking hours. |
| `WithJitter(bool)` | `true` | Introduces randomness to spread out retries. |
| `WithBackoff(func)`| `ExponentialBackoff` | The algorithm used to space out sleep times. Try `ConstantBackoff(2*time.Second)` or roll your own. |
| `WithOnRetry(func(...))` | `nil` | Lifecycle hook executed on failures. |
| `WithOnSuccess(func(int))` | `nil` | Lifecycle hook executed when a previous failure eventually succeeds. |
| `WithOnDrop(func(...))` | `nil` | Lifecycle hook executed when it gives up entirely due to MaxRetries. |
| `WithRetryIf(func(err) bool)` | `nil` | A filter function to conditionally cancel retry loops. |

---

## 🧠 Under the Hood: The Retry Execution Flow

Here is how `goretry` processes your requests gracefully:

```mermaid
flowchart TD
    Start([Start retry.Run/Do]) --> ExecFunc[Execute User Function]
    ExecFunc --> CheckValid{Success?}
    
    CheckValid -->|Yes| Success([Return Result])
    CheckValid -->|No| CheckFilter{RetryIf filter passes?}
    
    CheckFilter -->|No| MaxReached([Return Error Immediately])
    CheckFilter -->|Yes| CheckMax{Max attempts reached?}
    
    CheckMax -->|Yes| MaxReached
    CheckMax -->|No| CalcBackoff[Calculate Backoff \n Exponential + Jitter]
    
    CalcBackoff --> TriggerHook[Execute OnRetry Hook]
    TriggerHook --> SleepPhase[Select Channel]
    
    SleepPhase -->|Timer fired| ExecFunc
    SleepPhase -->|ctx.Done() triggered| ContextCancelled([Return Context Error])
```

---

## 🤝 Contributing

Pull requests are immensely welcome! 

1. Fork it
2. Create your feature branch (`git checkout -b feature/fooBar`)
3. Commit your changes (`git commit -am 'Add some fooBar'`)
4. Run tests (`go test -v -race ./...`)
5. Push to the branch (`git push origin feature/fooBar`)
6. Create a new Pull Request

## 📄 License

MIT. See `LICENSE` for details.
