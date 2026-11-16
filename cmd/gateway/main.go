package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/elsant/llm-gateway/internal/anthropic"
)

const usage = `usage: gateway [flags] "<prompt>"

Sends a prompt to the Claude Messages API and prints the response text.
Requires the ANTHROPIC_API_KEY environment variable.

flags:
`

func main() {
	cfg := defaultConfig()
	flag.StringVar(&cfg.Model, "model", cfg.Model, "Claude model ID")
	flag.IntVar(&cfg.MaxTokens, "max-tokens", cfg.MaxTokens, "maximum number of tokens to generate")
	flag.StringVar(&cfg.System, "system", "", "system prompt (empty = none)")
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 || strings.TrimSpace(flag.Arg(0)) == "" {
		flag.Usage()
		os.Exit(2)
	}
	if cfg.MaxTokens <= 0 {
		fmt.Fprintln(os.Stderr, "error: -max-tokens must be positive")
		os.Exit(2)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: ANTHROPIC_API_KEY is not set")
		fmt.Fprintln(os.Stderr, "hint: create a key at https://console.anthropic.com/settings/keys and run")
		fmt.Fprintln(os.Stderr, "      export ANTHROPIC_API_KEY=sk-ant-...")
		os.Exit(1)
	}

	client := NewClient(apiKey)
	resp, err := client.Send(context.Background(), newMessageRequest(cfg, flag.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", describeError(err))
		os.Exit(1)
	}

	fmt.Println(responseText(resp))
}

// describeError turns an error from Client.Send into a message for the user.
// API errors get a short explanation of the common cases; anything else
// (network failure, timeout, bad JSON) is printed as is.
func describeError(err error) string {
	var apiErr *anthropic.APIError
	if !errors.As(err, &apiErr) {
		return err.Error()
	}

	var hint string
	switch {
	case apiErr.Status >= 500:
		hint = "provider unavailable, try again later"
	case apiErr.Status == http.StatusUnauthorized:
		hint = "invalid API key, check ANTHROPIC_API_KEY"
	case apiErr.Status == http.StatusTooManyRequests:
		hint = "rate limit exceeded, wait and retry"
	default:
		hint = "request rejected"
	}
	return fmt.Sprintf("%s (%s)", hint, apiErr)
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
