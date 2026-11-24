package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchAll_Concurrent(t *testing.T) {
	const (
		n     = 20
		delay = 100 * time.Millisecond
	)
	// Every request sleeps for delay. Sequentially 20 of them would take
	// 2s; concurrently the whole batch should take about one delay.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	urls := make([]string, n)
	for i := range urls {
		urls[i] = fmt.Sprintf("%s/page/%d", srv.URL, i)
	}

	start := time.Now()
	results := fetchAll(context.Background(), srv.Client(), urls)
	elapsed := time.Since(start)

	if len(results) != n {
		t.Fatalf("got %d results, want %d", len(results), n)
	}
	for i, r := range results {
		if r.Err != nil || r.Status != http.StatusOK {
			t.Errorf("result %d: status=%d err=%v", i, r.Status, r.Err)
		}
		if r.URL != urls[i] {
			t.Errorf("result %d is for %s, want %s (order must match input)", i, r.URL, urls[i])
		}
		if want := int64(len(fmt.Sprintf("/page/%d", i))); r.Bytes != want {
			t.Errorf("result %d: %d bytes, want %d", i, r.Bytes, want)
		}
		if r.Duration < delay {
			t.Errorf("result %d: duration %s, want >= %s", i, r.Duration, delay)
		}
	}
	if limit := n * delay / 4; elapsed > limit {
		t.Errorf("took %s for %d requests of %s each: not concurrent (limit %s)", elapsed, n, delay, limit)
	}
}

func TestFetchAll_ErrorsDoNotStopOthers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)

	urls := []string{srv.URL, "http://127.0.0.1:1", "::not a url"}
	results := fetchAll(context.Background(), srv.Client(), urls)

	if results[0].Err != nil || results[0].Status != http.StatusTeapot {
		t.Errorf("ok server: status=%d err=%v", results[0].Status, results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("unreachable host: want error")
	}
	if results[2].Err == nil {
		t.Error("invalid URL: want error")
	}
}

func TestReadLines(t *testing.T) {
	in := "https://a\n\n  # comment\n  https://b  \n"
	got := readLines(strings.NewReader(in))
	want := []string{"https://a", "https://b"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
