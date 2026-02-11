package go_nova

import (
	"errors"
	"fmt"
)

// ValidationError indicates that a request is missing required fields or contains invalid data.
type ValidationError struct {
	Fields []FieldError
}

type FieldError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Fields) == 0 {
		return "validation error"
	}
	if len(e.Fields) == 1 {
		fe := e.Fields[0]
		if fe.Field == "" {
			return fmt.Sprintf("validation error: %s", fe.Message)
		}
		return fmt.Sprintf("validation error: %s: %s", fe.Field, fe.Message)
	}
	return fmt.Sprintf("validation error: %d fields", len(e.Fields))
}

func (e *ValidationError) Add(field, message string) {
	e.Fields = append(e.Fields, FieldError{Field: field, Message: message})
}

func (e *ValidationError) HasErrors() bool {
	return e != nil && len(e.Fields) > 0
}

// IsValidationError checks whether err is a *ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// APIError represents a non-2xx response from NovaPay.
type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "novapay api error"
	}
	if len(e.Body) == 0 {
		return fmt.Sprintf("novapay api error: status %d", e.StatusCode)
	}
	b := e.Body
	if len(b) > 1024 {
		b = b[:1024]
	}
	return fmt.Sprintf("novapay api error: status %d: %s", e.StatusCode, string(b))
}
