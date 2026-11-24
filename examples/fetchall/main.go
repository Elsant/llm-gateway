// Command fetchall fetches URLs concurrently and prints how long each took.
//
// Day 15 experiment (goroutines + sync.WaitGroup), a warm-up for hedged
// requests to several LLM providers in week 3. Not part of the gateway.
//
// Usage:
//
//	go run ./examples/fetchall https://go.dev https://pkg.go.dev ...
//	cat urls.txt | go run ./examples/fetchall
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// result is the outcome of fetching one URL.
type result struct {
	URL      string
	Status   int
	Bytes    int64
	Duration time.Duration
	Err      error
}

func main() {
	urls := os.Args[1:]
	if len(urls) == 0 {
		urls = readLines(os.Stdin)
	}
	if len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "usage: fetchall URL... (or URLs on stdin, one per line)")
		os.Exit(2)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	start := time.Now()
	results := fetchAll(context.Background(), client, urls)
	total := time.Since(start)

	var sum time.Duration
	for _, r := range results {
		sum += r.Duration
		if r.Err != nil {
			fmt.Printf("%8s  ERR  %s: %v\n", r.Duration.Round(time.Millisecond), r.URL, r.Err)
			continue
		}
		fmt.Printf("%8s  %3d  %7d B  %s\n", r.Duration.Round(time.Millisecond), r.Status, r.Bytes, r.URL)
	}
	// Wall time is close to the slowest request, not the sum: that is the
	// whole point of doing it concurrently.
	fmt.Printf("\n%d URLs: wall %s, sum of requests %s\n",
		len(results), total.Round(time.Millisecond), sum.Round(time.Millisecond))
}

// fetchAll fetches every URL in its own goroutine and returns the results in
// the same order as urls.
//
// No mutex or channel is needed: each goroutine writes only results[i], its
// own slot of a slice allocated up front, and wg.Wait() guarantees all
// writes happen before the caller reads the slice.
func fetchAll(ctx context.Context, client *http.Client, urls []string) []result {
	results := make([]result, len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		// Since Go 1.22 the loop variables are per iteration, so i and url
		// can be captured directly. wg.Go (Go 1.25) = wg.Add(1) + go + defer wg.Done().
		wg.Go(func() {
			results[i] = fetch(ctx, client, url)
		})
	}
	wg.Wait()
	return results
}

// fetch performs one GET and measures it, including reading the body.
func fetch(ctx context.Context, client *http.Client, url string) result {
	start := time.Now()
	r := result{URL: url}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		r.Err = err
		return r
	}
	resp, err := client.Do(req)
	if err != nil {
		r.Err = err
		r.Duration = time.Since(start)
		return r
	}
	defer func() { _ = resp.Body.Close() }()

	r.Status = resp.StatusCode
	r.Bytes, r.Err = io.Copy(io.Discard, resp.Body)
	r.Duration = time.Since(start)
	return r
}

// readLines returns the non-empty, non-comment lines of r.
func readLines(r io.Reader) []string {
	var lines []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines
}
