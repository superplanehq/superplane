import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import {
  FEATURE_ORGANIZATION_BYOK,
  FEATURE_SUPERPLANE_MCP_SERVER,
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";

import { type FactorySettingsNavGroup, filterFactorySettingsNavGroupsByPermission } from "./settingsNavItems";
import { useFactorySettingsNavGroups } from "./useFactorySettingsNavGroups";

/** Nav item id for Agent settings (MCP servers and skills). */
const WORKSPACE_AGENT_NAV_ITEM_ID = "workspace-agent";

/** Nav item id for the SuperPlane MCP Server (inbound MCP clients). */
const WORKSPACE_SUPERPLANE_MCP_SERVER_NAV_ITEM_ID = "workspace-superplane-mcp-server";

/** Nav item id for the organization LLM Models settings page, gated behind `FEATURE_ORGANIZATION_BYOK`. */
const ORGANIZATION_MODELS_NAV_ITEM_ID = "organization-models";

/**
 * Drops nav items whose experimental features are off, and skips any group
 * left with no items. The source groups stay static so other consumers
 * (e.g. route lookups) keep seeing the full, approved list.
 */
function visibleFactorySettingsNavGroups(
  groups: FactorySettingsNavGroup[],
  hasExperimentalFeature: (featureId: string) => boolean,
): FactorySettingsNavGroup[] {
  const hiddenNavItemIds = new Set<string>();
  if (!hasExperimentalFeature(FEATURE_WORKSPACE_MCP) && !hasExperimentalFeature(FEATURE_WORKSPACE_SKILLS)) {
    hiddenNavItemIds.add(WORKSPACE_AGENT_NAV_ITEM_ID);
  }
  if (!hasExperimentalFeature(FEATURE_SUPERPLANE_MCP_SERVER)) {
    hiddenNavItemIds.add(WORKSPACE_SUPERPLANE_MCP_SERVER_NAV_ITEM_ID);
  }
  if (!hasExperimentalFeature(FEATURE_ORGANIZATION_BYOK)) {
    hiddenNavItemIds.add(ORGANIZATION_MODELS_NAV_ITEM_ID);
  }
  if (hiddenNavItemIds.size === 0) {
    return groups;
  }

  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => !hiddenNavItemIds.has(item.id)),
    }))
    .filter((group) => group.items.length > 0);
}

/**
 * Settings nav groups the current user can open: permission-gated items are
 * removed, then items behind experimental features that are off.
 */
export function useVisibleFactorySettingsNavGroups(organizationId: string): FactorySettingsNavGroup[] {
  const settingsNavGroups = useFactorySettingsNavGroups();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { has: hasExperimentalFeature } = useExperimentalFeature(organizationId);
  return visibleFactorySettingsNavGroups(
    filterFactorySettingsNavGroupsByPermission(settingsNavGroups, canAct, permissionsLoading),
    hasExperimentalFeature,
  );
}
