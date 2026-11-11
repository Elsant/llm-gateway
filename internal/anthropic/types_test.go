package anthropic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessageRequest_TemperatureZeroIsKept(t *testing.T) {
	zero := 0.0
	req := MessageRequest{Model: "m", MaxTokens: 1, Temperature: &zero}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"temperature":0`) {
		t.Errorf("temperature 0 dropped: %s", b)
	}
}

func TestMessageRequest_OptionalFieldsOmitted(t *testing.T) {
	req := MessageRequest{Model: "m", MaxTokens: 1, Messages: []Message{}}

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"model":"m","max_tokens":1,"messages":[]}`
	if string(b) != want {
		t.Errorf("got %s, want %s", b, want)
	}
}

func TestMessageResponse_StopSequence(t *testing.T) {
	tests := []struct {
		name string
		json string
		want *string
	}{
		{"null", `{"stop_sequence":null}`, nil},
		{"value", `{"stop_sequence":"END"}`, ptr("END")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp MessageResponse
			if err := json.Unmarshal([]byte(tt.json), &resp); err != nil {
				t.Fatal(err)
			}
			got := resp.StopSequence
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
