package anthropic

import "testing"

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    APIError
		wantStr string
	}{
		{
			name:    "json body",
			status:  401,
			body:    `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_1"}`,
			want:    APIError{Status: 401, Type: "authentication_error", Message: "invalid x-api-key"},
			wantStr: "401 Unauthorized: authentication_error: invalid x-api-key",
		},
		{
			name:    "non-json body",
			status:  502,
			body:    "  <html>bad gateway</html>\n",
			want:    APIError{Status: 502, Message: "<html>bad gateway</html>"},
			wantStr: "502 Bad Gateway: <html>bad gateway</html>",
		},
		{
			name:    "empty body",
			status:  503,
			body:    "",
			want:    APIError{Status: 503},
			wantStr: "503 Service Unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAPIError(tt.status, []byte(tt.body))
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
			if got.Error() != tt.wantStr {
				t.Errorf("Error() = %q, want %q", got.Error(), tt.wantStr)
			}
		})
	}
}
