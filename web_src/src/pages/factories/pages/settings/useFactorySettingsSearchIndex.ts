import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useMemo } from "react";

import type { FactorySettingsNavGroup } from "./settingsNavItems";
import { buildFactorySettingsSearchIndex, type FactorySettingsSearchResult } from "./settingsSearch";

/** Find settings index for the groups the current user can open, plus one hit per integration provider. */
export function useFactorySettingsSearchIndex(navGroups: FactorySettingsNavGroup[]): FactorySettingsSearchResult[] {
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  return useMemo(
    () => buildFactorySettingsSearchIndex({ navGroups, integrations: availableIntegrations }),
    [availableIntegrations, navGroups],
  );
}
