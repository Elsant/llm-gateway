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
