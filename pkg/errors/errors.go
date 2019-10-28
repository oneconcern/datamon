// Package errors augments the standard errors
// provided by fmt (https://golang.org/src/fmt/errors.go)
// with a Wrap() method to wrap errors without resorting
// to fmt.Errorf("%w", err).
package errors

import (
	stderr "errors"
)

var _ error = New("")

// New Error
func New(msg string) *Error {
	return &Error{msg: msg}
}

// Error augments the standard error interface with a Wrap method.
//
// The main difference with github.com/pkg/errors is that we are wrapping
// errors from errors, not from text.
type Error struct {
	msg string
	err error
}

// Error message
func (e *Error) Error() string {
	return e.msg
}

// Unwrap nested error
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Wrap a nested error
func (e *Error) Wrap(err error) *Error {
	e.err = err
	return e
}

// Is of some error type?
func (e *Error) Is(target error) bool {
	return e == target || e.err == target
}

// As finds the first error in err's chain that matches target, and if so, sets target to that error value and returns true.
// (a shortcut to standard lib errors.As)
func As(err error, target interface{}) bool {
	return stderr.As(err, target)
}

// Is reports whether any error in err's chain matches target
// (a shortcut to standard lib errors.As)
func Is(err, target error) bool {
	return stderr.Is(err, target)
}
