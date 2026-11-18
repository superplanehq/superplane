import { usePermissions } from "@/contexts/usePermissions";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";

import { BrokenIntegrationsBanner, MissingGitConnectionBanner } from "../BrokenIntegrationsBanner";
import { factoryAgentPath, factorySettingsSectionPath } from "./factoryPagePaths";
import { findBrokenIntegrations, workspaceNeedsGitConnection } from "./brokenIntegrations";

/**
 * Surfaces organization integrations that broke after setup (an error
 * state) or were left half-configured (stuck pending), and a workspace that
 * lost its Git connection. Runs that depend on these integrations fail
 * silently otherwise, so this gives operators a workspace-level signal and a
 * link to the repair step.
 */
export function useBrokenIntegrationsBanner(
  organizationId: string,
  factoryKey: string,
  factory: Parameters<typeof workspaceNeedsGitConnection>[0],
) {
  const { canAct } = usePermissions();
  const canReadIntegrations = canAct("integrations", "read");
  const { data: integrations = [] } = useConnectedIntegrations(organizationId, { enabled: canReadIntegrations });

  const broken = canReadIntegrations ? findBrokenIntegrations(integrations) : [];
  const needsGitConnection = workspaceNeedsGitConnection(factory);
  if (broken.length === 0 && !needsGitConnection) {
    return undefined;
  }

  return (
    <>
      {needsGitConnection ? (
        <MissingGitConnectionBanner
          agentPath={canAct("factories", "update") ? factoryAgentPath(organizationId, factoryKey) : undefined}
        />
      ) : null}
      {broken.length > 0 ? (
        <BrokenIntegrationsBanner
          integrations={broken}
          canManageIntegrations={canAct("integrations", "update")}
          integrationsBasePath={factorySettingsSectionPath(organizationId, factoryKey, "organization", "integrations")}
        />
      ) : null}
    </>
  );
}
