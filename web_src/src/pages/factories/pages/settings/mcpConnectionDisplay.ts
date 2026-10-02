import type { FactoriesFactoryAgentResource } from "@/api-client";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { catalogEntryForResource } from "./mcpCatalog";

export function mcpConnectionCatalogEntry(resource: Pick<FactoriesFactoryAgentResource, "url" | "auth" | "name">) {
  return catalogEntryForResource(resource);
}

export function mcpConnectionDisplayName(resource: FactoriesFactoryAgentResource): string {
  const fromCatalog = mcpConnectionCatalogEntry(resource)?.label;
  if (fromCatalog) {
    return fromCatalog;
  }
  const rawName = resource.name?.trim();
  if (!rawName) {
    return AGENT_RESOURCES_COPY.unnamedResource;
  }
  return rawName.charAt(0).toUpperCase() + rawName.slice(1);
}
