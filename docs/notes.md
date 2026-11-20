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
