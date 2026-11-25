package config

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

// fakeEnv returns a getenv func backed by a map.
func fakeEnv(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoad(t *testing.T) {
	env := fakeEnv(map[string]string{EnvAnthropicAPIKey: "sk-test"})

	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "defaults",
			args: []string{"hello"},
			want: Config{
				Provider: DefaultProvider, Model: DefaultModel, MaxTokens: DefaultMaxTokens,
				Prompt: "hello", AnthropicAPIKey: "sk-test",
			},
		},
		{
			name: "all flags",
			args: []string{"-provider", "echo", "-model", "claude-haiku-5-5", "-max-tokens", "256", "-system", "be brief", "-stream", "hi there"},
			want: Config{
				Provider: "echo", Model: "claude-haiku-5-5", MaxTokens: 256, System: "be brief",
				Prompt: "hi there", Stream: true, AnthropicAPIKey: "sk-test",
			},
		},
		{
			name: "flag=value form",
			args: []string{"-max-tokens=10", "q"},
			want: Config{
				Provider: DefaultProvider, Model: DefaultModel, MaxTokens: 10,
				Prompt: "q", AnthropicAPIKey: "sk-test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args, env)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestLoad_MissingKeyIsNotAnError(t *testing.T) {
	got, err := Load([]string{"-provider", "echo", "hi"}, fakeEnv(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AnthropicAPIKey != "" {
		t.Errorf("AnthropicAPIKey = %q, want empty", got.AnthropicAPIKey)
	}
}

func TestLoad_UsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{"no prompt", nil, "expected exactly one non-empty prompt argument, got 0"},
		{"two prompts", []string{"a", "b"}, "got 2"},
		{"blank prompt", []string{"   "}, "non-empty prompt"},
		{"flag after prompt", []string{"hi", "-model", "x"}, "got 3"},
		{"unknown flag", []string{"-nope", "hi"}, "flag provided but not defined: -nope"},
		{"bad int", []string{"-max-tokens", "lots", "hi"}, `invalid value "lots" for flag -max-tokens`},
		{"zero max tokens", []string{"-max-tokens", "0", "hi"}, "-max-tokens must be positive, got 0"},
		{"negative max tokens", []string{"-max-tokens=-5", "hi"}, "got -5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(tt.args, fakeEnv(nil))
			var usageErr *UsageError
			if !errors.As(err, &usageErr) {
				t.Fatalf("err = %v, want *UsageError", err)
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("err = %q, want it to contain %q", err, tt.wantMsg)
			}
		})
	}
}

func TestLoad_Help(t *testing.T) {
	for _, arg := range []string{"-h", "-help", "--help"} {
		t.Run(arg, func(t *testing.T) {
			_, err := Load([]string{arg}, fakeEnv(nil))
			if !errors.Is(err, flag.ErrHelp) {
				t.Errorf("err = %v, want flag.ErrHelp", err)
			}
		})
	}
}

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	out := buf.String()
	for _, want := range []string{"usage: gateway", "-provider", "-model", DefaultModel, "-max-tokens", "-system", "-stream"} {
		if !strings.Contains(out, want) {
			t.Errorf("usage output missing %q:\n%s", want, out)
		}
	}
}
