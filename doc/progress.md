# JanusLLM Progress Board

Last updated: 2026-05-31

## Overview

- Phase 1, API ingress: in progress, about 85%.
- Phase 2, governance: in progress, about 55%.
- Phase 3, routing: in progress, about 60%.
- Phase 4, billing and admin: in progress, about 60%.
- Phase 5, semantic cache: not started.
- Phase 6, load testing and deployment: not started.

## Phase 1: API Ingress

- [x] Gin gateway service.
- [x] Native proxy: `/v1/chat/completions`.
- [x] Native proxy: `/v1/messages`.
- [x] Native proxy: `/v1/completions`.
- [x] Native proxy: `/v1/embeddings`.
- [x] Local `/v1/models` response filtered by key permissions.
- [x] OpenAI and Anthropic adapters.
- [x] SSE streaming proxy.
- [x] Swagger UI and OpenAPI JSON.
- [ ] Full error response standardization.
- [ ] Header forwarding allowlist.

## Phase 2: Governance

- [x] API key authentication.
- [x] Effective model permissions from key/team intersection.
- [x] RPM rate limiting.
- [x] `RequestPerMinute=0` means unlimited RPM.
- [x] Key cache refresh and idle eviction.
- [x] Admin Basic Auth.
- [x] Organization, team, and key admin APIs.
- [x] Legacy auth helpers no longer call `log.Fatal`; they expose error-returning variants.
- [ ] TPM limiting.
- [ ] Daily/monthly token quotas.
- [ ] Daily/monthly spend budgets.
- [ ] Concurrency limits.
- [ ] IP allowlists.
- [ ] Full role model.

## Phase 3: Routing

- [x] Round-robin strategy.
- [x] Weighted strategy.
- [x] Latency-based strategy.
- [x] Client-sticky strategy based on stable client identity.
- [x] Extensible balancer interface with request selection context.
- [x] Basic retry/fallback within a model group.
- [x] Upstream timeout control.
- [ ] Least-inflight strategy.
- [ ] Active health checks.
- [ ] Passive health checks.
- [ ] Circuit breaker and half-open recovery.
- [ ] More detailed same-provider and cross-provider fallback policies.

## Phase 4: Billing And Admin

- [x] Token-based billing.
- [x] `janus_spend_log` persistence.
- [x] Key balance deduction and total spend update.
- [x] Database-managed create/update timestamps.
- [x] PostgreSQL schema.
- [x] Runtime PostgreSQL driver.
- [x] Backend admin API.
- [x] Startup sync from YAML model config to DB model tables.
- [x] Spend log fields: `provider`, `latency_ms`, `cache_hit`, `tenant`.
- [x] Streaming billing skips records when upstream usage is missing.
- [x] Modern React admin frontend scaffold under `web/`.
- [ ] Admin frontend connected to live APIs.
- [ ] Dashboard/reporting query APIs.

## Phase 5: Semantic Cache

- [ ] L1 exact cache.
- [ ] L2 semantic cache with embeddings/vector search.
- [ ] L3 prompt fragment cache.
- [ ] `x-cache-hit` response header from Janus cache.
- [ ] Tenant-isolated cache policy.

## Phase 6: Load Testing And Deployment

- [ ] Load test baseline: QPS, p95/p99, error rate.
- [ ] Container and deployment manifests.
- [ ] Observability with Prometheus/Grafana/OpenTelemetry.
- [ ] Production rollout and rollback process.

## Current Branch Goals

- [x] Create the work branch from latest `main`.
- [x] Align docs with current code progress.
- [x] Keep YAML as the runtime source of truth and sync model config to DB at startup.
- [x] Replace `log.Fatal` in auth helpers with graceful error handling.
- [x] Add a modern minimal admin frontend.
- [x] Extend load balancing strategies for round-robin, weighted, latency, and client-sticky routing.
- [x] Add `provider`, `latency_ms`, `cache_hit`, and `tenant` to spend logs and improve streaming billing.

## Remaining Risks

- Frontend build was not verified in this environment because `npm` is not available on PATH.
- The DB migration script is idempotent for existing Janus tables, but it still needs a real PostgreSQL smoke test before release.
- The frontend currently uses mock data; live admin API integration is the next product step.

## Suggested Next Iteration

- [ ] Connect the React admin dashboard to `/v1/admin/*` and `/v1/models`.
- [ ] Add TPM, budget, and concurrency limits.
- [ ] Add active/passive health checks and circuit breaking.
- [ ] Add observability and load-test baselines.
