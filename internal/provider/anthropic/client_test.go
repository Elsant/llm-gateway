package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elsant/llm-gateway/internal/gateway"
	"github.com/elsant/llm-gateway/internal/provider/anthropic/api"
)

func userRequest(text string) gateway.Request {
	return gateway.Request{
		Model:     "claude-opus-5-5",
		MaxTokens: 1024,
		Messages:  []gateway.Message{{Role: gateway.RoleUser, Text: text}},
	}
}

// newTestClient starts a fake Messages API backed by handler and returns a
// Client pointed at it. The server is closed when the test ends.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := New("test-key")
	c.baseURL = srv.URL
	return c
}

// respond returns a handler that replies with status and body.
func respond(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func TestComplete_Success(t *testing.T) {
	tests := []struct {
		name string
		body string
		want gateway.Response
	}{
		{
			name: "text blocks are joined",
			body: `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5",
				"content":[{"type":"text","text":"Hello"},{"type":"text","text":", world"}],
				"stop_reason":"end_turn","stop_sequence":null,"usage":{"input_tokens":3,"output_tokens":4}}`,
			want: gateway.Response{
				Text: "Hello, world", Model: "claude-opus-5-5", StopReason: "end_turn",
				Usage: gateway.Usage{InputTokens: 3, OutputTokens: 4},
			},
		},
		{
			name: "non-text blocks are skipped",
			body: `{"model":"m","stop_reason":"end_turn","content":[{"type":"thinking"},{"type":"text","text":"ok"}]}`,
			want: gateway.Response{Text: "ok", Model: "m", StopReason: "end_turn"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, respond(http.StatusOK, tt.body))
			got, err := c.Complete(context.Background(), userRequest("hi"))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resp = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestComplete_Errors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		sentinel error             // matched with errors.Is; nil = none expected
		wantAPI  *gateway.APIError // nil = must not be an *APIError
	}{
		{
			name:     "401 unauthorized",
			status:   401,
			body:     `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`,
			sentinel: gateway.ErrUnauthorized,
			wantAPI:  &gateway.APIError{Status: 401, Type: "authentication_error", Message: "invalid x-api-key"},
		},
		{
			name:     "429 rate limited",
			status:   429,
			body:     `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`,
			sentinel: gateway.ErrRateLimited,
			wantAPI:  &gateway.APIError{Status: 429, Type: "rate_limit_error", Message: "slow down"},
		},
		{
			name:    "400 invalid request",
			status:  400,
			body:    `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"}}`,
			wantAPI: &gateway.APIError{Status: 400, Type: "invalid_request_error", Message: "bad model"},
		},
		{
			name:    "500 api error",
			status:  500,
			body:    `{"type":"error","error":{"type":"api_error","message":"internal"}}`,
			wantAPI: &gateway.APIError{Status: 500, Type: "api_error", Message: "internal"},
		},
		{
			name:    "529 overloaded",
			status:  529,
			body:    `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
			wantAPI: &gateway.APIError{Status: 529, Type: "overloaded_error", Message: "Overloaded"},
		},
		{
			name:    "502 html from proxy",
			status:  502,
			body:    `<html>bad gateway</html>`,
			wantAPI: &gateway.APIError{Status: 502, Message: "<html>bad gateway</html>"},
		},
		{
			name:   "200 broken json",
			status: 200,
			body:   `{"model":"m","content":[{"type":"text","text":"tru`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, respond(tt.status, tt.body))
			got, err := c.Complete(context.Background(), userRequest("hi"))
			if err == nil {
				t.Fatalf("want error, got resp %+v", got)
			}
			if got != (gateway.Response{}) {
				t.Errorf("resp = %+v, want zero value on error", got)
			}
			if !strings.HasPrefix(err.Error(), "anthropic: ") {
				t.Errorf("err = %q, want prefix %q", err, "anthropic: ")
			}
			if tt.sentinel != nil && !errors.Is(err, tt.sentinel) {
				t.Errorf("errors.Is(err, %v) = false; err = %v", tt.sentinel, err)
			}

			var apiErr *gateway.APIError
			isAPI := errors.As(err, &apiErr)
			switch {
			case tt.wantAPI == nil && isAPI:
				t.Errorf("err = %v, want a non-API error", err)
			case tt.wantAPI != nil && !isAPI:
				t.Errorf("err = %v, want *gateway.APIError", err)
			case tt.wantAPI != nil && *apiErr != *tt.wantAPI:
				t.Errorf("APIError = %+v, want %+v", *apiErr, *tt.wantAPI)
			}
		})
	}
}

func TestComplete_SendsHeadersAndBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Errorf("request = %s %s, want POST /v1/messages", r.Method, r.URL.Path)
		}
		for header, want := range map[string]string{
			"x-api-key":         "test-key",
			"anthropic-version": anthropicVersion,
			"content-type":      "application/json",
		} {
			if got := r.Header.Get(header); got != want {
				t.Errorf("header %s = %q, want %q", header, got, want)
			}
		}

		var body api.MessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if body.System != "be brief" || body.Model != "claude-opus-5-5" || body.MaxTokens != 1024 {
			t.Errorf("body = %+v", body)
		}
		respond(200, `{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}]}`)(w, r)
	})

	req := userRequest("hi")
	req.System = "be brief"
	if _, err := c.Complete(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func TestComplete_ContextCanceled(t *testing.T) {
	c := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("request reached the server despite a canceled context")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Complete(ctx, userRequest("hi"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestComplete_NetworkError(t *testing.T) {
	c := New("test-key")
	c.baseURL = "http://127.0.0.1:1" // nothing listens on port 1
	_, err := c.Complete(context.Background(), userRequest("hi"))
	if err == nil {
		t.Fatal("want error for unreachable server")
	}
	var apiErr *gateway.APIError
	if errors.As(err, &apiErr) {
		t.Errorf("err = %v, want a transport error, not *gateway.APIError", err)
	}
}

func TestToMessageRequest_JSON(t *testing.T) {
	body, err := json.Marshal(toMessageRequest(userRequest("Hi")))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"model":"claude-opus-5-5","max_tokens":1024,"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}]}`
	if string(body) != want {
		t.Errorf("got  %s\nwant %s", body, want)
	}
}
