package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/elsant/llm-gateway/internal/gateway"
)

// parseAPIError builds a *gateway.APIError from a non-2xx status and the
// response body. The Messages API returns errors as
//
//	{"type": "error", "error": {"type": "rate_limit_error", "message": "..."}}
//
// Any other body (e.g. an HTML page from a proxy) is kept as the message.
// 401 and 429 then match gateway.ErrUnauthorized and gateway.ErrRateLimited.
func parseAPIError(status int, body []byte) *gateway.APIError {
	var wire struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &wire); err == nil && wire.Error.Type != "" {
		return &gateway.APIError{Status: status, Type: wire.Error.Type, Message: wire.Error.Message}
	}
	return &gateway.APIError{Status: status, Message: strings.TrimSpace(string(body))}
}
