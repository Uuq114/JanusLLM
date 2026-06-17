import type { ApiKey, ConfigStatus, ModelGroup, OverviewMetric, UsagePoint } from "../types";

export const overviewMetrics: OverviewMetric[] = [
  {
    label: "Requests today",
    value: "128.4k",
    detail: "Across 4 active model groups",
    trend: "+12.8%",
    direction: "up"
  },
  {
    label: "Spend today",
    value: "$284.18",
    detail: "$1.9k remaining weekly budget",
    trend: "-3.1%",
    direction: "down"
  },
  {
    label: "Success rate",
    value: "99.42%",
    detail: "Rolling 24h gateway success",
    trend: "+0.4%",
    direction: "up"
  },
  {
    label: "P95 latency",
    value: "842 ms",
    detail: "Weighted by request volume",
    trend: "flat",
    direction: "flat"
  }
];

export const modelGroups: ModelGroup[] = [
  {
    name: "deepseek-v3",
    strategy: "round-robin",
    providers: 3,
    rpm: 1840,
    successRate: 99.8,
    avgLatencyMs: 710,
    costPer1kInput: 0.001,
    costPer1kOutput: 0.001,
    status: "healthy"
  },
  {
    name: "claude-3-sonnet",
    strategy: "weighted",
    providers: 2,
    rpm: 624,
    successRate: 98.9,
    avgLatencyMs: 1180,
    costPer1kInput: 0.003,
    costPer1kOutput: 0.015,
    status: "healthy"
  },
  {
    name: "embedding-small",
    strategy: "latency",
    providers: 2,
    rpm: 312,
    successRate: 97.6,
    avgLatencyMs: 330,
    costPer1kInput: 0.0001,
    costPer1kOutput: 0,
    status: "degraded"
  },
  {
    name: "gpt-4o-mini",
    strategy: "client-sticky",
    providers: 1,
    rpm: 92,
    successRate: 100,
    avgLatencyMs: 905,
    costPer1kInput: 0.00015,
    costPer1kOutput: 0.0006,
    status: "paused"
  }
];

export const apiKeys: ApiKey[] = [
  {
    id: 1024,
    name: "production-router",
    owner: "core-platform",
    models: ["*"],
    rpmLimit: 3600,
    balance: 1240,
    spend: 706.82,
    expiresAt: "Never",
    status: "active"
  },
  {
    id: 1031,
    name: "agent-lab",
    owner: "research",
    models: ["deepseek-v3", "claude-3-sonnet"],
    rpmLimit: 900,
    balance: 180,
    spend: 94.22,
    expiresAt: "2026-07-01",
    status: "active"
  },
  {
    id: 1038,
    name: "batch-evals",
    owner: "evaluation",
    models: ["deepseek-v3"],
    rpmLimit: 120,
    balance: 24,
    spend: 311.09,
    expiresAt: "2026-06-15",
    status: "low-balance"
  },
  {
    id: 1042,
    name: "legacy-chatbot",
    owner: "support",
    models: ["gpt-4o-mini"],
    rpmLimit: 60,
    balance: 0,
    spend: 72.8,
    expiresAt: "2026-05-01",
    status: "expired"
  }
];

export const usageSeries: UsagePoint[] = [
  { label: "Mon", requests: 82000, spend: 212 },
  { label: "Tue", requests: 94500, spend: 248 },
  { label: "Wed", requests: 88200, spend: 235 },
  { label: "Thu", requests: 118600, spend: 302 },
  { label: "Fri", requests: 128400, spend: 284 },
  { label: "Sat", requests: 76400, spend: 191 },
  { label: "Sun", requests: 62300, spend: 156 }
];

export const configStatuses: ConfigStatus[] = [
  {
    name: "Admin auth",
    description: "Basic auth backed by janus_admin_user",
    state: "ready",
    value: "master user synced"
  },
  {
    name: "Database",
    description: "JANUS_DATABASE_URL or secrets.database_url",
    state: "ready",
    value: "PostgreSQL connected"
  },
  {
    name: "Model routing",
    description: "Configured model_groups loaded at startup",
    state: "warning",
    value: "1 group degraded"
  },
  {
    name: "Spend flushing",
    description: "Buffered usage records flushed every minute",
    state: "ready",
    value: "queue normal"
  }
];
