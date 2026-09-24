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
import { MCPServerList } from "./MCPServerList";
import { useMCPPage } from "./useMCPPage";

export function FactorySettingsMCPPage() {
  const page = useMCPPage();
  usePageTitle([AGENT_RESOURCES_COPY.mcpTitle, "Settings", page.factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame
      title={AGENT_RESOURCES_COPY.mcpTitle}
      subtitle={
        <span className="flex flex-col gap-1">
          <span>{AGENT_RESOURCES_COPY.mcpHelper}</span>
          <span>{AGENT_RESOURCES_COPY.refinementNote}</span>
        </span>
      }
      wide
      actions={
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
      }
    >
      <div data-testid="factory-settings-mcp">
        {page.connections.isLoading ? (
          <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.loading}</p>
        ) : page.connections.isError ? (
          <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.loadError}</p>
        ) : (page.connections.data ?? []).length === 0 ? (
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
        ) : (
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
        )}
      </div>
      <MCPAddPicker
        open={page.addPickerOpen}
        onClose={() => page.setAddPickerOpen(false)}
        onSelect={page.openCatalogEntry}
      />
      <AgentResourceConnectionDialog
        open={page.connectionOpen}
        organizationId={page.organizationId}
        resource={page.editResource}
        defaults={page.catalogDefaults}
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
