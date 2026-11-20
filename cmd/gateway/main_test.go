package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/elsant/llm-gateway/internal/config"
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

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string // empty = success
	}{
		{"anthropic with key", config.Config{Provider: "anthropic", AnthropicAPIKey: "sk-test"}, ""},
		{"anthropic without key", config.Config{Provider: "anthropic"}, "ANTHROPIC_API_KEY is not set"},
		{"echo needs no key", config.Config{Provider: "echo"}, ""},
		{"unknown provider", config.Config{Provider: "openai"}, `unknown -provider "openai"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := newProvider(tt.cfg)
			if tt.wantErr == "" {
				if err != nil || p == nil {
					t.Fatalf("got (%v, %v), want a provider", p, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	cfg := config.Config{Model: "m", MaxTokens: 7, System: "sys", Prompt: "hi"}
	got := newRequest(cfg)
	if got.Model != "m" || got.MaxTokens != 7 || got.System != "sys" {
		t.Errorf("request params = %+v", got)
	}
	want := []gateway.Message{{Role: gateway.RoleUser, Text: "hi"}}
	if len(got.Messages) != 1 || got.Messages[0] != want[0] {
		t.Errorf("Messages = %+v, want %+v", got.Messages, want)
	}
}
