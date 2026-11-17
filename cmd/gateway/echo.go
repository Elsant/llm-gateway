package main

import (
	"context"

	"github.com/elsant/llm-gateway/internal/gateway"
)

// echoProvider is a stub Provider that returns the last user message back.
// It needs no network or API key and exists to check that the CLI works
// against gateway.Provider rather than a concrete client (-provider=echo).
type echoProvider struct{}

func (echoProvider) Complete(ctx context.Context, req gateway.Request) (gateway.Response, error) {
	if err := ctx.Err(); err != nil {
		return gateway.Response{}, err
	}
	var text string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == gateway.RoleUser {
			text = req.Messages[i].Text
			break
		}
	}
	return gateway.Response{Text: text, Model: "echo", StopReason: "end_turn"}, nil
}
