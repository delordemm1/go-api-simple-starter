package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldErrors maps JSON field names to a list of validation error messages.
type FieldErrors map[string][]string

// ValidationError implements a DomainProblem (from internal/httpx) without importing it directly,
// by providing the required method set. This avoids cycles and lets httpx.ToProblem format it.
type ValidationError struct {
	summary string
	fields  FieldErrors
}

func (e *ValidationError) Error() string { return e.summary }

// Domain-problem methods (structural typing against httpx.DomainProblem)

func (e *ValidationError) ProblemCode() string    { return "ErrValidation" }
func (e *ValidationError) ProblemStatus() int     { return 400 }
func (e *ValidationError) ProblemTitle() string   { return "Validation error" }
func (e *ValidationError) ProblemDetail() string  { return e.summary }
func (e *ValidationError) ProblemTypeURI() string { return "urn:problem:validation-error" }
func (e *ValidationError) ProblemContext() any    { return map[string]any{"fields": e.fields} }

// ValidateStruct validates a struct instance according to `validate` tags.
// On success it returns nil. On failure it returns a *ValidationError with:
// - summary: "invalid <field>, and N other errors" or "validation failed"
// - fields:  map of JSON field name to list of messages
func ValidateStruct(v any) error {
	validate := validator.New()

	// Use JSON tag names instead of struct field names.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		jsonTag := fld.Tag.Get("json")
		name := strings.Split(jsonTag, ",")[0]
		if name == "" || name == "-" {
			// Fallback: lower-camel case of field
			return lowerFirst(fld.Name)
		}
		return name
	})

	if err := validate.Struct(v); err != nil {
		if verrs, ok := err.(validator.ValidationErrors); ok {
			fields := make(FieldErrors)
			for _, fe := range verrs {
				field := fe.Field() // already JSON-tagged due to RegisterTagNameFunc
				msg := messageForTag(fe)
				fields[field] = append(fields[field], msg)
			}

			// Build summarized detail per spec, e.g. "invalid email, and 2 other errors"
			summary := summarize(fields)
			return &ValidationError{
				summary: summary,
				fields:  fields,
			}
		}
		// Non-standard error from validator, return a generic summary.
		return &ValidationError{
			summary: "validation failed",
			fields:  FieldErrors{},
		}
	}
	return nil
}

func messageForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		// Handle string length min
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at most %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "eqfield":
		// Match other field; convert to JSON lower-camel if needed
		return fmt.Sprintf("must match %s", toJSONFieldName(fe.Param()))
	case "eq":
		// Used e.g. AcceptTerms eq=true
		if fe.Param() == "true" {
			// Make friendlier wording for common boolean accept terms
			if strings.Contains(strings.ToLower(fe.Field()), "terms") {
				return "must be accepted"
			}
			return "must be true"
		}
		return fmt.Sprintf("must equal %s", fe.Param())
	default:
		return "is invalid"
	}
}

func summarize(fields FieldErrors) string {
	// Prefer specific phrase for common cases like "invalid email"
	if msgs, ok := fields["email"]; ok {
		for _, m := range msgs {
			if strings.Contains(m, "valid email") {
				others := countOthers(fields, "email")
				if others > 0 {
					return fmt.Sprintf("invalid email, and %d other error%s", others, plural(others))
				}
				return "invalid email"
			}
		}
	}
	// Fallback: use first field: first message
	firstField, firstMsg := first(fields)
	if firstField != "" && firstMsg != "" {
		others := totalCount(fields) - 1
		if others > 0 {
			return fmt.Sprintf("%s %s, and %d other error%s", firstField, firstMsg, others, plural(others))
		}
		return fmt.Sprintf("%s %s", firstField, firstMsg)
	}
	return "validation failed"
}

func toJSONFieldName(structField string) string {
	// Convert typical struct field (e.g., ConfirmPassword) to lower-camel
	return lowerFirst(structField)
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(strings.ToLower(string(r[0])))[0]
	return string(r)
}

func first(m FieldErrors) (string, string) {
	for k, list := range m {
		if len(list) > 0 {
			return k, list[0]
		}
	}
	return "", ""
}

func totalCount(m FieldErrors) int {
	n := 0
	for _, list := range m {
		n += len(list)
	}
	return n
}

func countOthers(m FieldErrors, field string) int {
	n := 0
	for k, list := range m {
		if k == field {
			continue
		}
		n += len(list)
	}
	return n
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}