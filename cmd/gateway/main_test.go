package main

import (
	"fmt"
	"testing"

	api "github.com/elsant/llm-gateway/internal/anthropic"
)

func TestDescribeError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"401", 401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`,
			"invalid API key, check ANTHROPIC_API_KEY (401 Unauthorized: authentication_error: invalid x-api-key)"},
		{"429", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`,
			"rate limit exceeded, wait and retry (429 Too Many Requests: rate_limit_error: slow down)"},
		{"400", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"}}`,
			"request rejected (400 Bad Request: invalid_request_error: bad model)"},
		{"529", 529, `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
			"provider unavailable, try again later (529: overloaded_error: Overloaded)"},
		{"502 html", 502, `<html>bad gateway</html>`,
			"provider unavailable, try again later (502 Bad Gateway: <html>bad gateway</html>)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wrapped the same way Client.Complete wraps it.
			err := fmt.Errorf("anthropic api: %w", api.ParseAPIError(tt.status, []byte(tt.body)))
			if got := describeError(err); got != tt.want {
				t.Errorf("describeError = %q\nwant          %q", got, tt.want)
			}
		})
	}
}
