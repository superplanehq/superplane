import { usePermissions } from "@/contexts/usePermissions";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";

import { BrokenIntegrationsBanner } from "../BrokenIntegrationsBanner";
import { factorySettingsSectionPath } from "./factoryPagePaths";
import { findBrokenIntegrations } from "./brokenIntegrations";

/**
 * Surfaces organization integrations that broke after setup (an error
 * state) or were left half-configured (stuck pending). Runs that depend on
 * these integrations fail silently otherwise, so this gives operators a
 * workspace-level signal and a link to the repair step.
 */
export function useBrokenIntegrationsBanner(organizationId: string, factoryKey: string) {
  const { canAct } = usePermissions();
  const canReadIntegrations = canAct("integrations", "read");
  const { data: integrations = [] } = useConnectedIntegrations(organizationId, { enabled: canReadIntegrations });

  if (!canReadIntegrations) {
    return undefined;
  }

  const broken = findBrokenIntegrations(integrations);
  if (broken.length === 0) {
    return undefined;
  }

  return (
    <BrokenIntegrationsBanner
      integrations={broken}
      canManageIntegrations={canAct("integrations", "update")}
      integrationsBasePath={factorySettingsSectionPath(organizationId, factoryKey, "organization", "integrations")}
    />
  );
}
