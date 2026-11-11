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

- [ ] Week 1: Go module, CLI client for the Claude API on plain `net/http`
- [ ] Week 2: `Provider` interface, typed errors, table-driven tests, CI
- [ ] Week 3: concurrency: hedged requests, cancellation, streaming, rate limiting
- [ ] Week 4: HTTP server, Postgres, API keys, request log
- [ ] Week 5: logs, metrics, tracing, graceful shutdown, Docker
- [ ] Week 6: benchmarks, profiling, performance work
- [ ] Week 7: gRPC, queues, MCP server, tool use

## Requirements

- Go 1.27+

## Run

```bash
go run ./cmd/gateway
```

## Project layout

```
cmd/gateway/         entrypoint
internal/anthropic/   Messages API wire types
```

## License

MIT

