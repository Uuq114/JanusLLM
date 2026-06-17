import {
  Activity,
  AlertTriangle,
  BarChart3,
  CheckCircle2,
  ChevronDown,
  CircleDollarSign,
  Copy,
  Database,
  Gauge,
  KeyRound,
  LayoutDashboard,
  ListFilter,
  LockKeyhole,
  MoreHorizontal,
  Network,
  RefreshCcw,
  Search,
  ServerCog,
  Settings2,
  ShieldCheck,
  Sparkles
} from "lucide-react";
import type { ReactNode } from "react";
import { apiKeys, configStatuses, modelGroups, overviewMetrics, usageSeries } from "../data/mock";
import type { ApiKey, ConfigStatus, ModelGroup, OverviewMetric, UsagePoint } from "../types";

const navItems = [
  { label: "Overview", icon: LayoutDashboard, active: true },
  { label: "Models", icon: Network },
  { label: "API Keys", icon: KeyRound },
  { label: "Usage", icon: BarChart3 },
  { label: "Config", icon: ServerCog }
];

const maxRequests = Math.max(...usageSeries.map((point) => point.requests));
const maxSpend = Math.max(...usageSeries.map((point) => point.spend));

export function Dashboard() {
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">J</div>
          <div>
            <strong>JanusLLM</strong>
            <span>Admin Console</span>
          </div>
        </div>

        <nav className="nav-list" aria-label="Admin navigation">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <button className={`nav-item ${item.active ? "active" : ""}`} key={item.label} type="button">
                <Icon size={18} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <div className="sidebar-footer">
          <div className="environment-pill">
            <span className="pulse-dot" />
            Local dev
          </div>
          <button className="icon-button" type="button" aria-label="Settings">
            <Settings2 size={18} />
          </button>
        </div>
      </aside>

      <main className="main">
        <header className="topbar">
          <div>
            <p className="eyebrow">Gateway control plane</p>
            <h1>Admin dashboard</h1>
          </div>
          <div className="topbar-actions">
            <label className="search-field">
              <Search size={17} />
              <input type="search" placeholder="Search keys, teams, models" />
            </label>
            <button className="ghost-button" type="button">
              <RefreshCcw size={16} />
              Sync
            </button>
            <button className="primary-button" type="button">
              <KeyRound size={16} />
              New key
            </button>
          </div>
        </header>

        <section className="metric-grid" aria-label="Overview metrics">
          {overviewMetrics.map((metric) => (
            <MetricCard metric={metric} key={metric.label} />
          ))}
        </section>

        <section className="content-grid">
          <div className="panel model-panel">
            <PanelHeader
              eyebrow="Routing"
              title="Model groups"
              action={
                <button className="subtle-button" type="button">
                  <ListFilter size={15} />
                  Filter
                </button>
              }
            />
            <div className="model-list">
              {modelGroups.map((group) => (
                <ModelGroupRow group={group} key={group.name} />
              ))}
            </div>
          </div>

          <div className="panel spend-panel">
            <PanelHeader eyebrow="Usage" title="Requests and spend" />
            <UsageBars points={usageSeries} />
            <div className="spend-summary">
              <div>
                <span>Total requests</span>
                <strong>650.4k</strong>
              </div>
              <div>
                <span>Weekly spend</span>
                <strong>$1,628</strong>
              </div>
              <div>
                <span>Avg cost / 1k</span>
                <strong>$2.50</strong>
              </div>
            </div>
          </div>
        </section>

        <section className="content-grid lower-grid">
          <div className="panel keys-panel">
            <PanelHeader
              eyebrow="Access"
              title="API keys"
              action={
                <button className="subtle-button" type="button">
                  <ChevronDown size={15} />
                  Active
                </button>
              }
            />
            <ApiKeysTable keys={apiKeys} />
          </div>

          <div className="panel config-panel">
            <PanelHeader eyebrow="Runtime" title="Configuration status" />
            <div className="status-list">
              {configStatuses.map((item) => (
                <ConfigStatusRow item={item} key={item.name} />
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}

function PanelHeader({
  eyebrow,
  title,
  action
}: {
  eyebrow: string;
  title: string;
  action?: ReactNode;
}) {
  return (
    <div className="panel-header">
      <div>
        <span>{eyebrow}</span>
        <h2>{title}</h2>
      </div>
      {action}
    </div>
  );
}

function MetricCard({ metric }: { metric: OverviewMetric }) {
  const iconMap = {
    "Requests today": Activity,
    "Spend today": CircleDollarSign,
    "Success rate": ShieldCheck,
    "P95 latency": Gauge
  };
  const Icon = iconMap[metric.label as keyof typeof iconMap] ?? Sparkles;

  return (
    <article className="metric-card">
      <div className="metric-icon">
        <Icon size={20} />
      </div>
      <div>
        <span>{metric.label}</span>
        <strong>{metric.value}</strong>
        <p>{metric.detail}</p>
      </div>
      <em className={`trend ${metric.direction}`}>{metric.trend}</em>
    </article>
  );
}

function ModelGroupRow({ group }: { group: ModelGroup }) {
  return (
    <article className="model-row">
      <div className="model-main">
        <span className={`status-dot ${group.status}`} />
        <div>
          <strong>{group.name}</strong>
          <span>
            {group.strategy} - {group.providers} upstreams
          </span>
        </div>
      </div>
      <div className="model-stats">
        <Stat label="RPM" value={group.rpm.toLocaleString()} />
        <Stat label="Success" value={`${group.successRate}%`} />
        <Stat label="Latency" value={`${group.avgLatencyMs} ms`} />
        <Stat label="Input/1k" value={`$${group.costPer1kInput}`} />
      </div>
      <button className="icon-button" type="button" aria-label={`More actions for ${group.name}`}>
        <MoreHorizontal size={18} />
      </button>
    </article>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function UsageBars({ points }: { points: UsagePoint[] }) {
  return (
    <div className="usage-chart" aria-label="Weekly usage chart">
      {points.map((point) => (
        <div className="usage-column" key={point.label}>
          <div className="bar-stack">
            <span
              className="bar requests"
              style={{ height: `${Math.max((point.requests / maxRequests) * 100, 8)}%` }}
            />
            <span className="bar spend" style={{ height: `${Math.max((point.spend / maxSpend) * 100, 8)}%` }} />
          </div>
          <span>{point.label}</span>
        </div>
      ))}
    </div>
  );
}

function ApiKeysTable({ keys }: { keys: ApiKey[] }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Key</th>
            <th>Owner</th>
            <th>Models</th>
            <th>RPM</th>
            <th>Balance</th>
            <th>Spend</th>
            <th>Status</th>
            <th aria-label="Actions" />
          </tr>
        </thead>
        <tbody>
          {keys.map((key) => (
            <tr key={key.id}>
              <td>
                <div className="key-name">
                  <LockKeyhole size={16} />
                  <div>
                    <strong>{key.name}</strong>
                    <span>key-{key.id}</span>
                  </div>
                </div>
              </td>
              <td>{key.owner}</td>
              <td>
                <div className="tag-list">
                  {key.models.slice(0, 2).map((model) => (
                    <span className="tag" key={model}>
                      {model}
                    </span>
                  ))}
                  {key.models.length > 2 && <span className="tag muted">+{key.models.length - 2}</span>}
                </div>
              </td>
              <td>{key.rpmLimit.toLocaleString()}</td>
              <td>${key.balance.toLocaleString()}</td>
              <td>${key.spend.toLocaleString()}</td>
              <td>
                <span className={`table-status ${key.status}`}>{key.status.replace("-", " ")}</span>
              </td>
              <td>
                <button className="icon-button" type="button" aria-label={`Copy ${key.name}`}>
                  <Copy size={16} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ConfigStatusRow({ item }: { item: ConfigStatus }) {
  const icon = {
    ready: CheckCircle2,
    warning: AlertTriangle,
    missing: Database
  };
  const Icon = icon[item.state];

  return (
    <article className={`config-row ${item.state}`}>
      <div className="config-icon">
        <Icon size={18} />
      </div>
      <div>
        <strong>{item.name}</strong>
        <span>{item.description}</span>
      </div>
      <em>{item.value}</em>
    </article>
  );
}
