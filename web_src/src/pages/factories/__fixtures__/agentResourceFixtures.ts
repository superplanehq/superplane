import type { FactoriesFactoryAgentResource, FactoryAgentResourceAuth } from "@/api-client";

import { MCP_CATALOG } from "../pages/settings/mcpCatalog";
import { PRIMARY_FACTORY_ID, YESTERDAY } from "./factoryPageIds";

const NOW = YESTERDAY;

export const HEADER_MCP_RESOURCE: FactoriesFactoryAgentResource = {
  id: "resource-docs",
  factoryId: PRIMARY_FACTORY_ID,
  kind: "KIND_MCP_SERVER",
  name: "docs",
  enabled: true,
  url: "https://mcp.example.com/mcp",
  auth: "AUTH_HEADERS",
  headers: [{ name: "Authorization", secretName: "vendor-mcp", secretKey: "token" }],
  createdAt: NOW,
  updatedAt: NOW,
};

export const OAUTH_NOT_CONNECTED_RESOURCE: FactoriesFactoryAgentResource = {
  id: "resource-mobbin",
  factoryId: PRIMARY_FACTORY_ID,
  kind: "KIND_MCP_SERVER",
  name: "mobbin",
  enabled: true,
  url: "https://api.mobbin.com/mcp",
  auth: "AUTH_OAUTH",
  oauthStatus: "OAUTH_STATUS_NOT_CONNECTED",
  createdAt: NOW,
  updatedAt: NOW,
};

export const OAUTH_CONNECTED_RESOURCE: FactoriesFactoryAgentResource = {
  ...OAUTH_NOT_CONNECTED_RESOURCE,
  id: "resource-mobbin-connected",
  oauthStatus: "OAUTH_STATUS_CONNECTED",
  oauthConnectedByUserId: "storybook-user",
  oauthConnectedAt: NOW,
};

export const OAUTH_NEEDS_RECONNECT_RESOURCE: FactoriesFactoryAgentResource = {
  ...OAUTH_NOT_CONNECTED_RESOURCE,
  id: "resource-mobbin-reconnect",
  oauthStatus: "OAUTH_STATUS_NEEDS_RECONNECT",
  oauthError: "SuperPlane could not refresh the access token. Connect again.",
};

export const OAUTH_VENDOR_REJECTED_RESOURCE: FactoriesFactoryAgentResource = {
  ...OAUTH_NOT_CONNECTED_RESOURCE,
  id: "resource-mobbin-rejected",
  oauthStatus: "OAUTH_STATUS_VENDOR_REJECTED",
  oauthError:
    "This vendor does not allow SuperPlane as an OAuth client. Use header authentication if the vendor ships an API key.",
};

export const INLINE_SKILL: FactoriesFactoryAgentResource = {
  id: "resource-review-copy",
  factoryId: PRIMARY_FACTORY_ID,
  kind: "KIND_SKILL",
  name: "review-copy",
  enabled: true,
  markdown: "---\nname: review-copy\ntitle: Review copy\ndescription: Review UI copy.\n---\n\nWrite STE copy.",
  createdAt: NOW,
  updatedAt: NOW,
};

export const UI_UX_PRO_MAX_SKILL: FactoriesFactoryAgentResource = {
  id: "resource-ui-ux-pro-max",
  factoryId: PRIMARY_FACTORY_ID,
  kind: "KIND_SKILL",
  name: "ui-ux-pro-max",
  enabled: true,
  repository: "nextlevelbuilder/ui-ux-pro-max-skill",
  ref: "v1.2.0",
  path: ".",
  createdAt: NOW,
  updatedAt: NOW,
};

function storybookInlineSkill({
  id,
  name,
  title,
  description,
  enabled = true,
}: {
  id: string;
  name: string;
  title: string;
  description: string;
  enabled?: boolean;
}): FactoriesFactoryAgentResource {
  return {
    id,
    factoryId: PRIMARY_FACTORY_ID,
    kind: "KIND_SKILL",
    name,
    enabled,
    markdown: `---\nname: ${name}\ntitle: ${title}\ndescription: ${description}\n---\n\nSkill body.`,
    createdAt: NOW,
    updatedAt: NOW,
  };
}

/** Storybook Agent Configured state: five workspace skills. */
export const CONFIGURED_SKILLS: FactoriesFactoryAgentResource[] = [
  INLINE_SKILL,
  UI_UX_PRO_MAX_SKILL,
  storybookInlineSkill({
    id: "resource-commit-messages",
    name: "commit-messages",
    title: "Commit messages",
    description: "Write Conventional Commits subjects and STE bodies.",
  }),
  storybookInlineSkill({
    id: "resource-pr-description",
    name: "pr-description",
    title: "Pull request description",
    description: "Draft pull request summaries and test plans.",
  }),
  {
    id: "resource-superplane-changelog",
    factoryId: PRIMARY_FACTORY_ID,
    kind: "KIND_SKILL",
    name: "superplane-changelog",
    enabled: false,
    repository: "superplane/superplane",
    ref: "main",
    path: ".agents/skills/superplane-changelog",
    createdAt: NOW,
    updatedAt: NOW,
  },
];

