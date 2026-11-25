package channels

import (
	"testing"
	"time"
)

// Channel behavior, verified below:
//
//	operation      | nil channel    | open channel                  | closed channel
//	---------------+----------------+-------------------------------+------------------------------
//	send  ch <- v  | blocks forever | blocks until a receiver/space | PANIC "send on closed channel"
//	recv  <-ch     | blocks forever | blocks until a value          | zero value immediately, ok=false
//	                                                                  (buffered: drains values first)
//	close(ch)      | PANIC          | OK; wakes all receivers        | PANIC "close of closed channel"
//	len / cap      | 0 / 0          | queued / buffer size          | remaining values / buffer size
//
// "Blocks forever" is checked with a short timeout: if the operation has
// not completed after blockWait, we treat it as blocked.

const blockWait = 50 * time.Millisecond

// blocks reports whether op is still running after blockWait. A blocked op
// leaves its goroutine behind; fine in a test binary that is about to exit.
func blocks(op func()) bool {
	done := make(chan struct{})
	go func() {
		op()
		close(done)
	}()
	select {
	case <-done:
		return false
	case <-time.After(blockWait):
		return true
	}
}

// panics returns the panic value of op, or nil if it did not panic.
func panics(op func()) (v any) {
	defer func() { v = recover() }()
	op()
	return nil
}

func TestNilChannel(t *testing.T) {
	var ch chan int // zero value of a channel type is nil

	if !blocks(func() { ch <- 1 }) {
		t.Error("send on nil channel: want block")
	}
	if !blocks(func() { <-ch }) {
		t.Error("receive from nil channel: want block")
	}
	if v := panics(func() { close(ch) }); v == nil {
		t.Error("close of nil channel: want panic")
	} else {
		t.Logf("close(nil) panics: %v", v)
	}
	if len(ch) != 0 || cap(ch) != 0 {
		t.Errorf("len/cap of nil channel = %d/%d", len(ch), cap(ch))
	}
}

func TestNilChannelDisablesSelectCase(t *testing.T) {
	// The useful side of "nil blocks forever": setting a channel to nil
	// switches its select case off.
	var disabled chan int
	ready := make(chan int, 1)
	ready <- 42

	select {
	case <-disabled:
		t.Error("nil channel case must never fire")
	case v := <-ready:
		if v != 42 {
			t.Errorf("got %d", v)
		}
	}
}

func TestOpenUnbufferedChannel(t *testing.T) {
	ch := make(chan int)

	if !blocks(func() { ch <- 1 }) {
		t.Error("send with no receiver: want block")
	}
	ch2 := make(chan int)
	if !blocks(func() { <-ch2 }) {
		t.Error("receive with no sender: want block")
	}

	// With a receiver on the other side the send completes: a rendezvous.
	ch3 := make(chan int)
	go func() { ch3 <- 7 }()
	if v := <-ch3; v != 7 {
		t.Errorf("got %d, want 7", v)
	}
}

func TestOpenBufferedChannel(t *testing.T) {
	ch := make(chan int, 2)

	if blocks(func() { ch <- 1; ch <- 2 }) {
		t.Error("send into free buffer: want no block")
	}
	if len(ch) != 2 || cap(ch) != 2 {
		t.Errorf("len/cap = %d/%d, want 2/2", len(ch), cap(ch))
	}
	if !blocks(func() { ch <- 3 }) {
		t.Error("send into full buffer: want block")
	}
}

func TestClosedChannel(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1
	close(ch)

	// Buffered values survive close and are drained first.
	if v, ok := <-ch; v != 1 || !ok {
		t.Errorf("first receive = %d, %v; want 1, true", v, ok)
	}
	// Then: zero value, ok=false, immediately, as many times as you like.
	for range 3 {
		if v, ok := <-ch; v != 0 || ok {
			t.Errorf("receive after drain = %d, %v; want 0, false", v, ok)
		}
	}

	if v := panics(func() { ch <- 2 }); v == nil {
		t.Error("send on closed channel: want panic")
	} else {
		t.Logf("send on closed panics: %v", v)
	}
	if v := panics(func() { close(ch) }); v == nil {
		t.Error("close of closed channel: want panic")
	} else {
		t.Logf("double close panics: %v", v)
	}
}

func TestCloseWakesAllReceivers(t *testing.T) {
	ch := make(chan struct{})
	const n = 5
	woke := make(chan struct{}, n)
	for range n {
		go func() {
			<-ch
			woke <- struct{}{}
		}()
	}
	close(ch) // broadcast: one close releases every receiver
	for i := range n {
		select {
		case <-woke:
		case <-time.After(time.Second):
			t.Fatalf("only %d of %d receivers woke up", i, n)
		}
	}
}

func TestRangeEndsOnClose(t *testing.T) {
	ch := make(chan int)
	go func() {
		defer close(ch) // without close, range below would block forever
		for i := range 3 {
			ch <- i
		}
	}()
	var sum int
	for v := range ch {
		sum += v
	}
	if sum != 0+1+2 {
		t.Errorf("sum = %d", sum)
	}
}
