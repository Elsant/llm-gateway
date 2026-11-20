package gateway

import (
	"errors"
	"fmt"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{"type and message", APIError{Status: 401, Type: "authentication_error", Message: "bad key"}, "401 Unauthorized: authentication_error: bad key"},
		{"message only", APIError{Status: 502, Message: "<html>"}, "502 Bad Gateway: <html>"},
		{"status only", APIError{Status: 503}, "503 Service Unavailable"},
		{"non-standard status", APIError{Status: 529, Type: "overloaded_error", Message: "Overloaded"}, "529: overloaded_error: Overloaded"},
		{"type without message", APIError{Status: 500, Type: "api_error"}, "500 Internal Server Error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		status           int
		wantUnauthorized bool
		wantRateLimited  bool
	}{
		{401, true, false},
		{429, false, true},
		{403, false, false},
		{500, false, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			// Wrapped, as providers return it: errors.Is must see through %w.
			err := fmt.Errorf("provider: %w", &APIError{Status: tt.status})
			if got := errors.Is(err, ErrUnauthorized); got != tt.wantUnauthorized {
				t.Errorf("errors.Is(ErrUnauthorized) = %v, want %v", got, tt.wantUnauthorized)
			}
			if got := errors.Is(err, ErrRateLimited); got != tt.wantRateLimited {
				t.Errorf("errors.Is(ErrRateLimited) = %v, want %v", got, tt.wantRateLimited)
			}
			if errors.Is(err, errors.New("unauthorized")) {
				t.Error("matched an unrelated error with the same text")
			}
		})
	}
}
