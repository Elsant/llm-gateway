package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: ANTHROPIC_API_KEY is not set")
		os.Exit(1)
	}

	if len(os.Args) < 2 || strings.TrimSpace(os.Args[1]) == "" {
		fmt.Fprintln(os.Stderr, `usage: gateway "<prompt>"`)
		os.Exit(1)
	}
	prompt := os.Args[1]

	client := NewClient(apiKey)
	resp, err := client.Send(context.Background(), newMessageRequest(prompt))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println(responseText(resp))
}

// responseText joins the text of all "text" content blocks in resp.
func responseText(resp *anthropic.MessageResponse) string {
	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}
