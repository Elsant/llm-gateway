package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

const (
	defaultBaseURL = "https://api.anthropic.com"

	// anthropicVersion is the required anthropic-version header value.
	anthropicVersion = "2023-06-01"

	// defaultTimeout bounds the whole HTTP exchange. http.Client has no
	// timeout by default, so without it a stuck connection hangs forever.
	// A non-streaming request with max_tokens=16000 can legitimately take
	// minutes, so this is generous; callers that need a tighter limit pass a
	// ctx with a deadline to Send.
	defaultTimeout = 10 * time.Minute

	// maxErrorBody caps how much of a non-2xx response body is read into the
	// error message.
	maxErrorBody = 64 << 10
)

// Client talks to the Anthropic Messages API.
//
// A Client is meant to be created once with NewClient and shared: it holds a
// single *http.Client, whose connection pool is reused across requests and
// which is safe for concurrent use.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewClient returns a Client with the default base URL and timeout.
//
// It returns *Client so every caller shares the same instance (and the same
// http.Client with its connection pool) instead of copying the struct.
func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
	}
}

// Send sends req to POST /v1/messages and returns the parsed response.
// A non-2xx status is returned as an error containing the status and body.
func (c *Client) Send(ctx context.Context, req anthropic.MessageRequest) (*anthropic.MessageResponse, error) {
	body, err := encodeRequest(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return nil, fmt.Errorf("anthropic api: %s: %s", resp.Status, bytes.TrimSpace(errBody))
	}

	var res anthropic.MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &res, nil
}
