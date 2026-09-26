import type { FactoryAgentResourceAuth } from "@/api-client";

import { AGENT_RESOURCES_COPY, type AgentResourceInstruction } from "./agentResourceCopy";

export type MCPCatalogCategory = "code" | "issues" | "chat" | "cicd" | "observability" | "incident" | "infrastructure";

export type MCPCatalogEntry = {
  id: string;
  name: string;
  label: string;
  icon: string;
  category: MCPCatalogCategory;
  url: string;
  auth: FactoryAgentResourceAuth;
  headerName?: string;
  instruction: AgentResourceInstruction;
};

export type MCPCatalogGroup = {
  id: MCPCatalogCategory;
  label: string;
  entries: MCPCatalogEntry[];
};

export type MCPCatalogConnectionDefaults = {
  name: string;
  url: string;
  auth: FactoryAgentResourceAuth;
  headers?: { name: string }[];
  instruction: AgentResourceInstruction;
};

export function catalogConnectionDefaults(entry?: MCPCatalogEntry): MCPCatalogConnectionDefaults | undefined {
  if (!entry) {
    return undefined;
  }
  return {
    name: entry.name,
    url: entry.url,
    auth: entry.auth,
    instruction: entry.instruction,
    ...(entry.headerName ? { headers: [{ name: entry.headerName }] } : {}),
  };
}

export function catalogEntryForResource(resource?: { url?: string }): MCPCatalogEntry | undefined {
  const url = resource?.url?.trim();
  if (!url) {
    return undefined;
  }
  return MCP_CATALOG.find((entry) => entry.url === url);
}

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

/** Remote MCP servers that SuperPlane can open with a documented HTTPS URL and auth method. */
export const MCP_CATALOG: MCPCatalogEntry[] = [
  {
    id: "github",
    name: "github",
    label: "GitHub",
    icon: "github",
    category: "code",
    url: "https://api.githubcopilot.com/mcp/",
    auth: "AUTH_HEADERS",
    headerName: "Authorization",
    instruction: AGENT_RESOURCES_COPY.githubInstruction,
  },
  {
    id: "jira",
    name: "jira",
    label: "Jira",
    icon: "jira",
    category: "issues",
    url: "https://mcp.atlassian.com/v2/mcp",
    auth: "AUTH_OAUTH",
    instruction: AGENT_RESOURCES_COPY.jiraInstruction,
  },
  {
    id: "linear",
    name: "linear",
    label: "Linear",
    icon: "linear",
    category: "issues",
    url: "https://mcp.linear.app/mcp",
    auth: "AUTH_OAUTH",
    instruction: AGENT_RESOURCES_COPY.linearInstruction,
  },
  {
    id: "circleci",
    name: "circleci",
    label: "CircleCI",
    icon: "circleci",
    category: "cicd",
    url: "https://mcp.circleci.com/v1/mcp",
    auth: "AUTH_OAUTH",
    instruction: AGENT_RESOURCES_COPY.circleciInstruction,
  },
  {
    id: "semaphore",
    name: "semaphore",
    label: "Semaphore",
    icon: "semaphore",
    category: "cicd",
    url: "https://mcp.semaphoreci.com/mcp",
    auth: "AUTH_OAUTH",
    instruction: AGENT_RESOURCES_COPY.semaphoreInstruction,
  },
  {
    id: "sentry",
    name: "sentry",
    label: "Sentry",
    icon: "sentry",
    category: "observability",
    url: "https://mcp.sentry.dev/mcp",
    auth: "AUTH_OAUTH",
    instruction: AGENT_RESOURCES_COPY.sentryInstruction,
  },
];
