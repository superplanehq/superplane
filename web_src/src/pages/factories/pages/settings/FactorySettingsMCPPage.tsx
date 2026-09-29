import { Plug } from "lucide-react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { usePageTitle } from "@/hooks/usePageTitle";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { AgentResourceConnectionDialog } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { MCPAddPicker } from "./MCPAddPicker";
import { MCPCatalogSetupDialog } from "./MCPCatalogSetupDialog";
import { MCPServerList } from "./MCPServerList";
import { SuperPlaneMCPServerSection } from "./SuperPlaneMCPServerSection";
import { SUPERPLANE_MCP_SERVER_COPY } from "./superplaneMCPServerCopy";
import { useMCPPage } from "./useMCPPage";

export function FactorySettingsMCPPage() {
  const page = useMCPPage();
  usePageTitle([AGENT_RESOURCES_COPY.mcpTitle, "Settings", page.factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame
      title={AGENT_RESOURCES_COPY.mcpTitle}
      subtitle={
        page.showAgentMCP ? (
          <span className="flex flex-col gap-1">
            <span>{AGENT_RESOURCES_COPY.mcpHelper}</span>
            <span>{AGENT_RESOURCES_COPY.refinementNote}</span>
          </span>
        ) : (
          SUPERPLANE_MCP_SERVER_COPY.helper
        )
      }
      wide
      actions={
        page.showAgentMCP ? (
          <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button
              type="button"
              onClick={() => page.setAddPickerOpen(true)}
              disabled={!page.canUpdate}
              data-testid="agent-resources-add-connection"
            >
              {AGENT_RESOURCES_COPY.addConnection}
            </Button>
          </PermissionTooltip>
        ) : undefined
      }
    >
      <div className="flex flex-col gap-5" data-testid="factory-settings-mcp">
        {page.showAgentMCP ? <WorkspaceMCPServers page={page} /> : null}
        {page.showSuperPlaneMCP ? (
          <SuperPlaneMCPServerSection
            organizationId={page.organizationId}
            factoryId={page.factoryId}
            clients={page.mcpClients.data ?? []}
            isLoading={page.mcpClients.isLoading}
            isError={page.mcpClients.isError}
            canUpdate={page.canUpdate}
          />
        ) : null}
      </div>
      <MCPAddPicker
        open={page.addPickerOpen}
        onClose={() => page.setAddPickerOpen(false)}
        onSelect={page.openCatalogEntry}
      />
      <MCPCatalogSetupDialog
        open={page.catalogSetupOpen}
        entry={page.catalogEntry}
        resource={page.editResource}
        isSaving={page.isSaving}
        onClose={page.closeCatalogSetup}
        onSignIn={page.startCatalogOAuth}
        onSaveToken={page.saveCatalogToken}
      />
      <AgentResourceConnectionDialog
        open={page.connectionOpen}
        organizationId={page.organizationId}
        resource={page.editResource}
        isSaving={page.isSaving}
        onClose={page.closeConnection}
        onSave={page.saveConnection}
      />
      <FactoryDeleteDialog
        open={Boolean(page.pendingDelete)}
        factoryName={page.pendingDelete?.name?.trim() || AGENT_RESOURCES_COPY.unnamedResource}
        title={`Delete "${page.pendingDelete?.name?.trim() || AGENT_RESOURCES_COPY.unnamedResource}"?`}
        description={AGENT_RESOURCES_COPY.deleteDescription}
        canDelete={page.canUpdate}
        isDeleting={page.isDeleting}
        onClose={() => page.setPendingDelete(undefined)}
        onConfirm={page.confirmDelete}
      />
    </FactorySettingsPageFrame>
  );
}

function WorkspaceMCPServers({ page }: { page: ReturnType<typeof useMCPPage> }) {
  if (page.connections.isLoading) {
    return <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.loading}</p>;
  }
  if (page.connections.isError) {
    return <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.loadError}</p>;
  }
  if ((page.connections.data ?? []).length === 0) {
    return (
      <FactorySettingsCard data-testid="agent-resources-connections-empty">
        <Empty className="border-0 p-6 md:p-10">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Plug />
            </EmptyMedia>
            <EmptyTitle>{AGENT_RESOURCES_COPY.emptyConnectionsTitle}</EmptyTitle>
            <EmptyDescription>{AGENT_RESOURCES_COPY.emptyConnectionsBody}</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
              <Button type="button" onClick={() => page.setAddPickerOpen(true)} disabled={!page.canUpdate}>
                {AGENT_RESOURCES_COPY.addConnection}
              </Button>
            </PermissionTooltip>
          </EmptyContent>
        </Empty>
      </FactorySettingsCard>
    );
  }

  return (
    <FactorySettingsCard>
      <MCPServerList
        organizationId={page.organizationId}
        factoryId={page.factoryId}
        resources={page.connections.data ?? []}
        canUpdate={page.canUpdate}
        onEdit={page.setEditResource}
        onDelete={page.setPendingDelete}
        onDisconnect={page.disconnectResource}
        onToggleEnabled={page.toggleEnabled}
        onToggleTools={page.toggleTools}
        onConnect={(resource) => void page.startOAuthRedirect(resource)}
      />
    </FactorySettingsCard>
  );
}
