package anthropic

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/elsant/llm-gateway/internal/gateway"
	"github.com/elsant/llm-gateway/internal/provider/anthropic/api"
)

// Run locally with, e.g.:
//
//	go test -fuzz=FuzzDecodeResponse -fuzztime=30s ./internal/provider/anthropic
//
// Without -fuzz, `go test` runs only the seed corpus (and anything saved in
// testdata/fuzz), so these also act as regular tests in CI.

// FuzzDecodeResponse feeds arbitrary bytes to the success-path decoder. It
// must never panic, and when it succeeds the text must be exactly the
// concatenation of the "text" blocks.
func FuzzDecodeResponse(f *testing.F) {
	for _, seed := range []string{
		`{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"hi"}]}`,
		`{"content":[{"type":"thinking"},{"type":"text","text":"a"},{"type":"text","text":"b"}]}`,
		`{"content":null,"stop_sequence":null}`,
		`{"content":[{"type":"text","text":"é😀"}]}`,
		`{"model":"m","content":[{"type":"text","text":"tru`,
		`[]`,
		`null`,
		``,
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		got, err := decodeResponse(bytes.NewReader(body))
		if err != nil {
			if got != (gateway.Response{}) {
				t.Fatalf("non-zero response %+v alongside error %v", got, err)
			}
			return
		}

		// Decoding succeeded, so the same bytes decode into the wire type too.
		var res api.MessageResponse
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&res); err != nil {
			t.Fatalf("decodeResponse succeeded but plain decode failed: %v", err)
		}
		var want strings.Builder
		for _, b := range res.Content {
			if b.Type == "text" {
				want.WriteString(b.Text)
			}
		}
		if got.Text != want.String() || got.Model != res.Model || got.StopReason != res.StopReason {
			t.Fatalf("got %+v, want text=%q model=%q stop=%q", got, want.String(), res.Model, res.StopReason)
		}
	})
}

// FuzzParseAPIError checks that any status/body pair yields an APIError that
// keeps the status, formats without panicking and matches the right
// sentinels.
func FuzzParseAPIError(f *testing.F) {
	f.Add(401, []byte(`{"type":"error","error":{"type":"authentication_error","message":"bad key"}}`))
	f.Add(429, []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`))
	f.Add(502, []byte("<html>bad gateway</html>"))
	f.Add(500, []byte(`{"error":{"type":""}}`))
	f.Add(503, []byte(""))
	f.Add(0, []byte(`{"error":"not an object"}`))

	f.Fuzz(func(t *testing.T, status int, body []byte) {
		e := parseAPIError(status, body)
		if e == nil {
			t.Fatal("parseAPIError returned nil")
		}
		if e.Status != status {
			t.Fatalf("Status = %d, want %d", e.Status, status)
		}
		_ = e.Error() // must not panic

		if got, want := errors.Is(e, gateway.ErrUnauthorized), status == 401; got != want {
			t.Fatalf("errors.Is(ErrUnauthorized) = %v for status %d", got, status)
		}
		if got, want := errors.Is(e, gateway.ErrRateLimited), status == 429; got != want {
			t.Fatalf("errors.Is(ErrRateLimited) = %v for status %d", got, status)
		}
		if e.Type == "" && e.Message != strings.TrimSpace(string(body)) {
			t.Fatalf("untyped error must keep the raw body: Message = %q", e.Message)
		}
	})
}
