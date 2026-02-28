// Package retry provides a minimal, composable retry mechanism for Go.
// All features are additive and the API is backward compatible.
package retry

import "errors"

// permanentError wraps an error to signal that it should not be retried.
type permanentError struct {
	cause error
}

func (e *permanentError) Error() string { return e.cause.Error() }
func (e *permanentError) Unwrap() error { return e.cause }

// Permanent wraps err so that the retry loop stops immediately without
// further attempts. The original error is preserved and can be retrieved
// with errors.Unwrap or errors.Is/As.
//
// Example:
//
//	Do(ctx, func() error {
//	    resp, err := http.Get(url)
//	    if err != nil {
//	        return err // transient — will retry
//	    }
//	    if resp.StatusCode == 401 {
//	        return retry.Permanent(ErrUnauthorized) // fatal — stop immediately
//	    }
//	    return nil
//	})
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{cause: err}
}

// IsPermanent reports whether err (or any error in its chain) was wrapped
// with Permanent.
func IsPermanent(err error) bool {
	var p *permanentError
	return errors.As(err, &p)
}
