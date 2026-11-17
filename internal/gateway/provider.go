// Package gateway defines the provider-neutral contract the gateway works
// against: the Provider interface and its Request/Response types.
//
// The interface lives here, on the consumer side, not in a provider package:
// providers return concrete types and satisfy Provider implicitly.
package gateway

import "context"

// Provider completes a single, non-streaming request against an LLM backend.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Role is the author of a message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is a single conversation turn with plain-text content.
type Message struct {
	Role Role
	Text string
}

// Request is a provider-neutral completion request.
type Request struct {
	Model     string
	MaxTokens int
	System    string // empty = no system prompt
	Messages  []Message
}

// Response is a provider-neutral completion result.
type Response struct {
	Text       string // concatenated text output
	Model      string // model that actually served the request
	StopReason string // provider-specific value, e.g. "end_turn"
}
