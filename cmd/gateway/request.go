package main

import (
	"github.com/elsant/llm-gateway/internal/config"
	"github.com/elsant/llm-gateway/internal/gateway"
)

// newRequest returns a request with a single user message containing
// cfg.Prompt and the model, max_tokens and system prompt from cfg.
func newRequest(cfg config.Config) gateway.Request {
	return gateway.Request{
		Model:     cfg.Model,
		MaxTokens: cfg.MaxTokens,
		System:    cfg.System,
		Messages:  []gateway.Message{{Role: gateway.RoleUser, Text: cfg.Prompt}},
	}
}
