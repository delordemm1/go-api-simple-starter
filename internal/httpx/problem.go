package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"
)

// Problem implements RFC 9457/7807-compatible problem+json with custom extensions.
// Extensions included:
//   - code: stable business code (e.g., ErrInvalidResetToken)
//   - context: extra error payload (e.g., validation fields map)
//   - requestId: propagated from chi middleware.RequestID
type Problem struct {
	// RFC 9457 standard fields
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`

	// Huma-compatible list of detailed errors (optional usage)
	Errors []*huma.ErrorDetail `json:"errors,omitempty"`

	// Extensions (custom)
	Code      string `json:"code,omitempty"`
	Context   any    `json:"context,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// Error implements error interface by returning the problem detail.
func (p *Problem) Error() string {
	if p.Detail != "" {
		return p.Detail
	}
	if p.Title != "" {
		return p.Title
	}
	return http.StatusText(p.GetStatus())
}

// GetStatus implements huma.StatusError to set HTTP response status.
func (p *Problem) GetStatus() int {
	if p.Status == 0 {
		return http.StatusInternalServerError
	}
	return p.Status
}

// ContentType implements huma.ContentTypeFilter to ensure application/problem+json.
func (p *Problem) ContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	if ct == "application/cbor" {
		return "application/problem+cbor"
	}
	return ct
}

// DomainProblem is a minimal interface for domain errors so the formatter
// can build RFC 7807 problems without enumerating all domain error types.
//
// Any domain error type across modules can satisfy this.
type DomainProblem interface {
	ProblemCode() string
	ProblemStatus() int
	ProblemTitle() string
	ProblemDetail() string
	ProblemTypeURI() string
	ProblemContext() any
}

// ToProblem converts any error into an RFC 7807 Problem with extensions.
//
// Behavior:
//   - If err already implements huma.StatusError (e.g., a Problem), it is returned as-is.
//   - If err implements DomainProblem, it is formatted into a Problem.
//   - Otherwise, returns a generic internal Problem with code ErrInternal.
func ToProblem(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// If it's already a Huma status error (including our Problem), pass through.
	if _, ok := err.(huma.StatusError); ok {
		return err
	}

	// Domain-driven mapping w/o enumerating types.
	var dp DomainProblem
	if errors.As(err, &dp) {
		code := dp.ProblemCode()
		status := dp.ProblemStatus()
		title := dp.ProblemTitle()
		detail := dp.ProblemDetail()
		typeURI := dp.ProblemTypeURI()
		if typeURI == "" {
			typeURI = "urn:problem:" + toKebab(code)
		}

		reqID := middleware.GetReqID(ctx)
		return &Problem{
			Type:      typeURI,
			Title:     defaultTitle(title, status),
			Status:    status,
			Detail:    defaultDetail(detail, status),
			Code:      code,
			Context:   dp.ProblemContext(),
			RequestID: reqID,
		}
	}

	// Fallback internal problem.
	return InternalProblem(ctx, "")
}

// ValidationProblem builds a 400 validation error with the required context fields map.
func ValidationProblem(ctx context.Context, summary string, fields map[string][]string) *Problem {
	if summary == "" {
		summary = "Validation error"
	}
	return &Problem{
		Type:      "urn:problem:validation-error",
		Title:     "Validation error",
		Status:    http.StatusBadRequest,
		Detail:    summary,
		Code:      "ErrValidation",
		Context:   map[string]any{"fields": fields},
		RequestID: middleware.GetReqID(ctx),
	}
}

// InternalProblem builds a generic 500 internal error problem. If detail is empty,
// a safe user-friendly message will be used.
func InternalProblem(ctx context.Context, detail string) *Problem {
	if detail == "" {
		detail = "Something went wrong. Please try again later."
	}
	return &Problem{
		Type:      "urn:problem:internal",
		Title:     http.StatusText(http.StatusInternalServerError),
		Status:    http.StatusInternalServerError,
		Detail:    detail,
		Code:      "ErrInternal",
		RequestID: middleware.GetReqID(ctx),
	}
}

func defaultTitle(title string, status int) string {
	if title != "" {
		return title
	}
	return http.StatusText(status)
}

func defaultDetail(detail string, status int) string {
	if detail != "" {
		return detail
	}
	switch status {
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not found"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusBadRequest:
		return "Bad request"
	default:
		return http.StatusText(status)
	}
}

// toKebab converts codes like ErrInvalidResetToken or USER_NOT_FOUND to
// kebab-case: err-invalid-reset-token, user-not-found
func toKebab(s string) string {
	if s == "" {
		return ""
	}
	// Normalize underscores/spaces to hyphen and camel-case boundaries to hyphen.
	var b strings.Builder
	var prevIsLowerOrDigit bool
	for i, r := range s {
		switch r {
		case '_', ' ', '-':
			if b.Len() > 0 {
				last, _ := lastRune(&b)
				if last != '-' {
					b.WriteByte('-')
				}
			}
			prevIsLowerOrDigit = false
			continue
		}
		if i > 0 && unicode.IsUpper(r) && prevIsLowerOrDigit {
			b.WriteByte('-')
		}
		b.WriteRune(unicode.ToLower(r))
		prevIsLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
	}

	// Collapse any duplicate hyphens (defensive)
	out := strings.ReplaceAll(b.String(), "--", "-")
	return out
}

func lastRune(b *strings.Builder) (rune, int) {
	str := b.String()
	if len(str) == 0 {
		return 0, 0
	}
	r, size := rune(str[len(str)-1]), 1
	return r, size
}