# llm-gateway

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
- [ ] Week 2: `Provider` interface, typed errors, table-driven tests, CI
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
- 4xx: `error.type` and `error.message` from the API (401 = invalid key, 429 = rate limit);
- 5xx: "provider unavailable" with the HTTP status.

## Project layout

```
cmd/gateway/         entrypoint
internal/anthropic/   Messages API wire types and APIError
```

## License

MIT

