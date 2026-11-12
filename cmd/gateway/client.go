package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

const (
	defaultBaseURL = "https://api.anthropic.com"

	// defaultTimeout bounds the whole HTTP exchange. A non-streaming request
	// with max_tokens=16000 can legitimately take minutes, so this is generous;
	// callers that need a tighter limit pass a ctx with a deadline to Send.
	defaultTimeout = 10 * time.Minute
)

// errNotImplemented is returned by Send until the HTTP call is written.
var errNotImplemented = errors.New("anthropic client: send not implemented")

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
//
// For now it only serializes the request and returns errNotImplemented.
func (c *Client) Send(ctx context.Context, req anthropic.MessageRequest) (*anthropic.MessageResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	body, err := encodeRequest(req)
	if err != nil {
		return nil, err
	}
	_ = body // will become the HTTP request body

	return nil, errNotImplemented
}
