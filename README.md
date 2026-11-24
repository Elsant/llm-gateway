# llm-gateway

[![CI](https://github.com/Elsant/llm-gateway/actions/workflows/ci.yml/badge.svg)](https://github.com/Elsant/llm-gateway/actions/workflows/ci.yml)

A gateway service for LLM providers, written in Go.

> **Status:** early development. This repository is built in public as part of my
> 8-week challenge of moving from Ruby on Rails to Go and Python. The project grows
> a little every day; see the commit history.

## Goal

A single service that sits between applications and LLM providers (Anthropic first,
others later) and handles the things every team ends up rebuilding:

- unified HTTP API with streaming responses (SSE)
- per-client API keys, rate limits and token budgets
- cost and usage tracking in Postgres
- caching, retries, circuit breaking and provider fallback
- observability: structured logs, Prometheus metrics, OpenTelemetry traces
- gRPC API and an MCP server for agents

## Roadmap

- [x] Week 1: Go module, CLI client for the Claude API on plain `net/http`
- [x] Week 2: `Provider` interface, typed errors, table-driven tests, CI
- [ ] Week 3: concurrency: hedged requests, cancellation, streaming, rate limiting
- [ ] Week 4: HTTP server, Postgres, API keys, request log
- [ ] Week 5: logs, metrics, tracing, graceful shutdown, Docker
- [ ] Week 6: benchmarks, profiling, performance work
- [ ] Week 7: gRPC, queues, MCP server, tool use

## Requirements

- Go 1.27+

## Install

```bash
go install github.com/elsant/llm-gateway/cmd/gateway@latest
```

The binary lands in `$(go env GOPATH)/bin` (make sure it is on your `PATH`).
From a local checkout: `go install ./cmd/gateway`.

## Configuration

The CLI reads your API key from the environment:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

Create a key at <https://console.anthropic.com/settings/keys>.

## Usage

```bash
gateway [flags] "<prompt>"
```

| Flag          | Default           | Description                         |
|---------------|-------------------|-------------------------------------|
| `-model`      | `claude-opus-5-5` | Claude model ID                     |
| `-max-tokens` | `1024`            | maximum number of tokens to generate |
| `-system`     | empty (none)      | system prompt                       |

Flags go before the prompt:

```bash
gateway "What is a goroutine?"
gateway -model claude-haiku-5-5 -max-tokens 256 -system "Answer in one sentence." "What is a goroutine?"
```

Errors go to stderr with exit code 1:

- `ANTHROPIC_API_KEY` not set: a hint on how to set it;
- 401 (`gateway.ErrUnauthorized`): invalid API key;
- 429 (`gateway.ErrRateLimited`): rate limit exceeded, wait and retry;
- other 4xx: request rejected;
- 5xx: provider unavailable;

each followed by the HTTP status, `error.type` and `error.message` from the API.

## Project layout

```
cmd/gateway/                     CLI entrypoint: wires config, provider and output
internal/config/                 flags + environment -> Config
internal/gateway/                Provider interface, Request/Response, APIError and sentinel errors
internal/provider/anthropic/     Anthropic client implementing gateway.Provider
internal/provider/anthropic/api/ Messages API wire types
internal/provider/echo/          fake provider that echoes the prompt (tests, -provider=echo)
examples/fetchall/                learning experiment: concurrent fetch with goroutines + WaitGroup
```

`internal/gateway` is the consumer-side contract; everything under
`internal/provider/` implements it. Provider packages never import each other.

## Development

```bash
go test -race -cover ./...        # unit tests (httptest fake server, no API key needed)
golangci-lint run                 # config in .golangci.yml
go test -fuzz=FuzzDecodeResponse -fuzztime=30s ./internal/provider/anthropic
```

CI (GitHub Actions) runs `go vet`, golangci-lint, tests with `-race` and a
short fuzz pass on every push.

## License

MIT

