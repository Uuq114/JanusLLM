# Multi-Agent Collaboration Protocol

Last updated: 2026-06-16

This protocol defines how multiple agents should cooperate on JanusLLM loop
engineering work. It is a process document only; it does not grant permission to
change business code.

## Roles

| Role | Purpose | Write access | Required output |
| --- | --- | --- | --- |
| Coordinator | Owns loop intake, task split, run records, integration, and final status. | `docs/agents/` and integration branch only. | Updated state, run record, merged conclusions, final risk list. |
| Explorer | Performs read-only code, docs, provider API, or architecture audits. | None unless explicitly authorized. | Findings with file paths, source links, assumptions, and gaps. |
| Worker | Implements a bounded change in an assigned branch and file scope. | Assigned files only. | Patch, tests, validation output, and changed-file list. |
| Reviewer | Reviews patches, tests, risks, and missed edge cases. | None by default. | Findings ordered by severity with file/line references. |
| Eval Agent | Runs or designs fixtures, contract tests, benchmarks, and eval summaries. | `docs/agents/evals/` or assigned test files only. | Test plan, fixture updates, commands, pass/fail notes, residual risk. |

## Loop Flow

1. Coordinator records the loop goal in `docs/agents/state.md` and creates or
   updates a run record under `docs/agents/runs/`.
2. Coordinator splits work into bounded tasks with explicit scope, expected
   output, validation method, and ownership.
3. Explorers gather evidence before workers make code changes.
4. Workers operate on their own branches and assigned files.
5. Eval Agent verifies fixture and test expectations independently where
   practical.
6. Reviewer inspects the integrated patch and calls out bugs, missing tests, and
   behavior risks.
7. Coordinator updates state, backlog, run record, eval notes, and decisions
   before closing the loop.

## Coordination Rules

- One coordinator owns a loop at a time.
- Do not assign the same write scope to multiple workers unless the coordinator
  explicitly serializes the handoff.
- Workers must assume other agents may also be editing the repository and must
  not revert changes they did not make.
- Explorers and reviewers should stay read-only unless the coordinator grants a
  specific write scope.
- All claims that affect implementation should link to code paths, provider
  docs, fixture examples, test output, or ADRs.
- Unknown provider behavior must be marked `TODO`, `Needs confirmation`, or
  `Expected gap`; do not convert it into an implementation assumption.

## Handoff Checklist

Every agent handoff should include:

- Branch name and base commit if code changed.
- Files read or changed.
- Commands run and their result.
- Open assumptions.
- Remaining risks.
- Suggested next step.

## Conflict Handling

- If two agents produce overlapping edits, the coordinator decides which branch
  becomes the integration source.
- The losing branch may be cherry-picked, manually ported, or discarded only
  after its useful findings are captured in the run record.
- Do not use destructive git commands to resolve conflicts unless the user
  explicitly requests them.
