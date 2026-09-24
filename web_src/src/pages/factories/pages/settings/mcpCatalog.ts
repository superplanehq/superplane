import type { FactoryAgentResourceAuth } from "@/api-client";

export type MCPCatalogCategory = "code" | "issues" | "chat" | "cicd" | "observability" | "incident" | "infrastructure";

export type MCPCatalogEntry = {
  id: string;
  name: string;
  label: string;
  icon: string;
  category: MCPCatalogCategory;
  url?: string;
  auth?: FactoryAgentResourceAuth;
};

export type MCPCatalogGroup = {
  id: MCPCatalogCategory;
  label: string;
  entries: MCPCatalogEntry[];
};

export const CUSTOM_MCP_CATALOG_ID = "custom";

export const MCP_CATALOG_CATEGORY_ORDER: MCPCatalogCategory[] = [
  "code",
  "issues",
  "chat",
  "cicd",
  "observability",
  "incident",
  "infrastructure",
];

export const MCP_CATALOG_CATEGORY_LABELS: Record<MCPCatalogCategory, string> = {
  code: "Code",
  issues: "Issues",
  chat: "Chat",
  cicd: "CI/CD",
  observability: "Observability",
  incident: "Incident",
  infrastructure: "Infrastructure",
};

function catalogSearchText(entry: MCPCatalogEntry): string {
  return `${entry.label} ${entry.name} ${entry.id} ${entry.category} ${MCP_CATALOG_CATEGORY_LABELS[entry.category]}`;
}

export function filterMCPCatalog(entries: MCPCatalogEntry[], query: string): MCPCatalogEntry[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return entries;
  }
  return entries.filter((entry) => catalogSearchText(entry).toLowerCase().includes(needle));
}

export function groupMCPCatalog(entries: MCPCatalogEntry[]): MCPCatalogGroup[] {
  const sorted = [...entries].sort((left, right) =>
    left.label.localeCompare(right.label, undefined, { sensitivity: "base" }),
  );
  return MCP_CATALOG_CATEGORY_ORDER.flatMap((id) => {
    const groupEntries = sorted.filter((entry) => entry.category === id);
    if (groupEntries.length === 0) {
      return [];
    }
    return [{ id, label: MCP_CATALOG_CATEGORY_LABELS[id], entries: groupEntries }];
  });
}

/** Common remote MCP servers that match SuperPlane integrations. URLs are set only when the vendor documents an https MCP endpoint. */
export const MCP_CATALOG: MCPCatalogEntry[] = [
  {
    id: "github",
    name: "github",
    label: "GitHub",
    icon: "github",
    category: "code",
    url: "https://api.githubcopilot.com/mcp/",
    auth: "AUTH_OAUTH",
  },
  { id: "gitlab", name: "gitlab", label: "GitLab", icon: "gitlab", category: "code" },
  { id: "bitbucket", name: "bitbucket", label: "Bitbucket", icon: "bitbucket", category: "code" },
  {
    id: "linear",
    name: "linear",
    label: "Linear",
    icon: "linear",
    category: "issues",
    url: "https://mcp.linear.app/mcp",
    auth: "AUTH_OAUTH",
  },
  { id: "jira", name: "jira", label: "Jira", icon: "jira", category: "issues" },
  { id: "slack", name: "slack", label: "Slack", icon: "slack", category: "chat" },
  { id: "discord", name: "discord", label: "Discord", icon: "discord", category: "chat" },
  { id: "teams", name: "teams", label: "Microsoft Teams", icon: "teams", category: "chat" },
  { id: "circleci", name: "circleci", label: "CircleCI", icon: "circleci", category: "cicd" },
  {
    id: "sentry",
    name: "sentry",
    label: "Sentry",
    icon: "sentry",
    category: "observability",
    url: "https://mcp.sentry.dev/mcp",
    auth: "AUTH_OAUTH",
  },
  { id: "datadog", name: "datadog", label: "Datadog", icon: "datadog", category: "observability" },
  { id: "grafana", name: "grafana", label: "Grafana", icon: "grafana", category: "observability" },
  { id: "logfire", name: "logfire", label: "Logfire", icon: "logfire", category: "observability" },
  { id: "honeycomb", name: "honeycomb", label: "Honeycomb", icon: "honeycomb", category: "observability" },
  { id: "newrelic", name: "newrelic", label: "New Relic", icon: "newrelic", category: "observability" },
  { id: "pagerduty", name: "pagerduty", label: "PagerDuty", icon: "pagerduty", category: "incident" },
  { id: "incident", name: "incident", label: "incident.io", icon: "incident", category: "incident" },
  { id: "rootly", name: "rootly", label: "Rootly", icon: "rootly", category: "incident" },
  { id: "firehydrant", name: "firehydrant", label: "FireHydrant", icon: "firehydrant", category: "incident" },
  { id: "cloudflare", name: "cloudflare", label: "Cloudflare", icon: "cloudflare", category: "infrastructure" },
  { id: "launchdarkly", name: "launchdarkly", label: "LaunchDarkly", icon: "launchdarkly", category: "infrastructure" },
];
