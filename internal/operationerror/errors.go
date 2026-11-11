// Package operationerror preserves retry decisions when an operation has cleanup.
package operationerror

import "errors"

type policyError struct {
	err   error
	retry bool
}

func (e *policyError) Error() string     { return e.err.Error() }
func (e *policyError) Unwrap() error     { return e.err }
func (e *policyError) SafeToRetry() bool { return e.retry }

// WithRetry attaches the retry decision for the complete operation.
func WithRetry(err error, retry bool) error {
	// Keep ordinary server errors intact when they already deny retry. Only
	// errors that could authorize retry need a wrapper to override that policy.
	if err == nil || (!retry && !safeToRetry(err)) {
		return err
	}
	return &policyError{err: err, retry: retry}
}

// Join retains cleanup diagnostics without replacing the operation's policy.
// A cleanup failure following a successful operation cannot authorize retry.
func Join(operation, cleanup error) error {
	if cleanup == nil {
		return operation
	}
	return WithRetry(errors.Join(operation, cleanup), safeToRetry(operation))
}

func safeToRetry(err error) bool {
	var classified interface{ SafeToRetry() bool }
	return errors.As(err, &classified) && classified.SafeToRetry()
}
