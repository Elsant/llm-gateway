// Package echo implements a fake gateway.Provider that returns the last user
// message back. It needs no network or API key: use it in tests and to run
// the CLI without credentials (-provider=echo).
package echo

import (
	"context"

	"github.com/elsant/llm-gateway/internal/gateway"
)

// Model is the value echo reports in Response.Model.
const Model = "echo"

// Provider must satisfy gateway.Provider.
var _ gateway.Provider = Provider{}

// Provider echoes the last user message. The zero value is ready to use.
type Provider struct{}

// Complete returns the text of the last user message in req, or an empty
// text if there is none. It fails only if ctx is already done.
func (Provider) Complete(ctx context.Context, req gateway.Request) (gateway.Response, error) {
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
	return gateway.Response{Text: text, Model: Model, StopReason: "end_turn"}, nil
}
