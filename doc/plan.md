# 🧠 LLM 智能网关设计方案（Golang 实现）

## 1. 项目目标

构建一个对标 LiteLLM / OpenRouter 的 **LLM 智能网关**，具备以下能力：

* 统一接入多种 LLM Provider（OpenAI / Anthropic / 自建模型等）
* 提供 API Key 管理、权限控制、计费与审计能力
* 实现智能路由、负载均衡与 fallback
* 支持语义缓存，降低成本
* 支持多租户与后台管理

---

## 2. 产品定位

网关分为三层：

### 2.1 接入层

* 统一 API（兼容 OpenAI / Anthropic）
* header / param 透传
* streaming 支持（SSE）

### 2.2 治理层

* API Key 管理
* 权限控制（模型级）
* 配额与限流
* 计费与消费统计

### 2.3 智能调度层

* 健康检查
* 负载均衡
* fallback / retry
* 语义缓存
* 成本优化

---

## 3. 整体架构

```text
Client
   │
   ▼
LLM Gateway
 ├── API Adapter
 ├── Auth & Policy
 ├── Routing Engine
 ├── Optimization Layer
 ├── Metering & Audit
   │
   ▼
Providers（OpenAI / Anthropic / vLLM 等）

Data Layer:
- PostgreSQL（配置 / 计费 / 审计）
- Redis（限流 / 缓存 / 状态）
- Vector DB（语义缓存）
```

---

## 4. 技术选型

### 后端

* Golang + Gin
* net/http（自定义 transport）

### 数据层

* PostgreSQL（主数据）
* Redis（缓存 / 限流）
* pgvector（语义缓存）

### 观测

* Prometheus + Grafana
* OpenTelemetry

### 管理后台

* React / Next.js

---

## 5. 核心设计

---

## 5.1 统一请求模型（Canonical Request）

```go
type CanonicalRequest struct {
    Model       string
    Messages    []Message
    Temperature *float64
    MaxTokens   *int
    Stream      bool
    Metadata    map[string]string
}
```

作用：

* 屏蔽 provider 差异
* 统一路由、缓存、计费逻辑

---

## 5.2 API 接入层

### 支持接口

* `/v1/chat/completions`（OpenAI）
* `/v1/messages`（Anthropic）

### 功能

* header 透传（白名单）
* streaming（SSE）
* 参数透传
* 错误标准化

---

## 5.3 账号与权限系统

### 角色

* 平台管理员
* 租户管理员
* API Key

### API Key 功能

* 有效期
* 模型权限
* 限流 / 配额
* IP 白名单

---

## 5.4 配额与限流

支持：

* RPM（请求速率）
* TPM（token速率）
* 日/月 token 限额
* 日/月费用限额
* 并发限制

执行顺序：

```text
鉴权 → 模型权限 → 并发 → 限流 → 预算
```

---

## 5.5 智能路由设计

### 路由层级

1. 逻辑模型 → provider
2. provider → 实例

---

### 负载均衡策略

* Round Robin
* Weighted
* Least inflight
* Latency-based

---

### 推荐：Score 路由

```text
score =
  health +
  latency +
  load +
  cost +
  error_penalty
```

---

### fallback 机制

* 同实例 retry
* 同 provider fallback
* 跨 provider fallback

---

## 5.6 健康检查

### 被动

* 错误率
* 延迟
* 超时

### 主动

* /models
* 小请求探测

### 熔断

* error rate 超阈值 → 熔断
* 半开恢复

---

## 5.7 语义缓存

### 三层缓存

| 层级 | 类型            |
| ---- | --------------- |
| L1   | 精确缓存        |
| L2   | 语义缓存        |
| L3   | prompt 片段缓存 |

---

### 语义缓存流程

1. prompt normalization
2. embedding
3. 向量检索
4. 相似度判断
5. 命中返回

---

### 命中条件

* 相似度 > 阈值
* model 相同
* system prompt 相同
* 低温度

---

### 建议

* 默认 **租户隔离缓存**
* 返回 `x-cache-hit` header

---

## 5.8 计费系统

### usage 记录字段

* request_id
* tenant_id
* model
* provider
* tokens
* cost
* latency
* cache_hit

---

### 计费公式

```text
cost =
  input_tokens * input_price
+ output_tokens * output_price
```

---

## 6. 数据库设计（核心表）

### tenants

* 租户

### api_keys

* key
* hash
* 权限
* 配额

### providers

* provider 信息

### model_endpoints

* 模型与 provider 映射

### pricing_rules

* 价格规则

### usage_logs

* 使用记录

### quota_policies

* 限流策略

---

## 7. 配置管理设计

## 7.1 分层配置

| 类型     | 存储           |
| -------- | -------------- |
| 静态配置 | yaml / env     |
| 动态配置 | PostgreSQL     |
| 敏感信息 | Secret / Vault |

---

## 7.2 推荐方案

* yaml：仅用于启动配置
* DB：模型 / 路由 / 价格 / 权限
* Redis：运行态状态

---

## 8. 核心能力补充

### 观测

* QPS
* latency p95/p99
* error rate
* cache hit

### 安全

* 审计日志
* IP 白名单
* 脱敏

### 模型别名

* `cheap-chat`
* `fast-reasoning`

### Shadow Traffic

* A/B 测试模型

### 幂等支持

* Idempotency-Key

---

## 9. 项目结构

```text
llm-gateway/
├── cmd/
├── internal/
│   ├── adapters/
│   ├── router/
│   ├── auth/
│   ├── billing/
│   ├── cache/
│   ├── health/
│   └── telemetry/
├── migrations/
├── deployments/
└── web-admin/
```

---

## 10. 开发排期（16周）

### Phase 1（3周）

* 基础代理
* OpenAI / Anthropic 支持
* SSE

### Phase 2（3周）

* API Key
* 限流 / 配额

### Phase 3（3周）

* 路由
* fallback
* 健康检查

### Phase 4（3周）

* 计费
* 管理后台

### Phase 5（2周）

* 语义缓存

### Phase 6（1周）

* 压测
* 部署

---

## 11. MVP 边界

### 必做

* API 兼容
* Key 管理
* 限流
* usage
* fallback
* 精确缓存

### 后续

* 语义缓存
* Shadow traffic
* 安全治理

---

## 12. 风险与建议

### 风险

* 协议复杂
* 流式处理难
* 缓存误命中
* 计费不准

### 建议

* 先做 80% 功能
* 分层设计
* usage 与 billing 解耦
* 配置中心化

---

## 13. 最终总结

这个网关的核心价值在于：

> **统一入口 + 智能调度 + 成本控制 + 权限治理**

优先级建议：

1. API + Key + 计费
2. 路由与 fallback
3. 语义缓存优化

---

如果你下一步想继续推进，我可以帮你细化：

* Go 项目骨架（可直接开工）
* 数据库 SQL
* API Swagger
* Router 核心代码设计
* Kubernetes 部署方案

直接可以进入“写代码阶段”。
