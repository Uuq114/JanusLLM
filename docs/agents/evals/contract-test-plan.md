# Contract Test Plan: Reasoning Model Gateway Compatibility

Last updated: 2026-06-16

## Objective

Plan, but do not yet implement, contract tests for reasoning-capable models and
provider APIs in the JanusLLM gateway.

The tests should cover DeepSeek V4 Flash/Pro, GLM-5.2, MiniMax M2.7, and
Qwen3.6-30B or their confirmed official provider model IDs.

## Non-Goals For This Loop

- No production code changes.
- No Go test files added yet.
- No provider credentials or live API calls.
- No hard-coded assertions for model IDs marked `Needs confirmation` in the
  capability matrix.

## Test Surfaces

| Surface | Contract | Expected current status |
| --- | --- | --- |
| Endpoint routing | `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings`, and `/v1/messages` route to the proxy. `/v1/responses` remains absent. | Existing support for the first four; `/v1/responses` expected gap. |
| Request extraction | `model` and bool `stream` are extracted from the raw body. | Existing support. |
| Model rewrite | Logical model group names are replaced with upstream model names. | Existing support. |
| Provider field preservation | Provider-specific fields such as `thinking`, `reasoning_effort`, `enable_thinking`, `reasoning_split`, and `stream_options` survive body preparation. | Expected to pass because unknown fields are preserved through JSON map rewrite. |
| Request defaults | Missing reasoning/default fields can be injected from `request_defaults` without overwriting caller fields. | Existing generic merge, needs fixtures. |
| Non-stream response | `message.reasoning_content`, `reasoning_details`, token details, and ordinary content are returned unchanged to the client. | Expected pass-through, needs fixtures. |
| Streaming response | SSE lines containing `delta.reasoning_content` are forwarded in order and final usage is parsed when present. | Expected pass-through, needs fixtures. |
| Usage extraction | Basic `prompt_tokens`, `completion_tokens`, `total_tokens`, `input_tokens`, and `output_tokens` are normalized for spend. | Existing support. |
| Usage details | Reasoning tokens, cached tokens, and provider detail objects remain in client responses but are not billed separately. | Expected gap for accounting. |
| Query string | Query parameters are not forwarded to upstream providers. | Existing gap; add a negative contract if needed. |

## Planned Fixture Groups

| Fixture ID | Provider/model | Endpoint | Mode | Purpose | Expected gateway status |
| --- | --- | --- | --- | --- | --- |
| `deepseek-v4-chat-reasoning-nonstream` | `deepseek-v4-pro` | `/v1/chat/completions` | Non-stream | Preserve thinking request fields and `message.reasoning_content`; parse base usage. | Pass. |
| `deepseek-v4-chat-reasoning-stream` | `deepseek-v4-flash` | `/v1/chat/completions` | SSE | Preserve `delta.reasoning_content`; parse final stream usage. | Pass if upstream includes final usage. |
| `glm-5.2-chat-reasoning-nonstream` | `glm-5.2` pending confirmation | `/v1/chat/completions` | Non-stream | Preserve Z.AI thinking fields and reasoning output. | Planned after SKU confirmation. |
| `minimax-m2.7-chat-reasoning-stream` | `MiniMax-M2.7` | `/v1/chat/completions` | SSE | Preserve `reasoning_split`, `reasoning_content`, and `reasoning_details`. | Pass-through expected; detail accounting gap. |
| `qwen3.6-30b-chat-enable-thinking-stream` | `qwen3.6-30b` pending confirmation | `/v1/chat/completions` | SSE | Preserve `enable_thinking`, thinking budget fields, reasoning deltas, and final usage. | Planned after SKU confirmation. |
| `qwen3.6-30b-responses-reasoning-stream` | Qwen confirmed Responses-capable model | `/v1/responses` | Event stream | Document expected Janus gap for Responses API routing and usage parsing. | Expected fail/gap until implemented. |

## Suggested Implementation Location

After this plan is reviewed, add executable fixtures under a test-only path such
as:

- `internal/proxy/testdata/contract/*.json`
- `internal/proxy/*_test.go`
- `internal/request/*_test.go`

Do not place fixture data in production config or provider-specific business
code.

## Validation Commands For Future Loop

```bash
go test ./internal/request ./internal/proxy
```

If `/v1/responses` is implemented later, extend the command to include the new
adapter/request packages that own Responses parsing.

## Open Questions

- What exact public model IDs should Janus use for `glm-5.2` and
  `qwen3.6-30b`?
- Should Janus support `/v1/responses` as raw pass-through first, or should it
  define a normalized adapter contract before routing?
- Should reasoning tokens and cached tokens be stored in the spend schema, or
  only preserved in client responses?
- Should Janus allow provider-specific reasoning defaults in
  `request_defaults`, or introduce an explicit model capability schema?
