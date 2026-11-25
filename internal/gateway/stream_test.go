package gateway

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeProvider returns a fixed response or error.
type fakeProvider struct {
	text string
	err  error
}

func (f fakeProvider) Complete(context.Context, Request) (Response, error) {
	return Response{Text: f.text}, f.err
}

func collect(ch <-chan string) []string {
	var got []string
	for chunk := range ch { // ends when the sender closes the channel
		got = append(got, chunk)
	}
	return got
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"hello", []string{"hello"}},
		{"hello world", []string{"hello ", "world"}},
		{"  lead  and trail  ", []string{"  lead  ", "and ", "trail  "}},
		{"line one\nline two\n", []string{"line ", "one\n", "line ", "two\n"}},
		{"привет, мир", []string{"привет, ", "мир"}},
		{"   ", []string{"   "}},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := splitWords(tt.in)
			if !slices.Equal(got, tt.want) {
				t.Errorf("splitWords(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if joined := strings.Join(got, ""); joined != tt.in {
				t.Errorf("joined = %q, want the original %q", joined, tt.in)
			}
		})
	}
}

func TestStreamWords(t *testing.T) {
	ch, err := StreamWords(context.Background(), fakeProvider{text: "Go has channels"}, Request{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Go ", "has ", "channels"}
	if got := collect(ch); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStreamWords_CompleteError(t *testing.T) {
	boom := errors.New("boom")
	ch, err := StreamWords(context.Background(), fakeProvider{err: boom}, Request{})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want %v", err, boom)
	}
	if ch != nil {
		t.Error("channel must be nil on error")
	}
}

func TestStreamWords_CancelStopsSender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := StreamWords(ctx, fakeProvider{text: strings.Repeat("word ", 1000)}, Request{})
	if err != nil {
		t.Fatal(err)
	}

	<-ch // read one word, then lose interest
	cancel()

	// The sender must notice ctx.Done() and close the channel instead of
	// blocking forever on the next send (a goroutine leak). At most one
	// more word may already be in flight.
	deadline := time.After(time.Second)
	for received := 0; ; received++ {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			if received > 1 {
				t.Fatalf("still receiving words after cancel (%d)", received)
			}
		case <-deadline:
			t.Fatal("channel not closed after cancel: sender goroutine leaked")
		}
	}
}
