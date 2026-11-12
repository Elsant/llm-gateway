package main

import (
	"errors"
	"testing"
)

func TestBuildRequest_EmptyPrompt(t *testing.T) {
	for _, prompt := range []string{"", "   ", "\n\t "} {
		body, err := buildRequest(prompt)
		if !errors.Is(err, errEmptyPrompt) {
			t.Errorf("buildRequest(%q) err = %v, want errEmptyPrompt", prompt, err)
		}
		if body != nil {
			t.Errorf("buildRequest(%q) body = %s, want nil", prompt, body)
		}
	}
}

func TestBuildRequest_JSON(t *testing.T) {
	body, err := buildRequest("Hi")
	if err != nil {
		t.Fatal(err)
	}
	want := `{"model":"claude-opus-5-5","max_tokens":16000,"messages":[{"role":"user","content":[{"type":"text","text":"Hi"}]}]}`
	if string(body) != want {
		t.Errorf("got  %s\nwant %s", body, want)
	}
}
