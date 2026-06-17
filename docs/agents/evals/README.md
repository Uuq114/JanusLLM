# Evaluations

This directory stores contract test plans, benchmark notes, fixture catalogs,
and evaluation conclusions for agent/loop engineering work.

Recommended document types:

- Contract test plans: scope, expected behavior, fixtures, and commands.
- Fixture catalogs: request, upstream response, expected gateway response, and
  expected spend extraction for each provider scenario.
- Test and evaluation protocols: validation levels, fixture requirements, live
  API safety, and closeout rules.
- Benchmark notes: workload, environment, metrics, and conclusions.
- Eval conclusions: what passed, what failed, residual risks, and follow-up
  loops.

Do not store raw provider dumps here. Put raw logs and scratch files in
`.agent-artifacts/`, then summarize durable findings in Markdown.

Start with `test-and-eval-protocol.md` before turning fixture plans into
executable tests.
