package echo

import (
	"context"
	"errors"
	"testing"

	"github.com/elsant/llm-gateway/internal/gateway"
)

func TestComplete(t *testing.T) {
	user := func(s string) gateway.Message { return gateway.Message{Role: gateway.RoleUser, Text: s} }
	assistant := func(s string) gateway.Message { return gateway.Message{Role: gateway.RoleAssistant, Text: s} }

	tests := []struct {
		name     string
		messages []gateway.Message
		want     string
	}{
		{"single user message", []gateway.Message{user("hi")}, "hi"},
		{"last user message wins", []gateway.Message{user("one"), assistant("ok"), user("two")}, "two"},
		{"skips trailing assistant", []gateway.Message{user("q"), assistant("a")}, "q"},
		{"no user message", []gateway.Message{assistant("a")}, ""},
		{"no messages", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Provider{}.Complete(context.Background(), gateway.Request{Messages: tt.messages})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := gateway.Response{Text: tt.want, Model: Model, StopReason: "end_turn"}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestComplete_ContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Provider{}.Complete(ctx, gateway.Request{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
