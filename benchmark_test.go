package retry

import (
	"context"
	"errors"
	"testing"
)

var errTest = errors.New("test error")

func BenchmarkRetrySuccess(b *testing.B) {

	ctx := context.Background()

	for i := 0; i < b.N; i++ {

		New().Do(ctx, func() error {
			return nil
		})

	}
}

func BenchmarkRetryFailure(b *testing.B) {

	ctx := context.Background()

	for i := 0; i < b.N; i++ {

		New().
			Attempts(3).
			Do(ctx, func() error {
				return errTest
			})

	}
}
