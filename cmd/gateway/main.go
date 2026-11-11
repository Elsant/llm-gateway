package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

func main() {
	temperature := 0.0

	req := anthropic.MessageRequest{
		Model:     "claude-opus-5-5",
		MaxTokens: 1024,
		Messages: []anthropic.Message{
			{
				Role:    anthropic.RoleUser,
				Content: []anthropic.ContentBlock{anthropic.NewTextBlock("Hello, Claude")},
			},
		},
		System:      "You are a helpful assistant.",
		Temperature: &temperature,
	}

	out, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		log.Fatalf("marshal request: %v", err)
	}
	fmt.Println(string(out))
}
