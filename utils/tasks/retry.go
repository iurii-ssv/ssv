package tasks

import "context"

// Retry executes a function until successful, or until it runs out of retries (in which case
// last retry error is returned).
func Retry(fn Fn, retries int) error {
	var err error
	for retries > 0 {
		if err = fn(); err == nil {
			return nil
		}
		retries--
	}
	return err
}

// RetryWithContext executes a function until successful, or until it runs out of retries (in which case
// last retry error is returned).
func RetryWithContext(pctx context.Context, fn Fn, retries int) error {
	var err error
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()
	for retries > 0 && ctx.Err() == nil {
		if err = fn(); err == nil {
			return nil
		}
		retries--
	}
	return err
}
