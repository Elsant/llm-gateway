// Package config builds the CLI configuration from command-line arguments
// and environment variables.
//
// Load takes its inputs as parameters (args, getenv) instead of reading
// os.Args and os.Getenv directly, so it can be tested without touching the
// real process state.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

const (
	// DefaultProvider is the provider used when -provider is not given.
	DefaultProvider = "anthropic"

	// DefaultModel is the current default Claude model (verified against the
	// Claude API docs, 2026-11). Use the bare ID, never a date-suffixed one.
	DefaultModel = "claude-opus-5-5"

	// DefaultMaxTokens is the default -max-tokens value: enough for a short
	// CLI answer. Raise it with the flag for longer outputs.
	DefaultMaxTokens = 1024

	// EnvAnthropicAPIKey is the environment variable holding the Anthropic key.
	EnvAnthropicAPIKey = "ANTHROPIC_API_KEY"
)

const usage = `usage: gateway [flags] "<prompt>"

Sends a prompt to an LLM provider and prints the response text.
The anthropic provider (default) requires the ANTHROPIC_API_KEY environment
variable; the echo provider prints the prompt back and needs no key.

flags:
`

// Config is everything the CLI needs for one run.
type Config struct {
	Provider  string // provider name, e.g. "anthropic" or "echo"
	Model     string
	MaxTokens int
	System    string // empty = no system prompt
	Prompt    string
	Stream    bool // print the response word by word as it arrives

	// AnthropicAPIKey may be empty: only the anthropic provider needs it, so
	// the check belongs to whoever builds that provider.
	AnthropicAPIKey string
}

// UsageError reports invalid command-line usage (unknown flag, missing
// prompt, bad flag value). The CLI prints it with the usage text and exits
// with status 2.
type UsageError struct {
	msg string
}

func (e *UsageError) Error() string { return e.msg }

func usageErrorf(format string, args ...any) *UsageError {
	return &UsageError{msg: fmt.Sprintf(format, args...)}
}

// Load parses args (without the program name) and reads secrets through
// getenv. It returns flag.ErrHelp for -h/-help and a *UsageError for invalid
// usage. Load prints nothing; use PrintUsage to show help.
func Load(args []string, getenv func(string) string) (Config, error) {
	cfg := Config{
		Provider:  DefaultProvider,
		Model:     DefaultModel,
		MaxTokens: DefaultMaxTokens,
	}
	fs := newFlagSet(&cfg)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, err
		}
		return Config{}, usageErrorf("%v", err)
	}

	if fs.NArg() != 1 || strings.TrimSpace(fs.Arg(0)) == "" {
		return Config{}, usageErrorf("expected exactly one non-empty prompt argument, got %d", fs.NArg())
	}
	if cfg.MaxTokens <= 0 {
		return Config{}, usageErrorf("-max-tokens must be positive, got %d", cfg.MaxTokens)
	}

	cfg.Prompt = fs.Arg(0)
	cfg.AnthropicAPIKey = getenv(EnvAnthropicAPIKey)
	return cfg, nil
}

// PrintUsage writes the usage text and the flag defaults to w.
func PrintUsage(w io.Writer) {
	cfg := Config{Provider: DefaultProvider, Model: DefaultModel, MaxTokens: DefaultMaxTokens}
	fs := newFlagSet(&cfg)
	fs.SetOutput(w)
	fmt.Fprint(w, usage)
	fs.PrintDefaults()
}

// newFlagSet defines the CLI flags, bound to the fields of cfg. Its current
// field values become the flag defaults. The flag set prints nothing on its
// own: errors are returned to the caller.
func newFlagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("gateway", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Provider, "provider", cfg.Provider, "provider to use: anthropic or echo")
	fs.StringVar(&cfg.Model, "model", cfg.Model, "model ID")
	fs.IntVar(&cfg.MaxTokens, "max-tokens", cfg.MaxTokens, "maximum number of tokens to generate")
	fs.StringVar(&cfg.System, "system", cfg.System, "system prompt (empty = none)")
	fs.BoolVar(&cfg.Stream, "stream", cfg.Stream, "print the response word by word")
	return fs
}
