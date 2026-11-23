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
	"io"
	"os"
	"os/signal"

	"github.com/elsant/llm-gateway/internal/config"
	"github.com/elsant/llm-gateway/internal/gateway"
	"github.com/elsant/llm-gateway/internal/provider/anthropic"
	"github.com/elsant/llm-gateway/internal/provider/echo"
)

// Exit codes.
const (
	exitOK    = 0
	exitError = 1 // runtime failure: missing key, provider error
	exitUsage = 2 // bad command line, as with the flag package
)

func main() {
	// Ctrl+C cancels the in-flight request instead of killing the process
	// mid-write; a second Ctrl+C kills it as usual.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// run is the whole CLI behind main. It takes its inputs and outputs as
// parameters and returns the exit code instead of calling os.Exit, so tests
// can drive it end to end.
func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	cfg, err := config.Load(args, getenv)
	var usageErr *config.UsageError
	switch {
	case errors.Is(err, flag.ErrHelp):
		config.PrintUsage(stdout)
		return exitOK
	case errors.As(err, &usageErr):
		fmt.Fprintln(stderr, "error:", err)
		config.PrintUsage(stderr)
		return exitUsage
	case err != nil:
		fmt.Fprintln(stderr, "error:", err)
		return exitError
	}

	provider, err := newProvider(cfg)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		if errors.Is(err, errNoAPIKey) {
			fmt.Fprintln(stderr, "hint: create a key at https://console.anthropic.com/settings/keys and run")
			fmt.Fprintln(stderr, "      export "+config.EnvAnthropicAPIKey+"=sk-ant-...")
		}
		return exitError
	}

	resp, err := provider.Complete(ctx, newRequest(cfg))
	if err != nil {
		fmt.Fprintln(stderr, "error:", describeError(err))
		return exitError
	}

	fmt.Fprintln(stdout, resp.Text)
	return exitOK
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
	if errors.Is(err, context.Canceled) {
		return "interrupted"
	}

	var apiErr *gateway.APIError
	if !errors.As(err, &apiErr) {
		return err.Error()
	}

	var hint string
	switch {
	case errors.Is(err, gateway.ErrUnauthorized):
		hint = "invalid API key, check " + config.EnvAnthropicAPIKey
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
