import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useBillingProductEnabled } from "@/hooks/useBillingProductEnabled";
import { useCodexUsage } from "@/hooks/useCodexUsage";
import { FEATURE_CODEX_SUBSCRIPTIONS, FEATURE_SUPERPLANE_MCP_SERVER } from "@/lib/experimentalFeatures";

import {
  type FactorySettingsNavGroup,
  filterFactorySettingsNavGroupsByPermission,
  omitFactorySettingsBillingItem,
} from "./settingsNavItems";
import { useFactorySettingsNavGroups } from "./useFactorySettingsNavGroups";

/** Nav item id for the SuperPlane MCP Server (inbound MCP clients). */
const WORKSPACE_SUPERPLANE_MCP_SERVER_NAV_ITEM_ID = "workspace-superplane-mcp-server";

/**
 * Drops nav items whose experimental features or connections are absent, and skips any group
 * left with no items. The source groups stay static so other consumers
 * (e.g. route lookups) keep seeing the full, approved list.
 */
function visibleFactorySettingsNavGroups(
  groups: FactorySettingsNavGroup[],
  hasExperimentalFeature: (featureId: string) => boolean,
  hasCodexConnection: boolean,
): FactorySettingsNavGroup[] {
  const hiddenNavItemIds = new Set<string>();
  if (!hasExperimentalFeature(FEATURE_CODEX_SUBSCRIPTIONS) || !hasCodexConnection) {
    hiddenNavItemIds.add("account-codex-usage");
  }
  if (!hasExperimentalFeature(FEATURE_SUPERPLANE_MCP_SERVER)) {
    hiddenNavItemIds.add(WORKSPACE_SUPERPLANE_MCP_SERVER_NAV_ITEM_ID);
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
 * removed, then items with unavailable experimental features or connections.
 */
export function useVisibleFactorySettingsNavGroups(organizationId: string): FactorySettingsNavGroup[] {
  const settingsNavGroups = useFactorySettingsNavGroups();
  const { hasPermission, isLoading: permissionsLoading } = usePermissions();
  const { has: hasExperimentalFeature, isLoading: featuresLoading } = useExperimentalFeature(organizationId);
  const codexUsage = useCodexUsage(
    organizationId,
    !featuresLoading && hasExperimentalFeature(FEATURE_CODEX_SUBSCRIPTIONS),
  );
  const billingProductEnabled = useBillingProductEnabled();
  const permitted = filterFactorySettingsNavGroupsByPermission(settingsNavGroups, hasPermission, permissionsLoading);
  const visible = visibleFactorySettingsNavGroups(permitted, hasExperimentalFeature, Boolean(codexUsage.data?.length));
  if (billingProductEnabled) {
    return visible;
  }
  return omitFactorySettingsBillingItem(visible);
}
