// Package errors augments the standard errors
// provided by fmt (https://golang.org/src/fmt/errors.go)
// with a Wrap() method to wrap errors without resorting
// to fmt.Errorf("%w", err).
package errors

import (
	stderr "errors"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	return e.String()
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
	if err == nil {
		return e
	}
	if e == nil {
		return &Error{msg: err.Error()}
	}
	e.err = err
	return e
}

// WrapMessage wraps a nested message error as with fmt.Errorf()
func (e *Error) WrapMessage(msg string, args ...interface{}) *Error {
	wrapped := fmt.Errorf(msg, args...)
	return e.Wrap(wrapped)
}

// WithLog logs the error with some extra fields, which are part of the log but
// not of the returned error.
func (e *Error) WithLog(logger *zap.Logger, args ...zapcore.Field) *Error {
	if logger != nil {
		logger.Error(e.String(), args...)
	}
	return e
}

// WrapWithLog logs with some extra fields and wraps a nested error.
// TODO(fred): marshal extra args into wrapped err...
func (e *Error) WrapWithLog(logger *zap.Logger, err error, args ...zapcore.Field) *Error {
	if logger != nil {
		extras := append([]zapcore.Field{zap.Error(err)}, args...)
		logger.Error(e.String(), extras...)
	}
	return e.Wrap(err)
}

// Is of some error type?
func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	if e == target {
		return true
	}
	if e.err != nil {
		if thisErr, ok := (e.err).(*Error); ok {
			return thisErr.Is(target)
		}
	}
	return false
}

// String displays the stack of errors
func (e *Error) String() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return e.msg
	}
	if stringer, ok := (e.err).(fmt.Stringer); ok {
		return e.msg + ": " + stringer.String()
	}
	return e.msg + ": " + e.err.Error()
}

// As finds the first error in err's chain that matches target, and if so, sets target to that error value and returns true.
// (a shortcut to standard lib errors.As)
func As(err error, target interface{}) bool {
	return stderr.As(err, target)
}

// Is reports whether any error in err's chain matches target
// (a shortcut to standard lib errors.As)
func Is(err, target error) bool {
	if err == nil {
		return target == nil
	}
	if thisErr, ok := err.(*Error); ok {
		return thisErr.Is(target)
	}
	return stderr.Is(err, target)
}
