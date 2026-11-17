// Package anthropic implements gateway.Provider on top of the Anthropic
// Messages API (POST /v1/messages).
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	api "github.com/elsant/llm-gateway/internal/anthropic"
	"github.com/elsant/llm-gateway/internal/gateway"
)

const (
	defaultBaseURL = "https://api.anthropic.com"

	// anthropicVersion is the required anthropic-version header value.
	anthropicVersion = "2023-06-01"

	// defaultTimeout bounds the whole HTTP exchange. http.Client has no
	// timeout by default, so without it a stuck connection hangs forever.
	// A non-streaming request with max_tokens=16000 can legitimately take
	// minutes, so this is generous; callers that need a tighter limit pass a
	// ctx with a deadline to Complete.
	defaultTimeout = 10 * time.Minute

	// maxErrorBody caps how much of a non-2xx response body is read into the
	// error message.
	maxErrorBody = 64 << 10
)

// Client talks to the Anthropic Messages API.
//
// A Client is meant to be created once with New and shared: it holds a
// single *http.Client, whose connection pool is reused across requests and
// which is safe for concurrent use.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// New returns a Client with the default base URL and timeout.
//
// It returns the concrete *Client, not gateway.Provider: callers that need
// the interface get it implicitly, and every caller shares the same instance
// (and the same http.Client with its connection pool).
func New(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
	}
}

// Complete sends req to POST /v1/messages and returns the text response.
// A non-2xx status is returned as a wrapped *gateway.APIError; use errors.As
// to inspect it, or errors.Is with gateway.ErrUnauthorized and
// gateway.ErrRateLimited for 401 and 429.
func (c *Client) Complete(ctx context.Context, req gateway.Request) (gateway.Response, error) {
	body, err := json.Marshal(toMessageRequest(req))
	if err != nil {
		return gateway.Response{}, fmt.Errorf("build request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return gateway.Response{}, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return gateway.Response{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return gateway.Response{}, fmt.Errorf("anthropic api: %w", parseAPIError(resp.StatusCode, errBody))
	}

	var res api.MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return gateway.Response{}, fmt.Errorf("decode response: %w", err)
	}
	return fromMessageResponse(res), nil
}

// toMessageRequest converts a gateway request into the Messages API body.
func toMessageRequest(req gateway.Request) api.MessageRequest {
	msgs := make([]api.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, api.Message{
			Role:    api.Role(m.Role),
			Content: []api.ContentBlock{api.NewTextBlock(m.Text)},
		})
	}
	return api.MessageRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		System:    req.System,
		Messages:  msgs,
	}
}

// fromMessageResponse joins the text of all "text" content blocks.
func fromMessageResponse(res api.MessageResponse) gateway.Response {
	var sb strings.Builder
	for _, block := range res.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return gateway.Response{Text: sb.String(), Model: res.Model, StopReason: res.StopReason}
}
