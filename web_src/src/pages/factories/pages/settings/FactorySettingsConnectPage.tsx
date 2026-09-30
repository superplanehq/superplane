import { useState } from "react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePageTitle } from "@/hooks/usePageTitle";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsPageFrame } from "./FactorySettingsCard";
import { SUPERPLANE_MCP_SERVER_COPY } from "./superplaneMCPServerCopy";
import { SuperPlaneMCPServerSection } from "./SuperPlaneMCPServerSection";
import { useConnectPage } from "./useConnectPage";

export function FactorySettingsConnectPage() {
  const page = useConnectPage();
  const [connectOpen, setConnectOpen] = useState(false);
  usePageTitle([AGENT_RESOURCES_COPY.superplaneMcpServerTitle, "Settings", page.factory.name ?? "Workspace"]);

  if (!page.showConnect) {
    return null;
  }

  return (
    <FactorySettingsPageFrame
      narrow
      title={AGENT_RESOURCES_COPY.superplaneMcpServerTitle}
      subtitle={AGENT_RESOURCES_COPY.superplaneMcpServerPageSubtitle}
      actions={
        <PermissionTooltip allowed={page.canUpdate} message={SUPERPLANE_MCP_SERVER_COPY.noUpdatePermission}>
          <Button
            type="button"
            size="sm"
            onClick={() => setConnectOpen(true)}
            disabled={!page.canUpdate}
            data-testid="superplane-mcp-connect-client"
          >
            {SUPERPLANE_MCP_SERVER_COPY.connectClient}
          </Button>
        </PermissionTooltip>
      }
    >
      <div data-testid="factory-settings-superplane-mcp-server">
        <SuperPlaneMCPServerSection
          organizationId={page.organizationId}
          factoryId={page.factoryId}
          clients={page.mcpClients.data ?? []}
          isLoading={page.mcpClients.isLoading}
          isError={page.mcpClients.isError}
          canUpdate={page.canUpdate}
          cardTitle={null}
          showCardDescription={false}
          connectAction="none"
          connectDialogOpen={connectOpen}
          onConnectDialogOpenChange={setConnectOpen}
        />
      </div>
    </FactorySettingsPageFrame>
  );
}
