import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryMCPClients } from "@/hooks/useFactoryMCPClients";
import { FEATURE_SUPERPLANE_MCP_SERVER } from "@/lib/experimentalFeatures";

import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function useConnectPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const { has } = useExperimentalFeature(organizationId);
  const showConnect = has(FEATURE_SUPERPLANE_MCP_SERVER);
  const mcpClients = useFactoryMCPClients(organizationId, factoryId, showConnect);

  return {
    organizationId,
    factoryId,
    factory,
    canUpdate,
    showConnect,
    mcpClients,
  };
}
