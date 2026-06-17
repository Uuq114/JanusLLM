# ADR 0001: Agent Loop Documentation Structure

Date: 2026-06-16

Status: Accepted

## Context

JanusLLM already has `doc/` for product planning, progress, and architecture
assets. The project also needs durable agent/loop engineering notes that do not
pollute business code directories or conflict with existing product docs.

## Decision

Use `docs/agents/` as the dedicated location for agent state, loop records,
capability matrices, decisions, and evaluation planning. Keep temporary logs and
raw outputs in `.agent-artifacts/`, which is ignored by git.

Create a root `AGENTS.md` that points coding agents to this structure.

## Consequences

- Agent progress is reviewable in Markdown without mixing with source code.
- Existing `doc/` product material remains untouched.
- Raw artifacts stay local unless intentionally summarized into tracked docs.
- Future loops should update `docs/agents/state.md`, `backlog.md`, and a run
  record instead of relying only on chat history.

## Links

- `docs/agents/README.md`
- `docs/agents/runs/2026-06-16-model-capability-audit.md`
