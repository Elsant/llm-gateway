package usage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/elsant/llm-gateway/internal/gateway"
)

func TestCounter_ZeroValue(t *testing.T) {
	var c Counter // no constructor needed
	if got := c.Get("nobody"); got != (Stats{}) {
		t.Errorf("Get on empty counter = %+v, want zero", got)
	}
	if got := c.Snapshot(); len(got) != 0 {
		t.Errorf("Snapshot on empty counter = %v, want empty", got)
	}
}

func TestCounter_Record(t *testing.T) {
	var c Counter
	c.Record("alice", gateway.Usage{InputTokens: 10, OutputTokens: 20})
	c.Record("alice", gateway.Usage{InputTokens: 1, OutputTokens: 2})
	c.Record("bob", gateway.Usage{InputTokens: 5})

	tests := []struct {
		client string
		want   Stats
	}{
		{"alice", Stats{Requests: 2, InputTokens: 11, OutputTokens: 22}},
		{"bob", Stats{Requests: 1, InputTokens: 5}},
		{"carol", Stats{}},
	}
	for _, tt := range tests {
		t.Run(tt.client, func(t *testing.T) {
			if got := c.Get(tt.client); got != tt.want {
				t.Errorf("Get(%q) = %+v, want %+v", tt.client, got, tt.want)
			}
		})
	}
}

func TestCounter_SnapshotIsACopy(t *testing.T) {
	var c Counter
	c.Record("alice", gateway.Usage{InputTokens: 1})

	snap := c.Snapshot()
	snap["alice"] = Stats{Requests: 999}
	delete(snap, "alice")
	c.Record("alice", gateway.Usage{InputTokens: 1})

	if got := c.Get("alice"); got != (Stats{Requests: 2, InputTokens: 2}) {
		t.Errorf("counter changed through the snapshot: %+v", got)
	}
	if len(snap) != 0 {
		t.Errorf("snapshot changed by a later Record: %v", snap)
	}
}

// TestCounter_Concurrent hammers the counter from many goroutines. Without
// the mutex, `go test -race` reports a data race here, and even without
// -race the totals come out short or the runtime crashes with
// "concurrent map writes".
func TestCounter_Concurrent(t *testing.T) {
	const (
		clients    = 10
		goroutines = 50
		perG       = 200
	)
	var c Counter
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range perG {
				client := fmt.Sprintf("client-%d", (g+i)%clients)
				c.Record(client, gateway.Usage{InputTokens: 1, OutputTokens: 2})
				_ = c.Get(client) // readers run concurrently with writers
			}
		})
	}
	// Snapshots taken while writers run must not race either.
	wg.Go(func() {
		for range 100 {
			_ = c.Snapshot()
		}
	})
	wg.Wait()

	var total Stats
	for _, s := range c.Snapshot() {
		total.Requests += s.Requests
		total.InputTokens += s.InputTokens
		total.OutputTokens += s.OutputTokens
	}
	const n = goroutines * perG
	if want := (Stats{Requests: n, InputTokens: n, OutputTokens: 2 * n}); total != want {
		t.Errorf("total = %+v, want %+v", total, want)
	}
}

func BenchmarkCounter_RecordParallel(b *testing.B) {
	var c Counter
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Record("client", gateway.Usage{InputTokens: 1, OutputTokens: 1})
		}
	})
}
