# Fixture Catalog: Reasoning Model Gateway Compatibility

Last updated: 2026-06-16

This catalog defines planned fixtures only. It is not an executable test suite
yet and does not include live provider output. Values are synthetic unless a
future run links them to captured official examples.

## Fixture Shape

Each executable fixture should eventually include:

- Logical client request.
- Expected upstream request after Janus model rewrite and defaults.
- Upstream response or SSE stream.
- Expected client response or stream.
- Expected spend extraction.
- Source documentation link and date reviewed.

## 1. DeepSeek V4 Pro Chat Reasoning, Non-Stream

Client request:

```json
{
  "model": "deepseek-v4",
  "messages": [
    {"role": "user", "content": "Return a one-line answer."}
  ],
  "stream": false,
  "thinking": {"type": "enabled"},
  "reasoning_effort": "high"
}
```

Expected upstream request:

```json
{
  "model": "deepseek-v4-pro",
  "messages": [
    {"role": "user", "content": "Return a one-line answer."}
  ],
  "stream": false,
  "thinking": {"type": "enabled"},
  "reasoning_effort": "high"
}
```

Upstream response template:

```json
{
  "id": "chatcmpl-deepseek-nonstream",
  "object": "chat.completion",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "reasoning_content": "synthetic hidden reasoning summary placeholder",
        "content": "Done."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 8,
    "total_tokens": 20
  }
}
```

Expected spend extraction:

```json
{
  "id": "chatcmpl-deepseek-nonstream",
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 8,
    "total_tokens": 20
  }
}
```

## 2. DeepSeek V4 Flash Chat Reasoning, Stream

Client request:

```json
{
  "model": "deepseek-v4-flash-group",
  "messages": [
    {"role": "user", "content": "Stream a concise answer."}
  ],
  "stream": true,
  "thinking": {"type": "enabled"},
  "reasoning_effort": "medium",
  "stream_options": {"include_usage": true}
}
```

Expected SSE upstream stream:

```text
data: {"id":"chatcmpl-deepseek-stream","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-deepseek-stream","choices":[{"index":0,"delta":{"reasoning_content":"synthetic reasoning chunk"},"finish_reason":null}]}

data: {"id":"chatcmpl-deepseek-stream","choices":[{"index":0,"delta":{"content":"Done"},"finish_reason":null}]}

data: {"id":"chatcmpl-deepseek-stream","choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}

data: [DONE]
```

Expected contract:

- All SSE lines are forwarded in order.
- `delta.reasoning_content` is not removed or renamed.
- Spend usage is extracted from the final usage chunk.

## 3. GLM-5.2 Chat Reasoning, Non-Stream

Status: Needs confirmation for exact public model ID and request field shape.

Client request template:

```json
{
  "model": "glm-5.2-group",
  "messages": [
    {"role": "user", "content": "Answer with one sentence."}
  ],
  "stream": false,
  "thinking": {"type": "enabled"}
}
```

Expected upstream request template:

```json
{
  "model": "glm-5.2",
  "messages": [
    {"role": "user", "content": "Answer with one sentence."}
  ],
  "stream": false,
  "thinking": {"type": "enabled"}
}
```

Expected response fields:

```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "reasoning_content": "synthetic reasoning placeholder",
        "content": "Done."
      }
    }
  ],
  "usage": {
    "prompt_tokens": 11,
    "completion_tokens": 7,
    "total_tokens": 18
  }
}
```

## 4. MiniMax M2.7 Chat Reasoning, Stream

Client request:

```json
{
  "model": "minimax-m2.7-group",
  "messages": [
    {"role": "user", "content": "Stream a short answer."}
  ],
  "stream": true,
  "reasoning_split": true,
  "stream_options": {"include_usage": true}
}
```

Expected upstream request:

```json
{
  "model": "MiniMax-M2.7",
  "messages": [
    {"role": "user", "content": "Stream a short answer."}
  ],
  "stream": true,
  "reasoning_split": true,
  "stream_options": {"include_usage": true}
}
```

Expected SSE upstream stream:

```text
data: {"id":"chatcmpl-minimax-stream","choices":[{"index":0,"delta":{"reasoning_content":"synthetic reasoning"},"finish_reason":null}]}

data: {"id":"chatcmpl-minimax-stream","choices":[{"index":0,"delta":{"content":"Done."},"finish_reason":"stop"}],"usage":{"prompt_tokens":9,"completion_tokens":6,"total_tokens":15}}

data: [DONE]
```

Expected contract:

- `reasoning_split` remains in the upstream request.
- `reasoning_content` and any `reasoning_details` objects are forwarded.
- Base usage is parsed; detail usage remains unaccounted.

## 5. Qwen3.6-30B Chat Thinking, Stream

Status: Needs confirmation for exact public model ID.

Client request template:

```json
{
  "model": "qwen3.6-30b-group",
  "messages": [
    {"role": "user", "content": "Stream a short answer."}
  ],
  "stream": true,
  "enable_thinking": true,
  "thinking_budget": 1024,
  "stream_options": {"include_usage": true}
}
```

Expected upstream request template:

```json
{
  "model": "qwen3.6-30b",
  "messages": [
    {"role": "user", "content": "Stream a short answer."}
  ],
  "stream": true,
  "enable_thinking": true,
  "thinking_budget": 1024,
  "stream_options": {"include_usage": true}
}
```

Expected SSE upstream stream:

```text
data: {"id":"chatcmpl-qwen-stream","choices":[{"index":0,"delta":{"reasoning_content":"synthetic reasoning"},"finish_reason":null}]}

data: {"id":"chatcmpl-qwen-stream","choices":[{"index":0,"delta":{"content":"Done."},"finish_reason":"stop"}]}

data: {"id":"chatcmpl-qwen-stream","choices":[],"usage":{"prompt_tokens":13,"completion_tokens":9,"total_tokens":22,"completion_tokens_details":{"reasoning_tokens":4}}}

data: [DONE]
```

Expected contract:

- Reasoning deltas are forwarded.
- Base usage is parsed into spend.
- `completion_tokens_details.reasoning_tokens` remains in the client stream but
  is not currently included in spend accounting.

## 6. Qwen Responses API Reasoning Stream

Status: Expected current gateway gap because `/v1/responses` is not registered.

Client request template:

```json
{
  "model": "qwen-responses-capable-model",
  "input": "Use reasoning and answer briefly.",
  "reasoning": {"effort": "medium"},
  "stream": true
}
```

Expected current Janus behavior:

```text
HTTP 404 or route-not-found behavior for /v1/responses
```

Future upstream event stream template:

```text
event: response.output_item.added
data: {"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_1"}}

event: response.reasoning_summary_text.delta
data: {"type":"response.reasoning_summary_text.delta","delta":"synthetic reasoning summary"}

event: response.output_text.delta
data: {"type":"response.output_text.delta","delta":"Done."}

event: response.completed
data: {"type":"response.completed","response":{"id":"resp_qwen","usage":{"input_tokens":8,"output_tokens":6,"total_tokens":14,"output_tokens_details":{"reasoning_tokens":3}}}}
```

Future contract:

- Decide whether Janus should raw-pass-through Responses events or normalize
  them.
- Add Responses-specific usage parsing before billing depends on it.
