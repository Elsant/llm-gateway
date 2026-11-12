package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

const (
	// defaultModel is the current default Claude model (verified against the
	// Claude API docs, 2026-11). Use the bare ID, never a date-suffixed one.
	defaultModel = "claude-opus-5-5"

	// defaultMaxTokens is the recommended ceiling for non-streaming requests:
	// large enough not to cut answers mid-thought, small enough to stay under
	// HTTP timeouts. Streaming requests can go higher.
	defaultMaxTokens = 16000
)

// errEmptyPrompt is returned by buildRequest for an empty or whitespace-only
// prompt. It is a package-level value so callers can check it with errors.Is.
var errEmptyPrompt = errors.New("prompt is empty")

// newMessageRequest returns a request with a single user message containing
// prompt and the default model and max_tokens.
func newMessageRequest(prompt string) anthropic.MessageRequest {
	return anthropic.MessageRequest{
		Model:     defaultModel,
		MaxTokens: defaultMaxTokens,
		Messages: []anthropic.Message{
			{
				Role:    anthropic.RoleUser,
				Content: []anthropic.ContentBlock{anthropic.NewTextBlock(prompt)},
			},
		},
	}
}

// encodeRequest serializes req into the JSON body of POST /v1/messages.
func encodeRequest(req anthropic.MessageRequest) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	return body, nil
}

// buildRequest returns the JSON body of POST /v1/messages with a single user
// message containing prompt.
func buildRequest(prompt string) ([]byte, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, errEmptyPrompt
	}
	return encodeRequest(newMessageRequest(prompt))
}
