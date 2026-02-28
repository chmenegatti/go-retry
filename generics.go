package retry

import "context"

// DoValue is a generic helper that executes fn using the provided Retry
// configuration and returns both the result and any error. This avoids
// capturing result variables outside the closure when retrying operations
// that produce a value.
//
// Example:
//
//	user, err := retry.DoValue(ctx, retry.New().Attempts(3), func() (*User, error) {
//	    return db.FindUser(id)
//	})
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
