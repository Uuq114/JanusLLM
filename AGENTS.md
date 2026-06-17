# Agent Guidance

This repository keeps agent and loop engineering material under `docs/agents/`.
Do not place agent progress notes, loop run logs, contract-test planning notes, or
raw model capability research in business code directories.

Use:

- `docs/agents/state.md` for the current loop state.
- `docs/agents/backlog.md` for planned loops, expected outcomes, and validation.
- `docs/agents/capability-matrix.md` for model/API capability mapping.
- `docs/agents/collaboration.md` for multi-agent roles and handoffs.
- `docs/agents/git-workflow.md` for branch, staging, commit, and integration
  rules.
- `docs/agents/runs/` for per-loop run records.
- `docs/agents/decisions/` for ADR-style long-term decisions.
- `docs/agents/evals/` for contract tests, benchmarks, fixtures, and eval results.

Keep transient logs, raw outputs, and scratch files in `.agent-artifacts/`. That
directory is ignored by git and should not be used for durable project state.

For multi-agent work, one coordinator should own the loop state and integration.
Workers should use separate task branches with explicit file ownership, while
explorers and reviewers stay read-only unless a write scope is assigned.

For documentation-only loops, do not run build, unit, integration, or live
provider tests unless the loop explicitly requires them. Use the test and eval
protocol in `docs/agents/evals/test-and-eval-protocol.md` to record what was
checked and what was intentionally skipped.
