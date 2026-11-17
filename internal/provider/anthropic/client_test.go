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
)

func userRequest(text string) gateway.Request {
	return gateway.Request{
		Model:     "claude-opus-5-5",
		MaxTokens: 1024,
		Messages:  []gateway.Message{{Role: gateway.RoleUser, Text: text}},
	}
}

func TestComplete_APIError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantType string
	}{
		{"401", 401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`, "authentication_error"},
		{"429", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`, "rate_limit_error"},
		{"400", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"}}`, "invalid_request_error"},
		{"529", 529, `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`, "overloaded_error"},
		{"502 html", 502, `<html>bad gateway</html>`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := New("test-key")
			c.baseURL = srv.URL
			_, err := c.Complete(context.Background(), userRequest("hi"))

			var apiErr *gateway.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("err = %v, want *gateway.APIError", err)
			}
			if apiErr.Status != tt.status || apiErr.Type != tt.wantType {
				t.Errorf("got status=%d type=%q, want %d %q", apiErr.Status, apiErr.Type, tt.status, tt.wantType)
			}
		})
	}
}

func TestComplete_RequestHeadersAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"system":"be brief"`) {
			t.Errorf("body missing system prompt: %s", body)
		}
		w.Write([]byte(`{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"o"},{"type":"text","text":"k"}]}`))
	}))
	defer srv.Close()

	c := New("test-key")
	c.baseURL = srv.URL
	req := userRequest("hi")
	req.System = "be brief"
	resp, err := c.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	want := gateway.Response{Text: "ok", Model: "m", StopReason: "end_turn"}
	if resp != want {
		t.Errorf("resp = %+v, want %+v", resp, want)
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
