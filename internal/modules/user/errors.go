package user

import "fmt"

// DomainError is a structured error type for the user module.
// It allows for wrapping underlying errors and providing stable, machine-readable codes.
type DomainError struct {
	// Code is a machine-readable, stable error code (e.g., "USER_NOT_FOUND").
	Code string
	// Message is a human-readable message for developers or logs.
	Message string
	// cause is the underlying error that triggered this one, if any.
	cause error
}

// Error satisfies the standard Go error interface.
// It includes the underlying cause's error message if it exists.
func (e *DomainError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap provides compatibility for Go's errors.Is and errors.As functions,
// allowing access to the underlying error chain.
func (e *DomainError) Unwrap() error {
	return e.cause
}

// WithCause returns a new instance of the DomainError, wrapping the provided cause.
// This is useful for adding context to an error from a lower layer (e.g., repository).
// Example: return user.ErrNotFound.WithCause(err)
func (e *DomainError) WithCause(err error) *DomainError {
	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		cause:   err,
	}
}

// --- Pre-defined Domain Errors ---
// These variables represent specific, known error conditions in the user domain.

var (
	ErrNotFound           = &DomainError{Code: "USER_NOT_FOUND", Message: "user not found"}
	ErrEmailExists        = &DomainError{Code: "USER_EMAIL_EXISTS", Message: "email already exists"}
	ErrInvalidCredentials = &DomainError{Code: "INVALID_CREDENTIALS", Message: "invalid credentials provided"}
	ErrInvalidOTP         = &DomainError{Code: "INVALID_OTP", Message: "invalid or expired one-time password"}
	ErrTermsNotAccepted   = &DomainError{Code: "TERMS_NOT_ACCEPTED", Message: "terms and conditions must be accepted"}
	ErrUnauthorized       = &DomainError{Code: "UNAUTHORIZED", Message: "user is not authorized to perform this action"}
	ErrInvalidResetToken  = &DomainError{Code: "INVALID_RESET_TOKEN", Message: "invalid reset token provided"}
)
