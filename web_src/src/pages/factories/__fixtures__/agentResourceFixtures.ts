import type { FactoriesFactoryAgentResource } from "@/api-client";

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
  markdown: "# Review copy\n\nWrite STE UI copy.",
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

export const MIXED_AGENT_RESOURCES: FactoriesFactoryAgentResource[] = [
  HEADER_MCP_RESOURCE,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
];
