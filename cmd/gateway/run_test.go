package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	noEnv := func(string) string { return "" }

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string // exact
		wantStderr string // substring; empty = stderr must be empty
	}{
		{
			name:       "echo provider prints the prompt",
			args:       []string{"-provider", "echo", "привет"},
			wantCode:   exitOK,
			wantStdout: "привет\n",
		},
		{
			name:       "help goes to stdout",
			args:       []string{"-h"},
			wantCode:   exitOK,
			wantStdout: "", // checked separately below
		},
		{
			name:       "missing prompt is a usage error",
			args:       []string{"-provider", "echo"},
			wantCode:   exitUsage,
			wantStderr: "error: expected exactly one non-empty prompt argument",
		},
		{
			name:       "unknown provider",
			args:       []string{"-provider", "nope", "hi"},
			wantCode:   exitError,
			wantStderr: `unknown -provider "nope"`,
		},
		{
			name:       "anthropic without key prints a hint",
			args:       []string{"hi"},
			wantCode:   exitError,
			wantStderr: "hint: create a key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tt.args, noEnv, &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if tt.args[0] == "-h" {
				if !strings.HasPrefix(stdout.String(), "usage: gateway") {
					t.Errorf("stdout = %q, want usage text", stdout.String())
				}
				return
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr == "" && stderr.Len() > 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestRun_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // as if the user pressed Ctrl+C before the reply arrived

	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{"-provider", "echo", "hi"}, func(string) string { return "" }, &stdout, &stderr)
	if code != exitError {
		t.Errorf("exit code = %d, want %d", code, exitError)
	}
	if got := stderr.String(); got != "error: interrupted\n" {
		t.Errorf("stderr = %q, want %q", got, "error: interrupted\n")
	}
}
