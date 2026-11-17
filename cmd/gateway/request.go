package main

import "github.com/elsant/llm-gateway/internal/gateway"

const (
	// defaultModel is the current default Claude model (verified against the
	// Claude API docs, 2026-11). Use the bare ID, never a date-suffixed one.
	defaultModel = "claude-opus-5-5"

	// defaultMaxTokens is the default -max-tokens value: enough for a short
	// CLI answer. Raise it with the flag for longer outputs.
	defaultMaxTokens = 1024
)

// requestConfig holds the request parameters set by command-line flags.
type requestConfig struct {
	Model     string
	MaxTokens int
	System    string // empty = no system prompt
}

// defaultConfig returns the config used when no flags are given.
func defaultConfig() requestConfig {
	return requestConfig{Model: defaultModel, MaxTokens: defaultMaxTokens}
}

// newRequest returns a request with a single user message containing prompt
// and the model, max_tokens and system prompt from cfg.
func newRequest(cfg requestConfig, prompt string) gateway.Request {
	return gateway.Request{
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
		System:    cfg.System,
		Messages:  []gateway.Message{{Role: gateway.RoleUser, Text: prompt}},
	}
}
