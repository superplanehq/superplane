import { Plug } from "lucide-react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { usePageTitle } from "@/hooks/usePageTitle";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { AgentResourceConnectionDialog } from "./AgentResourceConnectionDialog";
import { AgentSettingsSectionEmpty } from "./AgentSettingsSectionEmpty";
import { factoryRouteSegment, factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { MCPAddPicker } from "./MCPAddPicker";
import { MCPCatalogSetupDialog } from "./MCPCatalogSetupDialog";
import { MCPServerList } from "./MCPServerList";
import { useMCPPage } from "./useMCPPage";
import { WorkspaceSkillsSection } from "./WorkspaceSkillsSection";

export function FactorySettingsAgentPage() {
  const page = useMCPPage();
  const { has } = useExperimentalFeature(page.organizationId);
  const showSkills = has(FEATURE_WORKSPACE_SKILLS);
  const showMcp = page.showAgentMCP;
  usePageTitle([AGENT_RESOURCES_COPY.agentTitle, "Settings", page.factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame
      narrow
      title={AGENT_RESOURCES_COPY.agentTitle}
      subtitle={AGENT_RESOURCES_COPY.agentPageSubtitle}
    >
      <div className="flex flex-col gap-5" data-testid="factory-settings-agent">
        {showMcp ? <WorkspaceMCPServers page={page} /> : null}
        {showSkills ? (
          <WorkspaceSkillsSection
            organizationId={page.organizationId}
            factoryId={page.factoryId}
            factoryKey={factoryRouteSegment(page.factory)}
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
  const factoryKey = factoryRouteSegment(page.factory);
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
        <AgentSettingsSectionEmpty
          icon={Plug}
          title={AGENT_RESOURCES_COPY.emptyConnectionsTitle}
          description={AGENT_RESOURCES_COPY.emptyConnectionsBody}
        />
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
        organizationId={page.organizationId}
        factoryId={page.factoryId}
        resources={page.connections.data ?? []}
        canUpdate={page.canUpdate}
        configurePath={configurePath}
      />
    </FactorySettingsCard>
  );
}
