package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/elsant/llm-gateway/internal/gateway"
)

func TestDescribeError(t *testing.T) {
	// wrap mimics how a provider returns API errors: wrapped with %w.
	wrap := func(e *gateway.APIError) error { return fmt.Errorf("anthropic api: %w", e) }

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "401 unauthorized",
			err:  wrap(&gateway.APIError{Status: 401, Type: "authentication_error", Message: "invalid x-api-key"}),
			want: "invalid API key, check ANTHROPIC_API_KEY (401 Unauthorized: authentication_error: invalid x-api-key)",
		},
		{
			name: "429 rate limited",
			err:  wrap(&gateway.APIError{Status: 429, Type: "rate_limit_error", Message: "slow down"}),
			want: "rate limit exceeded, wait and retry (429 Too Many Requests: rate_limit_error: slow down)",
		},
		{
			name: "other 4xx",
			err:  wrap(&gateway.APIError{Status: 400, Type: "invalid_request_error", Message: "bad model"}),
			want: "request rejected, check -model, -max-tokens and the prompt (400 Bad Request: invalid_request_error: bad model)",
		},
		{
			name: "5xx",
			err:  wrap(&gateway.APIError{Status: 529, Type: "overloaded_error", Message: "Overloaded"}),
			want: "provider unavailable, try again later (529: overloaded_error: Overloaded)",
		},
		{
			name: "5xx non-json body",
			err:  wrap(&gateway.APIError{Status: 502, Message: "<html>bad gateway</html>"}),
			want: "provider unavailable, try again later (502 Bad Gateway: <html>bad gateway</html>)",
		},
		{
			name: "non-API error",
			err:  fmt.Errorf("send request: %w", context.DeadlineExceeded),
			want: "send request: context deadline exceeded",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeError(tt.err); got != tt.want {
				t.Errorf("describeError = %q\nwant          %q", got, tt.want)
			}
		})
	}
}
