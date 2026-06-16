# Agent Loop Engineering

Last updated: 2026-06-16

This directory is the durable workspace for agent and loop engineering progress.
It is separate from the existing `doc/` product planning material so agent
status, run records, and evaluation planning do not pollute business code or
product documentation.

## Structure

- `state.md` records the current goal, completed work, work in progress,
  blockers, validation status, and suggested next loop.
- `backlog.md` records pending loops. Each item must include the expected
  outcome and a validation method before it is scheduled.
- `capability-matrix.md` records model and API capabilities across providers,
  including request and response adaptation requirements.
- `collaboration.md` defines coordinator, explorer, worker, reviewer, and eval
  agent responsibilities.
- `git-workflow.md` defines branch, staging, commit, integration, and closeout
  rules for single-agent and multi-agent loops.
- `runs/` contains one Markdown file per loop. Each run record should cover the
  goal, changes, validation, findings, risks, and next steps.
- `decisions/` contains long-lived design decisions in ADR style.
- `evals/` contains contract test plans, benchmark notes, fixture definitions,
  and evaluation conclusions.

## Rules

- Write durable agent documentation in Markdown and keep it tracked by git.
- Do not invent progress. If a detail is unknown or not yet verified, write
  `TODO` or `Needs confirmation`.
- Link claims to code paths, run records, official provider docs, or validation
  output whenever practical.
- Keep raw command output, provider dumps, and scratch files in
  `.agent-artifacts/`; that directory is intentionally gitignored.
- Prefer updating the current run record during a loop instead of spreading
  status across chat-only notes.
- Keep mandatory short-form agent rules in the repository root `AGENTS.md`; keep
  detailed process documents in this directory.
- For multi-agent work, use one coordinator and separate worker branches unless
  the user asks for a different workflow.
- For documentation-only loops, run docs/git checks only unless extra validation
  is explicitly required.

## Naming

- Run records: `runs/YYYY-MM-DD-short-topic.md`
- ADRs: `decisions/NNNN-short-title.md`
- Eval plans: `evals/short-topic.md`
- Fixture catalogs: `evals/fixtures.md` or `evals/fixtures/*.md`

## Core Protocols

- `collaboration.md`
- `git-workflow.md`
- `evals/test-and-eval-protocol.md`
