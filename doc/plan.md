# JanusLLM Design Plan

Last updated: 2026-05-31

## 1. Goal

JanusLLM is an LLM gateway for multiple model providers. It provides one API entry point, authorization, routing, billing, admin operations, and future cache/observability features.

Core goals:

- Proxy OpenAI-compatible and Anthropic-compatible APIs.
- Control access by API key, team, and organization.
- Route logical model groups to one or more upstream endpoints.
- Record token usage, spend, provider metadata, latency, cache hit status, and tenant context.
- Keep YAML as the runtime source of truth while syncing model metadata into PostgreSQL for admin and audit views.
- Provide a modern admin console.

## 2. Current Architecture

```text
Client
  |
  v
Gin Gateway
  |-- Auth middleware
  |-- Spend middleware
  |-- Proxy adapters
  |-- Balancer
  v
Provider endpoints

PostgreSQL:
  - organizations, teams, API keys
  - admin users
  - model groups and endpoints
  - spend logs
```

## 3. Configuration Model

- `config/config.yaml` is the runtime authority for model routing.
- `JANUS_DATABASE_URL` and `JANUS_ADMIN_MASTER_KEY` override local YAML secrets.
- Startup sync upserts YAML model groups/endpoints into PostgreSQL.
- Items removed from YAML are marked `enabled=false` in DB instead of being deleted.
- Plain provider `api_key` values are not written to DB. Only `api_key_secret_ref` is synced.
- Proxy registration still uses YAML data directly.

## 4. API Ingress

Implemented:

- `/v1/chat/completions`
- `/v1/completions`
- `/v1/embeddings`
- `/v1/messages`
- `/v1/models`
- OpenAI adapter
- Anthropic adapter
- SSE streaming proxy
- Swagger UI at `/swagger/`

Planned:

- Complete error response standardization.
- Header forwarding allowlist.
- More provider-specific request validation.

## 5. Governance

Implemented:

- API key authentication.
- Key/team model permission intersection.
- Balance and expiration checks.
- Per-key RPM limiting.
- Key cache refresh and idle eviction.
- Admin Basic Auth.
- Organization, team, and key CRUD APIs.
- Auth helper functions that return errors instead of terminating the process.

Planned:

- TPM limiting.
- Daily/monthly token quotas.
- Daily/monthly spend budgets.
- Concurrency limits.
- IP allowlists.
- Platform and tenant admin roles.

## 6. Routing

Implemented:

- Extensible `Balancer` interface with request selection context.
- Round-robin strategy.
- Weighted strategy.
- Latency-based strategy using observed successful request latency.
- Client-sticky strategy using key/team/header/IP identity to improve prefix cache locality.
- Retry/fallback within the selected model group.
- Upstream timeout settings.

Planned:

- Least-inflight strategy.
- Active and passive health checks.
- Circuit breaker and half-open recovery.
- Cross-provider fallback policy.

## 7. Billing And Audit

Implemented:

- Token usage extraction from upstream responses.
- Request spend records in `janus_spend_log`.
- Key balance deduction and total spend update.
- Streaming billing when SSE usage is present.
- Skipping misleading zero-token spend records when usage is missing.
- Metadata fields: `provider`, `latency_ms`, `cache_hit`, `tenant`.

Planned:

- Dashboard query APIs.
- Aggregated spend summaries.
- More detailed provider and tenant reporting.

## 8. Admin Frontend

Implemented:

- `web/` Vite + React + TypeScript project.
- First screen is an admin dashboard, not a landing page.
- Sections for overview metrics, model groups, API keys, usage/spend, and configuration status.
- Mock data and API client helpers for future `/v1/admin/*` and `/v1/models` integration.

Planned:

- Live admin API integration.
- Authentication flow for admin credentials.
- Mutating forms for organizations, teams, keys, and model metadata.

## 9. Cache

Not started:

- L1 exact cache.
- L2 semantic cache.
- L3 prompt fragment cache.
- Janus-managed `x-cache-hit` response header.
- Tenant-isolated cache policy.

The spend schema already includes `cache_hit` so cache rollout can be measured later.

## 10. Deployment And Observability

Not started:

- Prometheus/Grafana metrics.
- OpenTelemetry tracing.
- Load test baseline.
- Container and deployment manifests.
- Rollout and rollback process.

## 11. MVP Boundary

The current MVP supports:

1. OpenAI/Anthropic API proxying.
2. API key and model permission checks.
3. Basic RPM limiting, fallback, and billing.
4. PostgreSQL schema initialization.
5. Admin API and admin dashboard scaffold.
6. Startup YAML-to-DB model config synchronization.

Before production use, prioritize:

1. Live frontend integration.
2. Health checks and circuit breaking.
3. TPM/budget/concurrency limits.
4. Observability and load testing.
