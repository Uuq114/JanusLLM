# Model Capability Matrix

Last updated: 2026-06-16

This matrix is a planning artifact for JanusLLM gateway contracts. It records
provider/API capability facts, current Janus fit, and gaps that should become
fixtures or contract tests before implementation work starts.

## Source Notes

- DeepSeek official docs: `https://api-docs.deepseek.com/api/create-chat-completion`
  and `https://api-docs.deepseek.com/api/anthropic-api`.
- Z.AI official docs: `https://docs.z.ai/api-reference/llm/chat-completion`,
  `https://docs.z.ai/guides/capabilities/thinking-mode`, and
  `https://docs.z.ai/guides/capabilities/streaming`.
- MiniMax official docs: `https://platform.minimax.io/docs/api-reference/text-chat-openai`,
  `https://platform.minimax.io/docs/api-reference/anthropic-chat-completion`,
  and `https://platform.minimax.io/docs/guides/m2_guide`.
- Alibaba Cloud Model Studio official docs:
  `https://www.alibabacloud.com/help/en/model-studio/qwen-api-reference/`,
  `https://www.alibabacloud.com/help/en/model-studio/deep-thinking`,
  and `https://www.alibabacloud.com/help/en/model-studio/compatibility-with-openai-responses-api`.

Some model names requested for audit are newer than the visible API reference
examples. Entries marked `Needs confirmation` should not become hard-coded
contract assertions until the exact public API SKU and field shape are confirmed.

## Current Janus Gateway Baseline

| Capability | Current implementation | Code reference | Gap for this loop |
| --- | --- | --- | --- |
| Chat completions | Supported through native proxy. | `cmd/main.go` `run()` registers `/v1/chat/completions`. | Add provider reasoning fixtures. |
| Text completions | Supported through the same proxy path. | `cmd/main.go` `run()` registers `/v1/completions`. | Not the primary target for reasoning fixtures. |
| Embeddings | Supported through the same proxy path. | `cmd/main.go` `run()` registers `/v1/embeddings`. | Out of current reasoning scope. |
| Anthropic messages | Supported as pass-through, not OpenAI-to-Anthropic translation. | `cmd/main.go` `run()` and `internal/proxy/adapter.go` `AnthropicAdapter.BuildRequest`. | Add explicit pass-through expectations if provider exposes Anthropic-compatible APIs. |
| Responses API | Not supported. | No `/v1/responses` route; `requiresModel()` does not include it. | Plan expected-gap fixtures before implementation. |
| Request model rewrite | Supported for top-level `model`. | `internal/proxy/proxy.go` `prepareUpstreamBody`. | Verify unknown provider fields survive JSON rewrite. |
| Request defaults | Supported by merging `model_group.request_defaults`. | `internal/proxy/proxy.go` `prepareUpstreamBody`. | Verify reasoning defaults can be injected without overwriting caller values. |
| Query string pass-through | Not preserved. | `OpenAIAdapter.BuildRequest` builds URL from `endpointPath` only. | Add a negative contract if provider options require query params. |
| Streaming pass-through | Supported for `stream=true` plus `text/event-stream`. | `internal/proxy/proxy.go` `shouldStream` and `streamToClient`. | Verify `delta.reasoning_content` is forwarded unchanged. |
| Non-stream usage | Basic OpenAI and Anthropic token usage. | `internal/proxy/adapter.go` `buildSpendPayloadFromEnvelope`. | Token detail fields are ignored. |
| Streaming usage | Parses SSE `data:` JSON usage when present. | `internal/proxy/adapter.go` `parseSpendStreamLine`. | Providers without final usage still skip spend. |
| `reasoning_content` | No special adapter logic. | No `reasoning_content` references in current source. | It is forwarded as ordinary JSON/SSE content only. |

## Provider Matrix

