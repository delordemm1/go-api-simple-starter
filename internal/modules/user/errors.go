package user

import (
	"fmt"
	"net/http"
)

// DomainError is a structured, self-describing domain error used across the user module.
// It carries HTTP/RFC7807-friendly metadata so a shared formatter can convert any domain
// error into a Problem response without enumerating error types.
type DomainError struct {
	// Code is a stable, machine-readable business code (e.g., "ErrInvalidResetToken").
	Code string

	// HTTPStatus is the HTTP status suggested for this error (e.g., 400, 401, 404, 409, 500).
	HTTPStatus int

	// Title is a short human summary; if empty the formatter will default to StatusText(HTTPStatus).
	Title string

	// Message is a human-readable message primarily for logs. When Detail is empty,
	// this is used as the public detail.
	Message string

	// Detail is a user-friendly, safe explanation for clients. If empty, Message is used.
	Detail string

	// TypeURI is an RFC7807 type URI for documentation, e.g., "urn:problem:user/err-invalid-reset-token".
	TypeURI string

	// Context is an optional extension payload for clients (e.g., validation fields map).
	Context any

	// cause is the underlying error that triggered this one, if any.
	cause error
}

// Error satisfies the standard Go error interface.
// It includes the underlying cause's error message if it exists.
func (e *DomainError) Error() string {
	msg := e.Detail
	if msg == "" {
		msg = e.Message
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

	// Unwrap provides compatibility for Go's errors.Is and errors.As functions,
	// allowing access to the underlying error chain.
	func (e *DomainError) Unwrap() error {
		return e.cause
	}

	// Is enables errors.Is comparisons based on the stable Code rather than pointer identity.
	// This ensures copies created via WithCause match their sentinel counterpart (e.g., ErrNotFound).
	func (e *DomainError) Is(target error) bool {
		t, ok := target.(*DomainError)
		if !ok {
			return false
		}
		return e.Code == t.Code
	}

// WithCause returns a new instance of the DomainError, wrapping the provided cause.
func (e *DomainError) WithCause(err error) *DomainError {
	if err == nil {
		return e
	}
	cp := *e
	cp.cause = err
	return &cp
}

// WithDetail sets a public-friendly detail message for clients.
func (e *DomainError) WithDetail(detail string) *DomainError {
	cp := *e
	cp.Detail = detail
	return &cp
}

// WithType sets the RFC7807 type URI for this error.
func (e *DomainError) WithType(uri string) *DomainError {
	cp := *e
	cp.TypeURI = uri
	return &cp
}

// WithContext attaches an extension payload for clients (e.g., validation fields).
func (e *DomainError) WithContext(ctx any) *DomainError {
	cp := *e
	cp.Context = ctx
	return &cp
}

// --- RFC7807 mapping accessors (satisfy httpx.DomainProblem) ---

func (e *DomainError) ProblemCode() string     { return e.Code }
func (e *DomainError) ProblemStatus() int      { if e.HTTPStatus == 0 { return http.StatusInternalServerError }; return e.HTTPStatus }
func (e *DomainError) ProblemTitle() string    { return e.Title }
func (e *DomainError) ProblemDetail() string   { if e.Detail != "" { return e.Detail }; return e.Message }
func (e *DomainError) ProblemTypeURI() string  { return e.TypeURI }
func (e *DomainError) ProblemContext() any     { return e.Context }

// --- Pre-defined Domain Errors ---
// These variables represent specific, known error conditions in the user domain.

var (
	// Resource & identity
	ErrNotFound = &DomainError{
		Code:       "ErrNotFound",
		HTTPStatus: http.StatusNotFound,
		Title:      "Not Found",
		Message:    "user not found",
		TypeURI:    "urn:problem:user/err-not-found",
	}

	ErrUnauthorized = &DomainError{
		Code:       "ErrUnauthorized",
		HTTPStatus: http.StatusUnauthorized,
		Title:      "Unauthorized",
		Message:    "user is not authorized to perform this action",
		TypeURI:    "urn:problem:user/err-unauthorized",
	}

	// Auth & credentials
	ErrInvalidCredentials = &DomainError{
		Code:       "ErrInvalidCredentials",
		HTTPStatus: http.StatusUnauthorized,
		Title:      "Unauthorized",
		Message:    "invalid email or password",
		TypeURI:    "urn:problem:user/err-invalid-credentials",
	}

	// Email verification gating
	ErrEmailNotVerified = &DomainError{
		Code:       "ErrEmailNotVerified",
		HTTPStatus: http.StatusForbidden,
		Title:      "Forbidden",
		Message:    "email not verified",
		TypeURI:    "urn:problem:user/err-email-not-verified",
	}

	ErrInvalidOTP = &DomainError{
		Code:       "ErrInvalidOTP",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "invalid or expired one-time password",
		TypeURI:    "urn:problem:user/err-invalid-otp",
	}

	// Abuse controls for code sending/verification
	ErrResendTooSoon = &DomainError{
		Code:       "ErrResendTooSoon",
		HTTPStatus: http.StatusTooManyRequests,
		Title:      "Too Many Requests",
		Message:    "please wait before requesting another code",
		TypeURI:    "urn:problem:user/err-resend-too-soon",
	}

	ErrTooManyAttempts = &DomainError{
		Code:       "ErrTooManyAttempts",
		HTTPStatus: http.StatusTooManyRequests,
		Title:      "Too Many Requests",
		Message:    "too many invalid attempts",
		TypeURI:    "urn:problem:user/err-too-many-attempts",
	}

	ErrInvalidResetToken = &DomainError{
		Code:       "ErrInvalidResetToken",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "the provided token is invalid or has expired",
		TypeURI:    "urn:problem:user/err-invalid-reset-token",
	}

	// Registration
	ErrEmailExists = &DomainError{
		Code:       "ErrEmailExists",
		HTTPStatus: http.StatusConflict,
		Title:      "Conflict",
		Message:    "a user with this email already exists",
		TypeURI:    "urn:problem:user/err-email-exists",
	}

	ErrTermsNotAccepted = &DomainError{
		Code:       "ErrTermsNotAccepted",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "terms and conditions must be accepted",
		TypeURI:    "urn:problem:user/err-terms-not-accepted",
	}

	// OAuth
	ErrUnsupportedOAuthProvider = &DomainError{
		Code:       "ErrUnsupportedOAuthProvider",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "unsupported oauth provider",
		TypeURI:    "urn:problem:user/err-unsupported-oauth-provider",
	}

	ErrOAuthStateInvalid = &DomainError{
		Code:       "ErrOAuthStateInvalid",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "invalid oauth state",
		TypeURI:    "urn:problem:user/err-oauth-state-invalid",
	}

	ErrOAuthStateExpired = &DomainError{
		Code:       "ErrOAuthStateExpired",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "oauth state has expired",
		TypeURI:    "urn:problem:user/err-oauth-state-expired",
	}

	ErrOAuthExchangeFailed = &DomainError{
		Code:       "ErrOAuthExchangeFailed",
		HTTPStatus: http.StatusUnauthorized,
		Title:      "Unauthorized",
		Message:    "oauth authentication failed",
		TypeURI:    "urn:problem:user/err-oauth-exchange-failed",
	}

	ErrOAuthEmailMissing = &DomainError{
		Code:       "ErrOAuthEmailMissing",
		HTTPStatus: http.StatusBadRequest,
		Title:      "Bad Request",
		Message:    "email not provided by oauth provider",
		TypeURI:    "urn:problem:user/err-oauth-email-missing",
	}

	// Generic internal
	ErrInternal = &DomainError{
		Code:       "ErrInternal",
		HTTPStatus: http.StatusInternalServerError,
		Title:      "Internal Server Error",
		Message:    "internal server error",
		TypeURI:    "urn:problem:user/err-internal",
	}
)
