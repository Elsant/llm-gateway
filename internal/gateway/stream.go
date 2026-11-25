package gateway

import (
	"context"
	"unicode"
	"unicode/utf8"
)

// StreamWords calls p.Complete and delivers the response text on the
// returned channel one word at a time. It is a fake stream: the whole
// response is fetched first. Real token streaming (SSE) replaces it later in
// week 3; callers that range over the channel won't need to change.
//
// An error from Complete is returned directly and no channel is created.
// Otherwise the channel is closed after the last word, or as soon as ctx is
// done, so the sending goroutine never outlives the caller's interest.
// Concatenating every received chunk gives back the full text, including
// whitespace.
func StreamWords(ctx context.Context, p Provider, req Request) (<-chan string, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make(chan string) // unbuffered: each word is handed over when the reader is ready
	go func() {
		defer close(out)
		for _, chunk := range splitWords(resp.Text) {
			// select picks at random when both cases are ready, so check
			// ctx first: after cancel, at most the send already in progress
			// gets through.
			if ctx.Err() != nil {
				return
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				return // reader gave up; don't block forever on out <- chunk
			}
		}
	}()
	return out, nil
}

// splitWords cuts s into chunks of "word + following whitespace". Leading
// whitespace stays with the first chunk, so strings.Join(chunks, "") == s.
func splitWords(s string) []string {
	var chunks []string
	start := 0
	inSpace := true // treat leading whitespace as part of the first chunk
	for i, r := range s {
		space := unicode.IsSpace(r)
		if !space && inSpace && i > start && hasNonSpace(s[start:i]) {
			chunks = append(chunks, s[start:i])
			start = i
		}
		inSpace = space
	}
	if start < len(s) {
		chunks = append(chunks, s[start:])
	}
	return chunks
}

// hasNonSpace reports whether s contains a non-whitespace rune.
func hasNonSpace(s string) bool {
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if !unicode.IsSpace(r) {
			return true
		}
		s = s[size:]
	}
	return false
}
