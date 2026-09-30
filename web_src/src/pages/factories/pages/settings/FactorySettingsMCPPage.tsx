import { Plug } from "lucide-react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { usePageTitle } from "@/hooks/usePageTitle";

import { AgentResourceConnectionDialog } from "./AgentResourceConnectionDialog";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { MCPAddPicker } from "./MCPAddPicker";
import { MCPCatalogSetupDialog } from "./MCPCatalogSetupDialog";
import { MCPServerList } from "./MCPServerList";
import { SuperPlaneMCPServerSection } from "./SuperPlaneMCPServerSection";
import { useMCPPage } from "./useMCPPage";

export function FactorySettingsMCPPage() {
  const page = useMCPPage();
  usePageTitle([AGENT_RESOURCES_COPY.mcpTitle, "Settings", page.factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame title={AGENT_RESOURCES_COPY.mcpTitle} subtitle={AGENT_RESOURCES_COPY.mcpPageSubtitle}>
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
    </FactorySettingsPageFrame>
  );
}

function AddMCPConnectionButton({ canUpdate, onClick }: { canUpdate: boolean; onClick: () => void }) {
  return (
    <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
      <Button
        type="button"
        size="sm"
        onClick={onClick}
        disabled={!canUpdate}
        data-testid="agent-resources-add-connection"
      >
        {AGENT_RESOURCES_COPY.connectMcpServer}
      </Button>
    </PermissionTooltip>
  );
}

function WorkspaceMCPServers({ page }: { page: ReturnType<typeof useMCPPage> }) {
  const factoryKey = page.factory.key ?? "";
  const configurePath = (resourceId: string) =>
    `${factorySettingsSectionPath(page.organizationId, factoryKey, "workspace", "mcp")}/${resourceId}`;
  const addConnectionAction = (
    <AddMCPConnectionButton canUpdate={page.canUpdate} onClick={() => page.setAddPickerOpen(true)} />
  );
  if (page.connections.isLoading) {
    return <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.loading}</p>;
  }
  if (page.connections.isError) {
    return <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.loadError}</p>;
  }
  if ((page.connections.data ?? []).length === 0) {
    return (
      <FactorySettingsCard
        title={AGENT_RESOURCES_COPY.mcpConnectionsSectionTitle}
        description={AGENT_RESOURCES_COPY.mcpConnectionsSectionDescription}
        action={addConnectionAction}
        data-testid="agent-resources-connections-empty"
      >
        <Empty className="flex-none border-0 p-6 md:p-8">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Plug />
            </EmptyMedia>
            <EmptyTitle>{AGENT_RESOURCES_COPY.emptyConnectionsTitle}</EmptyTitle>
            <EmptyDescription>{AGENT_RESOURCES_COPY.emptyConnectionsBody}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </FactorySettingsCard>
    );
  }

  return (
    <FactorySettingsCard
      title={AGENT_RESOURCES_COPY.mcpConnectionsSectionTitle}
      description={AGENT_RESOURCES_COPY.mcpConnectionsSectionDescription}
      action={addConnectionAction}
      attachedList
    >
      <MCPServerList
        resources={page.connections.data ?? []}
        canUpdate={page.canUpdate}
        configurePath={configurePath}
      />
    </FactorySettingsCard>
  );
}
