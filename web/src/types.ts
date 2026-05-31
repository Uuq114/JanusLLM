export type TrendDirection = "up" | "down" | "flat";

export type OverviewMetric = {
  label: string;
  value: string;
  detail: string;
  trend: string;
  direction: TrendDirection;
};

export type ModelGroup = {
  name: string;
  strategy: "round-robin" | "weighted" | "latency" | "client-sticky";
  providers: number;
  rpm: number;
  successRate: number;
  avgLatencyMs: number;
  costPer1kInput: number;
  costPer1kOutput: number;
  status: "healthy" | "degraded" | "paused";
};

export type ApiKey = {
  id: number;
  name: string;
  owner: string;
  models: string[];
  rpmLimit: number;
  balance: number;
  spend: number;
  expiresAt: string;
  status: "active" | "low-balance" | "expired";
};

export type UsagePoint = {
  label: string;
  requests: number;
  spend: number;
};

export type ConfigStatus = {
  name: string;
  description: string;
  state: "ready" | "warning" | "missing";
  value: string;
};
