package anthropic

import (
	"errors"
	"testing"

	"github.com/elsant/llm-gateway/internal/gateway"
)

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name             string
		status           int
		body             string
		want             gateway.APIError
		wantStr          string
		wantUnauthorized bool
		wantRateLimited  bool
	}{
		{
			name:             "401",
			status:           401,
			body:             `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_1"}`,
			want:             gateway.APIError{Status: 401, Type: "authentication_error", Message: "invalid x-api-key"},
			wantStr:          "401 Unauthorized: authentication_error: invalid x-api-key",
			wantUnauthorized: true,
		},
		{
			name:            "429",
			status:          429,
			body:            `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`,
			want:            gateway.APIError{Status: 429, Type: "rate_limit_error", Message: "slow down"},
			wantStr:         "429 Too Many Requests: rate_limit_error: slow down",
			wantRateLimited: true,
		},
		{
			name:    "other 4xx",
			status:  400,
			body:    `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"}}`,
			want:    gateway.APIError{Status: 400, Type: "invalid_request_error", Message: "bad model"},
			wantStr: "400 Bad Request: invalid_request_error: bad model",
		},
		{
			name:    "5xx json",
			status:  529,
			body:    `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
			want:    gateway.APIError{Status: 529, Type: "overloaded_error", Message: "Overloaded"},
			wantStr: "529: overloaded_error: Overloaded",
		},
		{
			name:    "5xx non-json body",
			status:  502,
			body:    "  <html>bad gateway</html>\n",
			want:    gateway.APIError{Status: 502, Message: "<html>bad gateway</html>"},
			wantStr: "502 Bad Gateway: <html>bad gateway</html>",
		},
		{
			name:    "5xx empty body",
			status:  503,
			body:    "",
			want:    gateway.APIError{Status: 503},
			wantStr: "503 Service Unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAPIError(tt.status, []byte(tt.body))
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
			if got.Error() != tt.wantStr {
				t.Errorf("Error() = %q, want %q", got.Error(), tt.wantStr)
			}
			if is := errors.Is(got, gateway.ErrUnauthorized); is != tt.wantUnauthorized {
				t.Errorf("errors.Is(ErrUnauthorized) = %v, want %v", is, tt.wantUnauthorized)
			}
			if is := errors.Is(got, gateway.ErrRateLimited); is != tt.wantRateLimited {
				t.Errorf("errors.Is(ErrRateLimited) = %v, want %v", is, tt.wantRateLimited)
			}
		})
	}
}
