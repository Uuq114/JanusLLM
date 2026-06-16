# Model Capability Audit Run

Date: 2026-06-16

## Goal

Set up durable agent/loop engineering documentation, then prepare a model
capability matrix, fixture catalog, and contract test plan for new reasoning
models in the JanusLLM gateway.

## Scope

- Documentation only.
- No business code changes.
- No executable tests added in this loop.

## Code Audit Findings

- `/v1/chat/completions`, `/v1/completions`, `/v1/embeddings`, and
  `/v1/messages` are registered in `cmd/main.go`.
- `/v1/responses` is not registered and is not included in `requiresModel()`.
- `prepareUpstreamBody()` rewrites the top-level `model` and merges request
  defaults while preserving unknown JSON fields through unmarshal/marshal.
- OpenAI-compatible providers use `OpenAIAdapter`; Anthropic-compatible
  providers use `AnthropicAdapter`.
- Streaming forwards SSE lines to the client and opportunistically parses usage.
- Basic usage extraction supports OpenAI-style `prompt_tokens`,
  `completion_tokens`, `total_tokens` and Anthropic-style `input_tokens`,
  `output_tokens`.
- There is no dedicated `reasoning_content` handling or reasoning-token
  accounting.

## Documentation Changes

- Created root `AGENTS.md`.
- Added `.agent-artifacts/` to `.gitignore`.
- Created `docs/agents/README.md`, `state.md`, `backlog.md`, and
  `capability-matrix.md`.
- Created run, decision, and eval documentation subdirectories with Markdown
  templates and planning files.
- Created `docs/agents/evals/contract-test-plan.md` and
  `docs/agents/evals/fixtures.md`.
- Added `docs/agents/collaboration.md` for multi-agent roles, handoffs, and
  coordinator rules.
- Added `docs/agents/git-workflow.md` for branch, staging, commit, integration,
  and closeout rules.
- Added `docs/agents/evals/test-and-eval-protocol.md` for validation levels,
  fixture requirements, live API safety, and documentation-only loop checks.

## Validation

- `git diff --check` passed with no whitespace errors.
- Confirmed all required Markdown files and directories exist with `Test-Path`.
- Confirmed `.gitignore` contains `.agent-artifacts/` with `Select-String`.
- Confirmed `.agent-artifacts/` exists locally and is ignored by git.
- `go test ./...` passed.
- Staged the new and modified Markdown/gitignore files for git tracking.
- For the later protocol-only update, planned validation is limited to
  `git diff --check`, `git diff --cached --check`, and `git status --short`;
  build, unit, integration, and live provider checks are intentionally skipped.
- Confirmed the three protocol files exist and are linked from agent entry
  documents.

## Risks

- `glm-5.2` and `qwen3.6-30b` exact public API model IDs still need official
  confirmation before they are used as hard assertions.
- Provider docs may change quickly; contract fixtures should store the exact
  provider doc URL and date reviewed when implemented.
- Current pass-through uses JSON re-encoding and does not preserve query strings,
  so some provider-specific edge cases may need explicit tests.

## Next Steps

- Review `docs/agents/capability-matrix.md` and `docs/agents/evals/fixtures.md`.
- Convert approved fixture definitions into Go tests.
- Write an ADR for the `/v1/responses` support decision before implementing it.
