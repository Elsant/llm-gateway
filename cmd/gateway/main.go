// Command gateway sends a prompt to an LLM provider and prints the reply.
//
// Usage:
//
//	gateway [-provider anthropic|echo] [-model ID] [-max-tokens N] [-system TEXT] "<prompt>"
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/elsant/llm-gateway/internal/config"
	"github.com/elsant/llm-gateway/internal/gateway"
	"github.com/elsant/llm-gateway/internal/provider/anthropic"
	"github.com/elsant/llm-gateway/internal/provider/echo"
)

func main() {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	var usageErr *config.UsageError
	switch {
	case errors.Is(err, flag.ErrHelp):
		config.PrintUsage(os.Stdout)
		return
	case errors.As(err, &usageErr):
		fmt.Fprintln(os.Stderr, "error:", err)
		config.PrintUsage(os.Stderr)
		os.Exit(2)
	case err != nil:
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	provider, err := newProvider(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if errors.Is(err, errNoAPIKey) {
			fmt.Fprintln(os.Stderr, "hint: create a key at https://console.anthropic.com/settings/keys and run")
			fmt.Fprintln(os.Stderr, "      export "+config.EnvAnthropicAPIKey+"=sk-ant-...")
		}
		os.Exit(1)
	}

	resp, err := provider.Complete(context.Background(), newRequest(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", describeError(err))
		os.Exit(1)
	}

	fmt.Println(resp.Text)
}

// errNoAPIKey is returned by newProvider when the anthropic provider is
// selected but no API key is configured.
var errNoAPIKey = errors.New(config.EnvAnthropicAPIKey + " is not set")

// newProvider returns the provider selected by -provider.
func newProvider(cfg config.Config) (gateway.Provider, error) {
	switch cfg.Provider {
	case "anthropic":
		if cfg.AnthropicAPIKey == "" {
			return nil, errNoAPIKey
		}
		return anthropic.New(cfg.AnthropicAPIKey), nil
	case "echo":
		return echo.Provider{}, nil
	default:
		return nil, fmt.Errorf("unknown -provider %q (want anthropic or echo)", cfg.Provider)
	}
}

// describeError turns an error from Provider.Complete into a message for the
// user. Provider API errors get a short explanation; anything else (network
// failure, timeout, bad JSON) is printed as is. The raw API error follows in
// parentheses so the details are never lost.
func describeError(err error) string {
	var apiErr *gateway.APIError
	if !errors.As(err, &apiErr) {
		return err.Error()
	}

	var hint string
	switch {
	case errors.Is(err, gateway.ErrUnauthorized):
		hint = "invalid API key, check ANTHROPIC_API_KEY"
	case errors.Is(err, gateway.ErrRateLimited):
		hint = "rate limit exceeded, wait and retry"
	case apiErr.Status >= 500:
		hint = "provider unavailable, try again later"
	case apiErr.Status >= 400:
		hint = "request rejected, check -model, -max-tokens and the prompt"
	default:
		hint = "unexpected response from provider"
	}
	return fmt.Sprintf("%s (%s)", hint, apiErr)
}
