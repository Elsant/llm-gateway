// Package usage accounts requests and tokens per client.
//
// The gateway will serve many clients concurrently (week 4: one API key per
// client), and every finished request updates the totals, so the counter
// must be safe for concurrent use.
package usage

import (
	"maps"
	"sync"

	"github.com/elsant/llm-gateway/internal/gateway"
)

// Stats are the running totals for one client.
type Stats struct {
	Requests     int64
	InputTokens  int64
	OutputTokens int64
}

// Counter accumulates Stats per client. It is safe for concurrent use; the
// zero value is ready to use. A Counter must not be copied after first use
// (it contains a sync.Mutex; go vet's copylocks check catches copies).
type Counter struct {
	mu      sync.Mutex // guards clients
	clients map[string]Stats
}

// Record adds one request with usage u to client's totals.
func (c *Counter) Record(client string, u gateway.Usage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.clients == nil { // lazy init keeps the zero value usable
		c.clients = make(map[string]Stats)
	}
	// Map values are not addressable (c.clients[client].Requests++ does not
	// compile), so read, modify and write back the struct, all under the lock.
	s := c.clients[client]
	s.Requests++
	s.InputTokens += int64(u.InputTokens)
	s.OutputTokens += int64(u.OutputTokens)
	c.clients[client] = s
}

// Get returns client's totals; zero Stats for an unknown client.
func (c *Counter) Get(client string) Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clients[client] // reading a nil map is fine
}

// Snapshot returns a copy of all totals. The copy is the caller's: later
// Record calls don't change it, and changing it doesn't affect the Counter.
// Returning c.clients itself would hand out the map without the lock.
func (c *Counter) Snapshot() map[string]Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.clients == nil {
		return map[string]Stats{}
	}
	return maps.Clone(c.clients) // Stats has no pointers, so a shallow copy is a full copy
}
