package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	api "github.com/elsant/llm-gateway/internal/anthropic"
	"github.com/elsant/llm-gateway/internal/gateway"
	"github.com/elsant/llm-gateway/internal/provider/anthropic"
)

// Compile-time checks that the providers satisfy gateway.Provider.
var (
	_ gateway.Provider = (*anthropic.Client)(nil)
	_ gateway.Provider = echoProvider{}
)

const usage = `usage: gateway [flags] "<prompt>"

Sends a prompt to an LLM provider and prints the response text.
The anthropic provider (default) requires the ANTHROPIC_API_KEY environment
variable; the echo provider prints the prompt back and needs no key.

flags:
`

func main() {
	cfg := defaultConfig()
	providerName := flag.String("provider", "anthropic", "provider to use: anthropic or echo")
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

	provider, err := newProvider(*providerName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	resp, err := provider.Complete(context.Background(), newRequest(cfg, flag.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", describeError(err))
		os.Exit(1)
	}

	fmt.Println(resp.Text)
}

// newProvider returns the provider selected by -provider.
func newProvider(name string) (gateway.Provider, error) {
	switch name {
	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, errors.New("ANTHROPIC_API_KEY is not set\n" +
				"hint: create a key at https://console.anthropic.com/settings/keys and run\n" +
				"      export ANTHROPIC_API_KEY=sk-ant-...")
		}
		return anthropic.New(apiKey), nil
	case "echo":
		return echoProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown -provider %q (want anthropic or echo)", name)
	}
}

// describeError turns an error from Provider.Complete into a message for the user.
// API errors get a short explanation of the common cases; anything else
// (network failure, timeout, bad JSON) is printed as is.
func describeError(err error) string {
	var apiErr *api.APIError
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
