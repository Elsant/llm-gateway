// Package anthropic contains wire types for the Anthropic Messages API
// (POST /v1/messages). Field names and shapes follow the HTTP API reference,
// not any SDK: https://docs.anthropic.com/en/api/messages
package anthropic

// Role is the author of a message in the conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// MessageRequest is the body of POST /v1/messages.
//
// Required fields have no omitempty: if they are zero we want the API to reject
// the request loudly instead of the field silently disappearing from JSON.
type MessageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`

	// System is the system prompt. The API also accepts an array of text
	// blocks here; only the string form is supported for now.
	System string `json:"system,omitempty"`

	// Temperature is a pointer because 0 is a meaningful value (deterministic
	// sampling). With a plain float64 and omitempty, 0 would be dropped and the
	// API default (1.0) would apply instead. nil = "not set, use API default".
	Temperature *float64 `json:"temperature,omitempty"`

	StopSequences []string `json:"stop_sequences,omitempty"`

	// Stream: false is the API default, so dropping it is harmless.
	Stream bool `json:"stream,omitempty"`

	// Metadata is a pointer so that "no metadata" omits the key entirely;
	// omitempty does not omit a non-pointer struct value.
	Metadata *Metadata `json:"metadata,omitempty"`
}

// Metadata is the optional request metadata object.
type Metadata struct {
	// UserID is an opaque identifier (uuid/hash) of the end user, used by
	// Anthropic for abuse detection. Never put names or emails here.
	UserID string `json:"user_id,omitempty"`
}

// Message is a single conversation turn.
//
// In the API, content is either a plain string or an array of content blocks.
// Only the array form is supported for now; the string shorthand
// ("content": "Hello") is not.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock is one block of message content. Only "text" blocks are
// modelled so far (no image, tool_use, tool_result, thinking, ...).
type ContentBlock struct {
	Type string `json:"type"`
	// omitempty so that future non-text blocks don't carry "text": "".
	Text string `json:"text,omitempty"`
}

// NewTextBlock returns a content block of type "text".
func NewTextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// MessageResponse is the body of a successful non-streaming response.
type MessageResponse struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"` // always "message"
	Role    Role           `json:"role"` // always "assistant"
	Model   string         `json:"model"`
	Content []ContentBlock `json:"content"`

	// StopReason is one of "end_turn", "max_tokens", "stop_sequence",
	// "tool_use", "pause_turn", "refusal". It is non-null in non-streaming
	// responses (it is null only in the streaming message_start event), so a
	// plain string is enough here.
	StopReason string `json:"stop_reason"`

	// StopSequence is null unless StopReason == "stop_sequence". A pointer
	// separates "null" (nil) from a matched empty string (non-nil "").
	StopSequence *string `json:"stop_sequence"`

	Usage Usage `json:"usage"`
}

// Usage reports billed token counts. The API also returns cache-related
// fields (cache_creation_input_tokens, cache_read_input_tokens); they are not
// modelled yet.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