| Provider/model target | Official model ID status | Chat/completions | Responses API | Streaming | Reasoning request fields to fixture | Reasoning response fields to fixture | Usage fields to fixture | Current Janus fit/gap |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| DeepSeek V4 Flash | Official docs show `deepseek-v4-flash`. | OpenAI-compatible Chat Completions. | Not confirmed in DeepSeek docs. | SSE streaming; final usage can be requested with `stream_options.include_usage`. | `thinking` and `reasoning_effort` for thinking models. | `message.reasoning_content` and `delta.reasoning_content`; request messages should not include prior `reasoning_content`. | `usage.prompt_tokens`, `usage.completion_tokens`, `usage.total_tokens`; token details need confirmation. | Janus should pass fields through after model rewrite. Usage parsing covers base fields. No Responses endpoint and no reasoning-specific accounting. |
| DeepSeek V4 Pro | Official docs show `deepseek-v4-pro`. | OpenAI-compatible Chat Completions. | Not confirmed in DeepSeek docs. | SSE streaming; include final usage with stream options. | `thinking` and `reasoning_effort`; pro model may use stronger default reasoning. | Same as Flash: non-stream message and stream delta reasoning content. | Same base OpenAI usage fields. | Same as Flash. Add separate fixtures because upstream model IDs and defaults may differ. |
| GLM-5.2 | Needs confirmation: Z.AI docs expose GLM-5 family APIs and mention GLM-5.2 in newer material, but the exact public chat model ID should be verified before executable tests. | Z.AI Chat Completion API is OpenAI-compatible at `/paas/v4/chat/completions`. | Not confirmed. | Z.AI streaming is supported; usage appears in final stream chunks depending on API mode. | Thinking mode docs describe a `thinking` object with enabled/disabled modes. Exact GLM-5.2 allowed values need confirmation. | Z.AI reasoning output uses `reasoning_content` in assistant messages and streaming deltas for thinking mode. | OpenAI-style base usage; token detail fields need confirmation. | Janus pass-through should preserve `thinking` and `reasoning_content`, but no provider-specific validation, no Responses support, and no reasoning token accounting. |
| MiniMax M2.7 | Official docs use `MiniMax-M2.7` and `MiniMax-M2.7-highspeed`; lowercase `minimax-m2.7` should be treated as a logical/local alias until provider casing is confirmed. | MiniMax OpenAI-compatible Chat Completions are supported. | Not confirmed. | SSE streaming is supported. | M2 guide documents M2.7 thinking behavior and `reasoning_split`; disabling thinking is not a reliable contract for M2.x. | `reasoning_content` and `reasoning_details` when split reasoning is enabled. | OpenAI-style usage plus possible token detail fields such as cached tokens. | Janus should preserve unknown fields and stream deltas, but usage parser ignores token details and there is no MiniMax-specific adapter. |
| Qwen3.6-30B | Needs confirmation: no official Alibaba/Qwen API page was found for exact `qwen3.6-30b` during this audit. Treat it as a requested target placeholder until the public SKU is confirmed. | Alibaba Cloud Model Studio supports Qwen through OpenAI-compatible Chat Completions. | Alibaba Cloud Model Studio documents OpenAI-compatible Responses API. | Chat streaming is supported; usage can be emitted with `stream_options.include_usage`. Responses streaming uses event-style output. | Qwen thinking docs use `enable_thinking` and thinking-budget controls for supported models. Exact field placement depends on SDK/API mode and needs fixture confirmation. | Chat output uses `reasoning_content` for thinking traces; Responses output uses reasoning items/events rather than only chat deltas. | Base usage plus token detail fields such as reasoning tokens in Responses-style usage. | Janus can likely pass chat fields through, but `/v1/responses` is absent and Responses usage/events are not parsed. Exact model ID must be verified. |

## Contract Focus

The first executable test loop should prove these minimum contracts:

1. Top-level `model` is rewritten to the upstream model while provider-specific
   reasoning request fields remain unchanged.
2. Non-stream chat responses preserve `message.reasoning_content` for clients.
3. Streaming chat responses preserve `delta.reasoning_content` lines in order.
4. Basic `usage` is extracted for spend records when providers emit OpenAI-style
   token fields.
5. Token detail fields remain present in client responses even when Janus does
   not yet account for them.
6. `/v1/responses` is documented as an expected gap until a route and adapter
   contract are accepted.
