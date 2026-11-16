package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a non-2xx response from the Messages API.
//
// The API returns errors as
//
//	{"type": "error", "error": {"type": "rate_limit_error", "message": "..."}}
//
// Type and Message are taken from the inner "error" object. If the body is
// not in that shape (e.g. an HTML page from a proxy), Type is empty and
// Message holds the raw body.
type APIError struct {
	Status  int    // HTTP status code
	Type    string // error.type, e.g. "authentication_error"
	Message string // error.message
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

// ParseAPIError builds an APIError from a non-2xx status and response body.
func ParseAPIError(status int, body []byte) *APIError {
	var wire struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &wire); err == nil && wire.Error.Type != "" {
		return &APIError{Status: status, Type: wire.Error.Type, Message: wire.Error.Message}
	}
	return &APIError{Status: status, Message: strings.TrimSpace(string(body))}
}
