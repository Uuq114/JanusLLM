# Agent Git Workflow

Last updated: 2026-06-16

This workflow keeps multi-agent work reviewable and prevents agents from
overwriting each other.

## Branch Model

- The shared integration branch should use the repository's agent prefix:
  `codex/<topic>`.
- Each development worker should branch from the current integration baseline:
  `codex/<loop-id>-<agent-role>-<short-task>`.
- Explorers and reviewers should not create branches unless they are asked to
  produce patches.
- A worker branch must have one primary owner and a bounded file scope.

Example:

```text
codex/loop-002-worker-proxy-contract-tests
codex/loop-002-worker-fixtures
codex/loop-002-reviewer-contract-risk
```

## Commit Rules

- Keep commits scoped to one loop or one worker task.
- Commit messages should state the loop or task, for example:
  `docs: add agent loop protocols`
- Do not commit provider API keys, raw provider dumps, local config, build
  outputs, or `.agent-artifacts/`.
- Do not amend or force-push a branch that another agent or user may be using
  unless the coordinator explicitly approves it.

## Staging Rules

- Stage only files that belong to the current loop.
- Before staging, inspect `git status --short` and avoid unrelated user changes.
- If unrelated changes exist, leave them unstaged and mention them in the run
  record or final handoff only if they affect the loop.
- Use `.agent-artifacts/` for scratch outputs; it is intentionally ignored.

## Merge And Integration

- Worker branches should be integrated by the coordinator, not by peer workers.
- Coordinator should review worker diffs before merge or cherry-pick.
- If conflict resolution changes behavior, document the resolution in the run
  record.
- After integration, run the validation commands defined in the loop intake or
  explain why they were skipped.

## Prohibited By Default

- `git reset --hard`
- `git checkout -- <path>` to discard another user's work
- Force-pushing shared branches
- Bulk formatting unrelated files
- Moving generated or scratch artifacts into tracked docs without summarizing
  them

## Closeout Checklist

Before a loop branch is committed or pushed:

- `git status --short` reviewed.
- Relevant docs under `docs/agents/` updated.
- Validation commands run or explicitly skipped with reason.
- Changed files listed in the final handoff.
- Commit and push target branch confirmed.
