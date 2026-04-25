# JanusLLM 开发看板（Phase 版）

最后更新：2026-04-04

## 总览

- Phase 1（接入层）：进行中（约 75%）
- Phase 2（治理层）：进行中（约 40%）
- Phase 3（调度层）：进行中（约 35%）
- Phase 4（计费与后台）：进行中（约 30%）
- Phase 5（语义缓存）：未开始（0%）
- Phase 6（压测与部署）：未开始（0%）

## Phase 1：接入层（基础代理 + OpenAI/Anthropic + SSE）

- [x] Gin 网关基础服务
- [x] 原生接口透传：`/v1/chat/completions`
- [x] 原生接口透传：`/v1/messages`
- [x] 原生接口透传：`/v1/completions`
- [x] 原生接口透传：`/v1/embeddings`
- [x] `/v1/models` 网关本地返回可访问模型列表
- [x] OpenAI/Anthropic 基础 adapter
- [x] SSE 流式透传
- [ ] 错误标准化（统一错误码/错误结构）
- [ ] header 透传白名单（当前是透传为主）

## Phase 2：治理层（API Key + 限流/配额）

- [x] API Key 鉴权
- [x] 模型权限控制（`model_list`）
- [x] RPM 限流
- [x] `RequestPerMinute=0` 不限流语义修复
- [x] 并发安全（请求环与消费队列加锁）
- [ ] TPM 限流
- [ ] 日/月 token 限额
- [ ] 日/月费用预算限额
- [ ] 并发限制
- [ ] IP 白名单
- [ ] 角色体系（平台管理员/租户管理员）

## Phase 3：调度层（路由 + fallback + 健康检查）

- [x] Round Robin
- [x] Weighted
- [x] 基础 retry/fallback（同模型组内实例）
- [x] 上游超时控制
- [ ] Least inflight
- [ ] Latency-based
- [ ] Score 路由
- [ ] 主动健康检查
- [ ] 被动健康检查
- [ ] 熔断与半开恢复
- [ ] 同 provider / 跨 provider fallback 策略细化

## Phase 4：计费 + 管理后台

- [x] 按 token 计费
- [x] `janus_spend_log` 消费落库
- [x] key 余额扣减
- [x] 时间字段由数据库维护（create/update）
- [ ] usage 字段补齐（provider、latency、cache_hit、tenant）
- [ ] 流式请求计费落库
- [ ] 管理后台（key、配额、报表）

## Phase 5：语义缓存

- [ ] L1 精确缓存
- [ ] L2 语义缓存（embedding + 向量检索）
- [ ] L3 prompt 片段缓存
- [ ] `x-cache-hit` 响应头
- [ ] 租户隔离缓存策略

## Phase 6：压测与部署

- [ ] 压测基线（QPS、p95/p99、错误率）
- [ ] 部署清单（容器化、回滚）
- [ ] 观测体系（Prometheus/Grafana/OTel）
- [ ] 生产发布流程（灰度/回滚）

## 当前阻塞与技术债

- [ ] `internal/auth/organization.go` 与 `internal/auth/user.go` 仍有 `log.Fatal`
- [ ] 运行时 DB 驱动仍是 MySQL，目标是 PostgreSQL
- [x] PostgreSQL 建表脚本已落地：`scripts/db/create_core_tables.sql`
- [ ] 动态配置尚未从数据库加载（模型配置仍以启动配置为主）

## 下一迭代建议（优先级）

- [ ] P0：Phase 2 配额体系（TPM + 预算 + 并发）
- [ ] P0：Phase 3 健康检查 + 熔断
- [ ] P1：计费字段补齐 + 流式计费
- [ ] P1：切 PostgreSQL 运行时驱动并接入现有 DDL
