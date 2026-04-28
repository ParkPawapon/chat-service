package domain

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid_input")
	ErrNotFound        = errors.New("not_found")
	ErrForbidden       = errors.New("forbidden")
	ErrGone            = errors.New("gone")
	ErrConflict        = errors.New("conflict")
	ErrNotImplemented  = errors.New("not_implemented")
	ErrDependency      = errors.New("dependency_error")
	ErrUnexpectedState = errors.New("unexpected_state")
)

type AppError struct {
	Kind    error
	Message string
	Cause   error
}

func NewAppError(kind error, message string) *AppError {
	return &AppError{Kind: kind, Message: message}
}

func WrapAppError(kind error, message string, cause error) *AppError {
	return &AppError{Kind: kind, Message: message, Cause: cause}
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "application error"
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Cause != nil {
		return e.Cause
	}
	return e.Kind
}

func (e *AppError) Is(target error) bool {
	return errors.Is(e.Kind, target)
}
