package gateway

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for provider failures callers usually handle specially.
// A provider returns an *APIError; these match it through errors.Is:
//
//	if errors.Is(err, gateway.ErrRateLimited) { ... }
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrRateLimited  = errors.New("rate limited")
)

// APIError is a non-2xx response from a provider's API.
//
// Type and Message come from the provider's error body when it has one
// (for Anthropic: error.type and error.message). If the body is not in the
// expected shape (e.g. an HTML page from a proxy), Type is empty and Message
// holds the raw body.
type APIError struct {
	Status  int    // HTTP status code
	Type    string // provider error type, e.g. "authentication_error"
	Message string // provider error message
}

func (e *APIError) Error() string {
	status := fmt.Sprint(e.Status)
	if text := http.StatusText(e.Status); text != "" {
		status += " " + text
	}
	switch {
	case e.Type != "" && e.Message != "":
		return fmt.Sprintf("%s: %s: %s", status, e.Type, e.Message)
	case e.Message != "":
		return fmt.Sprintf("%s: %s", status, e.Message)
	default:
		return status
	}
}

// Is makes errors.Is(err, ErrUnauthorized) true for a 401 and
// errors.Is(err, ErrRateLimited) true for a 429.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.Status == http.StatusUnauthorized
	case ErrRateLimited:
		return e.Status == http.StatusTooManyRequests
	}
	return false
}
