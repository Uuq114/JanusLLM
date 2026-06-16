# Test And Evaluation Protocol

Last updated: 2026-06-16

This protocol defines how JanusLLM loop work should be validated. It applies to
documentation-only loops, fixture loops, contract-test loops, and future
implementation loops.

## Validation Levels

| Level | Purpose | Example | Required when |
| --- | --- | --- | --- |
| Docs check | Verify Markdown structure, links, and gitignore rules. | `git diff --check`, file existence checks. | Documentation-only loops. |
| Static/code audit | Confirm current behavior without changing code. | `rg`, file reads, function references. | Capability or design loops. |
| Unit test | Verify a small function or package behavior. | `go test ./internal/request`. | Function-level behavior changes. |
| Contract test | Verify gateway request/response behavior from fixtures. | Proxy tests with synthetic upstream servers and fixture bodies. | Provider/API compatibility changes. |
| Golden fixture test | Ensure serialized request, response, SSE, and usage outputs stay stable. | Testdata compare for chat and streaming payloads. | Adapter and compatibility changes. |
| Live provider smoke test | Verify real provider compatibility with controlled credentials. | One gated request against an upstream API. | Only after contract tests pass and credentials are supplied. |
| Benchmark/load test | Measure latency, throughput, error rate, or billing overhead. | Gateway load baseline. | Performance-sensitive loops. |

## Fixture Requirements

Contract fixtures should include:

- Client request.
- Expected upstream request after Janus model rewrite and defaults.
- Upstream non-stream response or SSE event stream.
- Expected client response or stream.
- Expected spend extraction.
- Expected gap or expected failure when behavior is not implemented.
- Source documentation URL and review date when based on provider docs.

## Reasoning Model Coverage

Reasoning model fixtures should cover:

- Provider-specific request fields such as `thinking`, `reasoning_effort`,
  `enable_thinking`, `thinking_budget`, `reasoning_split`, and
  `stream_options.include_usage`.
- Non-stream `message.reasoning_content`.
- Streaming `delta.reasoning_content`.
- Provider detail objects such as `reasoning_details`.
- Base usage fields used by current spend accounting.
- Token detail fields that should be preserved for clients even if Janus does
  not bill them yet.
- `/v1/responses` expected-gap behavior until the endpoint is implemented.

## Live API Safety

- Live provider tests must be opt-in and gated by environment variables.
- Never commit provider API keys, raw live responses, or customer prompts.
- Store raw live output in `.agent-artifacts/` only.
- Summarize durable findings in `docs/agents/evals/`.
- Redact request IDs, account IDs, keys, and any sensitive content before adding
  evidence to tracked Markdown.

## Loop Closeout Requirements

Each run record must state:

- Commands or checks run.
- Checks intentionally skipped and why.
- Whether failures are expected gaps or regressions.
- What fixtures or tests need to be added next.
- Residual risk after validation.

## Documentation-Only Loops

For loops that intentionally avoid local development environment setup, the
minimum validation is:

```bash
git diff --check
git diff --cached --check
git status --short
```

Do not run build, unit, integration, or live provider tests unless the user asks
for them or the loop explicitly requires them.

## Implementation Loops

Before changing production code:

1. Record the expected behavior in `docs/agents/evals/fixtures.md` or a
   dedicated fixture file.
2. Decide whether the current behavior should pass, fail, or be marked as an
   expected gap.
3. Add executable tests for accepted behavior.
4. Implement the smallest code change that satisfies the tests.
5. Update the run record with validation output and remaining risk.
