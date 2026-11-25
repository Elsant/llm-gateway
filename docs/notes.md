# Notes

## Day 2: Messages API types (now `internal/provider/anthropic/api/types.go`)

Source of truth: the HTTP API reference (https://docs.anthropic.com/en/api/messages), not the SDK.

### Not supported yet
- `content` as a plain string (`"content": "Hello"`). Only the array of blocks is supported.
- `system` as an array of text blocks. Only the string form is supported.
- Content blocks other than `text` (image, tool_use, tool_result, thinking, ...).
- `usage.cache_creation_input_tokens` / `cache_read_input_tokens`.
- Request params: top_p, top_k, tools, tool_choice, thinking, ...

### omitempty: where it helps, where it hurts
omitempty drops a field when it holds the type's zero value. That's only safe
when the zero value means the same as "not sent".

| Field | Tag | Why |
|---|---|---|
| model, max_tokens, messages | no omitempty | Required. If one is empty, the API should reject it with an error instead of the field silently disappearing. |
| system | omitempty | `""` means "no system prompt", same as not sending it. |
| temperature | `*float64` + omitempty | `0` is a real value (deterministic sampling). A plain `float64` + omitempty would drop 0, and the API would fall back to its default of 1.0. A pointer makes nil mean "not set" and `&0.0` send `0`. |
| stop_sequences | omitempty | Both nil and an empty slice mean "none". |
| stream | omitempty | `false` is the API default. |
| metadata | `*Metadata` + omitempty | omitempty never drops a non-pointer struct. Use a pointer (or the `omitzero` tag, Go 1.24+). |

General rule: when the zero value means something different from "absent", use
a pointer (or `omitzero` with an `IsZero()` method).

### Nullable fields in the response
- `stop_sequence` is `null` unless `stop_reason == "stop_sequence"` → `*string`.
  `null` decodes to `nil`. A plain `string` would turn `null` into `""`, and you
  could no longer tell "no stop sequence" from "matched an empty string".
- `stop_reason` is always set in non-streaming responses (it is `null` only in the
  streaming `message_start` event), so it stays a `string` for now. When we add
  streaming, revisit this.
- Decoding: `null` into a non-pointer field is a no-op (the field keeps its zero
  value), not an error.

## Day 3: buildRequest (`cmd/gateway/request.go`)

### Defaults
- `defaultModel = "claude-opus-5-5"`: the current default model per the Claude API
  docs (checked 2026-11). Use the bare ID; never append a date suffix.
- `defaultMaxTokens = 16000`: recommended for non-streaming requests. Too low and
  the answer is cut off mid-thought (`stop_reason: "max_tokens"`); much higher
  risks HTTP timeouts. Streaming requests can go up to the model's 128K cap.

### Sampling params are rejected on claude-opus-5-5
`temperature`, `top_p` and `top_k` are removed on Claude Opus 5.5: sending any of
them returns **400**. The Day 2 `main.go` sent `temperature: 0` with this model,
so that request would have failed. `buildRequest` leaves `Temperature` nil, and
`*float64` + omitempty drops the key from the JSON.
The field stays in `MessageRequest` for older models, but don't set it for
`claude-opus-5-5`. (From API version `2026-10-21` these params return 400 on
every model.)

### Errors
- First real error in the project: `errEmptyPrompt` for an empty or
  whitespace-only prompt (`strings.TrimSpace`). It's a package-level
  `errors.New` value, so callers can check it with `errors.Is`.
- Marshal errors are wrapped with `fmt.Errorf("build request: %w", err)`, so the
  cause stays reachable through `errors.Is`/`errors.As`.

### Shadowing
`if _, err := f(); err != nil` declares a new `err` that hides the outer one.
When `err` already exists, reuse it: `_, err = f()`. Check with:

    go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
    go vet -vettool=$(go env GOPATH)/bin/shadow -strict ./...

## Week 2 (days 8-12): what landed

- `gateway.Provider` interface, declared by the consumer (`internal/gateway`),
  two implementations: `provider/anthropic` and `provider/echo`.
- Typed errors: `*gateway.APIError` + `ErrUnauthorized` / `ErrRateLimited`
  matched through a custom `Is` method.
- Layout: `cmd/gateway`, `internal/config`, `internal/gateway`,
  `internal/provider/...`. No `utils`, no import cycles.
- Tests: table-driven, `httptest` fake server, `t.Helper` / `t.Cleanup`,
  fuzz tests for response and error parsing. golangci-lint + GitHub Actions.

## Day 13: idiomatic review

What the review found and how it was fixed (commit `e38c676`):

| Finding | Why it is not idiomatic | Fix |
|---|---|---|
| `main` called `os.Exit` in four places | `os.Exit` skips deferred calls, and `main` can't be tested | `run(ctx, args, getenv, stdout, stderr) int`; `main` only wires `os.*` and exits |
| Ctrl+C killed the process mid-request | `context.Background()` is never cancelled | `signal.NotifyContext(ctx, os.Interrupt)` |
| Error prefixes `send request:`, `decode response:`, `anthropic api:` mixed | The caller can't tell which provider failed | Wrap once at the package boundary: `anthropic: ...` |
| Header names in lower case | `Header.Set` canonicalizes anyway; source should show the real key | `X-Api-Key`, `Anthropic-Version`, `Content-Type` |
| One test table with a `wantOK` switch | Two kinds of cases in one table, half the fields unused per row | `TestComplete_Success` + `TestComplete_Errors` |

Rule of thumb that came out of it: **everything that touches the process
(`os.Args`, `os.Getenv`, `os.Stdout`, `os.Exit`, signals) lives in `main`;
everything else takes those as parameters.** Same idea as `config.Load(args, getenv)`.

## Day 14: errors and interfaces through a Rubyist's eyes

### Interfaces

| Ruby | Go |
|---|---|
| Duck typing at runtime: `respond_to?(:call)` or just call it and rescue `NoMethodError` | Duck typing at compile time: a type satisfies an interface if it has the methods. No `implements` keyword |
| Modules (`include Comparable`) bring behaviour *into* a class | Interfaces bring nothing; they only describe what a caller needs. Code reuse is embedding, not interfaces |
| The library defines the "interface" (base class, module) and you inherit from it | The **consumer** declares the interface it needs. `gateway.Provider` lives next to the code that calls `Complete`, not in `provider/anthropic` |
| Big classes with many methods are normal (`ActiveRecord::Base`) | Small interfaces win: `io.Reader` has one method. "The bigger the interface, the weaker the abstraction" |

Things that surprised me:
- **Accept interfaces, return structs.** `anthropic.New` returns `*Client`,
  not `gateway.Provider`. Callers get the interface implicitly when they need
  it, and keep access to everything else.
- **`var _ gateway.Provider = (*Client)(nil)`** is the only way to say
  "this type implements that interface" — a compile-time assertion, not a
  declaration. In Ruby the closest thing is a spec.
- **Nil interface vs interface holding a nil pointer.** A `Provider` holding
  `(*Client)(nil)` is *not* `== nil`. Ruby has one `nil`; Go has typed nils.
- Faking a dependency needs no mocking library: `echo.Provider` is a
  ten-line struct. In RSpec I'd reach for `instance_double`.

### Errors

| Ruby | Go |
|---|---|
| `raise` unwinds the stack until some `rescue` catches it | An error is a return value. Every call site decides: handle, wrap, or return |
| Exception class hierarchy: `rescue Net::HTTPError` | Sentinels (`ErrRateLimited`) compared with `errors.Is`; types (`*APIError`) extracted with `errors.As` |
| `raise MyError, "msg"` with a `cause` set implicitly | `fmt.Errorf("anthropic: %w", err)` — wrapping is explicit, `%w` keeps the chain |
| `retry`, `ensure` | No retry keyword; `defer` instead of `ensure` |
| Exceptions for control flow are common (`ActiveRecord::RecordNotFound` -> 404) | `panic` is for bugs only; "not found" is an ordinary error value |

Things that surprised me:
- **`errors.Is` vs `errors.As`.** `Is` asks "is this *that* error?" (a value,
  walks the chain comparing). `As` asks "is there an error of *this type* in
  the chain? give it to me" (fills a pointer). `*APIError` implements a custom
  `Is`, so a 429 `APIError` *is* `ErrRateLimited` without being the same value.
- **Wrap once per boundary.** Adding context at every level gives
  `anthropic: send request: Post "...": dial tcp: ...` — readable, but only if
  each layer adds something new.
- **The happy path is on the left.** `if err != nil { return ... }` early,
  no `else`. After a week it reads faster than nested `begin/rescue`.
- Error strings are lower case, no trailing punctuation or newlines
  (staticcheck ST1005). The multi-line "hint" had to move out of the error
  into `main`.

### Self-check for the end of week 2
- [ ] Explain `errors.Is` vs `errors.As` without notes.
- [ ] Explain why the interface lives with the consumer.
- [ ] From scratch, no hints: an interface, two implementations, a
      table-driven test.
- [x] CI green.

## Day 15: goroutines and WaitGroup (`examples/fetchall`)

Week 3 starts. No gateway changes today (theory + experiments); the
experiment is `examples/fetchall`: fetch N URLs concurrently, print the time
of each request.

### Concurrency is not parallelism
- **Concurrency** is about *structure*: the program is split into
  independently running pieces (goroutines) that can make progress
  interleaved.
- **Parallelism** is about *execution*: pieces actually run at the same
  instant on different cores.
- A concurrent program on one core is still concurrent, just not parallel.
  `fetchall` is mostly waiting on the network, so it gains from concurrency
  even with `GOMAXPROCS=1`.
- Ruby comparison: MRI threads are concurrent but, because of the GVL, not
  parallel for Ruby code. They still help for I/O — the same reason
  `fetchall` is fast. Goroutines are concurrent *and* parallel, and cost
  ~2 KB of stack instead of an OS thread.

### What `fetchall` shows
- 20 requests × 100 ms finish in ~100 ms wall time, not 2 s (checked in
  `TestFetchAll_Concurrent`).
- **Results without a mutex**: allocate `results := make([]result, n)` up
  front; goroutine `i` writes only `results[i]`. Different slice elements
  are different memory locations, so there is no data race; `wg.Wait()`
  makes all writes visible to the caller. `go test -race` agrees.
- **Order is preserved for free** by indexing, no sorting afterwards.
- **One failure doesn't stop the others**: each goroutine records its own
  error. (Stopping everything on the first error is `errgroup` + context —
  later this week.)

### WaitGroup rules
- `wg.Add` must happen *before* `go`, not inside the goroutine, or `Wait`
  may return before the goroutine has registered.
- Go 1.25+: `wg.Go(func() { ... })` does `Add(1)`, `go` and `defer Done()` in
  one call — the first two rules can't be broken by accident.
- Go 1.22+: loop variables are per iteration, so capturing `i` and `url` in
  the closure is safe. Before 1.22 every goroutine could see the last value.
- A WaitGroup must not be copied after first use (pass `*sync.WaitGroup`);
  `go vet` (copylocks) catches it.

## Day 16: channels and select

### Project: `gateway.StreamWords`
`StreamWords(ctx, p, req) (<-chan string, error)` calls `Complete` and hands
the text out word by word: a fake stream, so the CLI (`-stream`) and future
callers can be written against a channel now, and real SSE streaming slots
in later this week.

Design points:
- **Return `<-chan string`, not `chan string`.** The receive-only type means
  the caller can't send into it or close it; only the producer closes.
- **The producer closes** (`defer close(out)`), so `for chunk := range ch`
  ends by itself.
- **Every send is inside `select` with `<-ctx.Done()`.** Otherwise a reader
  that stops reading leaves the goroutine blocked on `out <- chunk` forever:
  a goroutine leak. `TestStreamWords_CancelStopsSender` checks the channel
  closes after cancel.
- **`select` is random when several cases are ready**, so after cancel the
  send could still win again and again. Checking `ctx.Err()` at the top of
  each iteration caps it at one word in flight.
- Errors that happen before streaming starts are returned directly; the
  channel only carries data. (Mid-stream errors will need a different shape —
  e.g. a `Chunk{Text, Err}` struct — when streaming is real.)

### Channel edge cases (`examples/channels`, each row is a test)

| operation | nil channel | open channel | closed channel |
|---|---|---|---|
| send `ch <- v` | blocks forever | blocks until receiver / buffer space | **panic** |
| receive `<-ch` | blocks forever | blocks until a value | buffered values first, then zero value + `ok=false` immediately |
| `close(ch)` | **panic** | OK, wakes *all* receivers | **panic** |
| `len` / `cap` | 0 / 0 | queued / buffer size | remaining / buffer size |

Takeaways:
- Close is a **broadcast**: one `close` releases every receiver. Send
  wakes exactly one.
- A nil channel in a `select` case disables that case — handy for turning
  inputs off once they're drained.
- Only the sender should close, and only once. With several senders, close
  after a `WaitGroup` says they're all done.
- Ruby comparison: closest is `Queue` + `Queue#close` (Ruby 2.3+), but `pop`
  on a closed empty Ruby `Queue` returns `nil`, not a separate `ok` flag, and
  there is no `select` over several queues.

## Day 17: sync, data races, memory model

### Project: `usage.Counter`
Per-client totals of requests, input and output tokens
(`internal/usage`). `gateway.Response` now carries `Usage` (Anthropic: from
`usage.input_tokens/output_tokens`; echo: word count), so there is
something to count. Wiring it into requests waits for the HTTP server and
per-client API keys in week 4.

- `sync.Mutex` + `map[string]Stats`, every method locks for its whole body
  (`Lock(); defer Unlock()`).
- **Zero value is usable**: the map is created lazily inside `Record`, so
  `var c usage.Counter` works without a constructor — same as
  `sync.Mutex`, `bytes.Buffer`.
- **Map values are not addressable**: `c.clients[k].Requests++` does not
  compile. Read the struct, change it, write it back — all under the lock.
- **`Snapshot` returns `maps.Clone`**, never the internal map: once the map
  leaves the lock, nothing protects it.
- A struct with a `Mutex` must not be copied after use (the copy gets its
  own, unrelated lock). `go vet` copylocks catches it; pass `*Counter`.
- `BenchmarkCounter_RecordParallel`: ~135 ns/op on 10 cores. Fine for now;
  if the lock ever becomes hot, shard by client or use `sync/atomic`
  counters per client.

### Race detector experiment
Removed the `Lock/Unlock` from `Record` and ran
`go test -race -run Concurrent ./internal/usage`:

- 7 × `WARNING: DATA RACE`, each with the two conflicting stacks (who wrote,
  who read, which goroutines);
- then the runtime itself: `fatal error: concurrent map read and map write`.

### Why racy code often "works"
- The race detector reports races that **actually happened during this
  run** (it instruments memory accesses and tracks happens-before between
  goroutines). It does not prove the code is race-free; untested paths
  stay unchecked. No false positives, but false negatives are possible.
- Without `-race`, a racy `count++` (load, add, store) mostly gives the
  right answer when goroutines rarely overlap: low load, one core, short
  critical section. It fails under real concurrency — in production.
- A data race is **undefined behaviour** in the Go memory model, not just
  "a lost increment": the compiler and CPU may reorder or cache accesses.
  Maps are worse — the runtime detects concurrent writes and crashes on
  purpose.
- Happens-before is what makes a write visible to another goroutine:
  `Unlock` → next `Lock`, a channel send → its receive, `close` → a receive
  that sees it, `wg.Done` → `wg.Wait` returning. No such edge = no
  guarantee.

### Mutex vs atomic vs channel
- One integer counter: `atomic.Int64` — no lock, can't forget to unlock.
- Several fields updated together, or a map: `Mutex` — atomics can't make
  "requests and tokens change together" atomic.
- Handing data from one goroutine to another: channel. "Share memory by
  communicating" is about ownership transfer, not about counters.
- Ruby comparison: MRI's GVL hides many races in pure Ruby code (but not
  all — `@count += 1` across threads is still racy). Go has no such net:
  `-race` in CI is the net, which is why tests run with `-race`.
