package retry

import "context"

func DoValue[T any](ctx context.Context, r *Retry, fn func() (T, error)) (T, error) {

	var result T

	err := r.Do(ctx, func() error {

		v, err := fn()
		if err != nil {
			return err
		}

		result = v
		return nil

	})

	return result, err
}
