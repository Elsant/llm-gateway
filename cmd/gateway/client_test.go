package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

func TestSend_APIError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantType string
		wantDesc string
	}{
		{
			name:     "401",
			status:   401,
			body:     `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`,
			wantType: "authentication_error",
			wantDesc: "invalid API key, check ANTHROPIC_API_KEY (401 Unauthorized: authentication_error: invalid x-api-key)",
		},
		{
			name:     "429",
			status:   429,
			body:     `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`,
			wantType: "rate_limit_error",
			wantDesc: "rate limit exceeded, wait and retry (429 Too Many Requests: rate_limit_error: slow down)",
		},
		{
			name:     "400",
			status:   400,
			body:     `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"}}`,
			wantType: "invalid_request_error",
			wantDesc: "request rejected (400 Bad Request: invalid_request_error: bad model)",
		},
		{
			name:     "529",
			status:   529,
			body:     `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
			wantType: "overloaded_error",
			wantDesc: "provider unavailable, try again later (529: overloaded_error: Overloaded)",
		},
		{
			name:     "502 html",
			status:   502,
			body:     `<html>bad gateway</html>`,
			wantDesc: "provider unavailable, try again later (502 Bad Gateway: <html>bad gateway</html>)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewClient("test-key")
			c.baseURL = srv.URL
			_, err := c.Send(context.Background(), newMessageRequest(defaultConfig(), "hi"))

			var apiErr *anthropic.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *anthropic.APIError", err)
			}
			if apiErr.Status != tt.status || apiErr.Type != tt.wantType {
				t.Errorf("got status=%d type=%q, want %d %q", apiErr.Status, apiErr.Type, tt.status, tt.wantType)
			}
			if got := describeError(err); got != tt.wantDesc {
				t.Errorf("describeError = %q\nwant          %q", got, tt.wantDesc)
			}
		})
	}
}

func TestSend_RequestHeadersAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"system":"be brief"`) {
			t.Errorf("body missing system prompt: %s", body)
		}
		w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.baseURL = srv.URL
	cfg := requestConfig{Model: "m", MaxTokens: 10, System: "be brief"}
	resp, err := c.Send(context.Background(), newMessageRequest(cfg, "hi"))
	if err != nil {
		t.Fatal(err)
	}
	if got := responseText(resp); got != "ok" {
		t.Errorf("responseText = %q", got)
	}
}