export const MIXED_AGENT_RESOURCES: FactoriesFactoryAgentResource[] = [
  HEADER_MCP_RESOURCE,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
];

function catalogEntry(catalogId: string) {
  const entry = MCP_CATALOG.find((item) => item.id === catalogId);
  if (!entry) {
    throw new Error(`Missing MCP catalog entry: ${catalogId}`);
  }
  return entry;
}

function storybookMcpResource({
  id,
  catalogId,
  oauthStatus,
  oauthError,
  enabled = true,
}: {
  id: string;
  catalogId: string;
  oauthStatus?: FactoriesFactoryAgentResource["oauthStatus"];
  oauthError?: string;
  enabled?: boolean;
}): FactoriesFactoryAgentResource {
  const entry = catalogEntry(catalogId);
  const auth = entry.auth as FactoryAgentResourceAuth;
  return {
    id,
    factoryId: PRIMARY_FACTORY_ID,
    kind: "KIND_MCP_SERVER",
    name: entry.name,
    enabled,
    url: entry.url,
    auth,
    ...(auth === "AUTH_HEADERS" && entry.headerName
      ? { headers: [{ name: entry.headerName, secretName: "vendor-mcp", secretKey: "token" }] }
      : {}),
    ...(oauthStatus ? { oauthStatus, oauthConnectedAt: oauthStatus === "OAUTH_STATUS_CONNECTED" ? NOW : undefined } : {}),
    ...(oauthError ? { oauthError } : {}),
    createdAt: NOW,
    updatedAt: NOW,
  };
}

/** Storybook Configured state: catalog labels, auth methods, and connection statuses. */
export const CONFIGURED_MCP_RESOURCES: FactoriesFactoryAgentResource[] = [
  {
    ...storybookMcpResource({ id: "resource-github", catalogId: "github" }),
    disabledTools: Array.from({ length: 11 }, (_, index) => `tool_${index + 1}`),
  },
  storybookMcpResource({ id: "resource-gitlab", catalogId: "gitlab", oauthStatus: "OAUTH_STATUS_CONNECTED" }),
  storybookMcpResource({ id: "resource-linear", catalogId: "linear", oauthStatus: "OAUTH_STATUS_CONNECTED" }),
  storybookMcpResource({ id: "resource-sentry", catalogId: "sentry", oauthStatus: "OAUTH_STATUS_CONNECTED" }),
  storybookMcpResource({ id: "resource-circleci", catalogId: "circleci" }),
  storybookMcpResource({ id: "resource-semaphore", catalogId: "semaphore" }),
  storybookMcpResource({ id: "resource-jira", catalogId: "jira", oauthStatus: "OAUTH_STATUS_NOT_CONNECTED" }),
  storybookMcpResource({
    id: "resource-notion",
    catalogId: "notion",
    oauthStatus: "OAUTH_STATUS_NEEDS_RECONNECT",
    oauthError: "SuperPlane could not refresh the access token. Connect again.",
  }),
  storybookMcpResource({
    id: "resource-figma",
    catalogId: "figma",
    oauthStatus: "OAUTH_STATUS_VENDOR_REJECTED",
    oauthError:
      "This vendor does not allow SuperPlane as an OAuth client. Use header authentication if the vendor ships an API key.",
  }),
  storybookMcpResource({ id: "resource-slack", catalogId: "slack", oauthStatus: "OAUTH_STATUS_NOT_CONNECTED" }),
  storybookMcpResource({ id: "resource-vercel", catalogId: "vercel", oauthStatus: "OAUTH_STATUS_CONNECTED" }),
  storybookMcpResource({ id: "resource-datadog", catalogId: "datadog", oauthStatus: "OAUTH_STATUS_NOT_CONNECTED" }),
];

function toolList(count: number) {
  return Array.from({ length: count }, (_, index) => ({
    name: `tool_${index + 1}`,
    description: `Tool ${index + 1}.`,
    readOnly: index % 2 === 0,
  }));
}

/** Tool counts for connected rows in the Configured story. */
export const CONFIGURED_MCP_RESOURCE_TOOLS_BY_ID: Record<
  string,
  Array<{ name: string; description?: string; readOnly?: boolean }>
> = {
  "resource-github": toolList(45),
  "resource-linear": toolList(12),
  "resource-sentry": toolList(8),
  "resource-semaphore": toolList(3),
  "resource-vercel": toolList(21),
};
