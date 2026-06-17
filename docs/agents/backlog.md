# Agent Loop Backlog

Last updated: 2026-06-16

| ID | Loop | Expected outcome | Validation method | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| LOOP-001 | Gateway capability audit for new reasoning models | A reviewed capability matrix covering provider API surfaces, request fields, response fields, streaming, usage, and Janus gaps. | Compare entries against official provider docs and code references from the current gateway audit. | In progress | Current loop. |
| LOOP-002 | Contract fixture implementation | Executable tests that prove request rewrite, field pass-through, SSE forwarding, usage extraction, and current `/v1/responses` behavior. | `go test ./internal/proxy ./internal/request` with provider fixture cases. | TODO | Do not start until fixture catalog is reviewed. |
| LOOP-003 | Responses API decision | A decision on whether Janus should support `/v1/responses` as pass-through, translated endpoint, or out of scope. | ADR accepted and linked to tests or implementation plan. | TODO | Depends on LOOP-001 and LOOP-002. |
| LOOP-004 | Reasoning usage accounting | A clear billing stance for reasoning tokens, cached tokens, and provider token detail fields. | Eval notes plus spend schema/test plan review. | TODO | May require DB schema and pricing decisions. |
