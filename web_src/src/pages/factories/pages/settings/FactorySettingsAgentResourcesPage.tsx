import { MoreHorizontal } from "lucide-react";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { AgentResourceConnectionDialog } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import {
  connectionAuthLabel,
  connectionNeedsOAuthAction,
  connectionStatusLabel,
  skillSourceLabel,
} from "./agentResourceDisplay";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useAgentResourcesPage } from "./useAgentResourcesPage";

export function FactorySettingsAgentResourcesPage() {
  const page = useAgentResourcesPage();
  usePageTitle([AGENT_RESOURCES_COPY.title, "Settings", page.factory.name ?? "Workspace"]);

  return (
    <FactorySettingsPageFrame
      title={AGENT_RESOURCES_COPY.title}
      subtitle={
        <span className="flex flex-col gap-1">
          <span>{AGENT_RESOURCES_COPY.helper}</span>
          <span>{AGENT_RESOURCES_COPY.refinementNote}</span>
        </span>
      }
      wide
      actions={<AgentResourcesActions page={page} />}
    >
      <Tabs value={page.tab} onValueChange={page.setTab} data-testid="factory-settings-agent-resources">
        <TabsList>
          <TabsTrigger value="connections" data-testid="agent-resources-tab-connections">
            {AGENT_RESOURCES_COPY.connectionsTab}
          </TabsTrigger>
          <TabsTrigger value="skills" data-testid="agent-resources-tab-skills">
            {AGENT_RESOURCES_COPY.skillsTab}
          </TabsTrigger>
        </TabsList>
        <TabsContent value="connections" className="mt-4">
          <ConnectionsPanel
            canUpdate={page.canUpdate}
            isLoading={page.connections.isLoading}
            isError={page.connections.isError}
            resources={page.connections.data ?? []}
            onAdd={() => page.setAddDialogOpen(true)}
            onEdit={page.setEditResource}
            onDelete={page.setPendingDelete}
            onDisconnect={page.disconnectResource}
            onToggleEnabled={page.toggleEnabled}
            onConnect={(resource) => void page.startOAuthRedirect(resource)}
          />
        </TabsContent>
        <TabsContent value="skills" className="mt-4">
          <SkillsPanel
            canUpdate={page.canUpdate}
            isLoading={page.skills.isLoading}
            isError={page.skills.isError}
            resources={page.skills.data ?? []}
          />
        </TabsContent>
      </Tabs>
      <AgentResourcePageDialogs page={page} />
    </FactorySettingsPageFrame>
  );
}

function AgentResourcesActions({ page }: { page: ReturnType<typeof useAgentResourcesPage> }) {
  if (page.tab === "skills") {
    return (
      <Button
        type="button"
        disabled
        title={AGENT_RESOURCES_COPY.skillsUnavailable}
        data-testid="agent-resources-add-skill"
      >
        {AGENT_RESOURCES_COPY.addSkill}
      </Button>
    );
  }
  return (
    <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
      <Button
        type="button"
        onClick={() => page.setAddDialogOpen(true)}
        disabled={!page.canUpdate}
        data-testid="agent-resources-add-connection"
      >
        {AGENT_RESOURCES_COPY.addConnection}
      </Button>
    </PermissionTooltip>
  );
}

function AgentResourcePageDialogs({ page }: { page: ReturnType<typeof useAgentResourcesPage> }) {
  return (
    <>
      <AgentResourceConnectionDialog
        open={page.addDialogOpen || Boolean(page.editResource)}
        organizationId={page.organizationId}
        resource={page.editResource}
        isSaving={page.isSaving}
        onClose={() => {
          page.setEditResource(undefined);
          if (page.addDialogOpen) {
            page.setAddDialogOpen(false);
          }
        }}
        onSave={page.saveConnection}
      />
      <FactoryDeleteDialog
        open={Boolean(page.pendingDelete)}
        factoryName={page.pendingDelete?.name ?? ""}
        title={`Delete "${page.pendingDelete?.name ?? AGENT_RESOURCES_COPY.unnamedResource}"?`}
        description={AGENT_RESOURCES_COPY.deleteDescription}
        canDelete={page.canUpdate}
        isDeleting={page.isDeleting}
        onClose={() => page.setPendingDelete(undefined)}
        onConfirm={page.confirmDelete}
      />
    </>
  );
}

function ConnectionsPanel({
  canUpdate,
  isLoading,
  isError,
  resources,
  onAdd,
  onEdit,
  onDelete,
  onDisconnect,
  onToggleEnabled,
  onConnect,
}: {
  canUpdate: boolean;
  isLoading: boolean;
  isError: boolean;
  resources: FactoriesFactoryAgentResource[];
  onAdd: () => void;
  onEdit: (resource: FactoriesFactoryAgentResource) => void;
  onDelete: (resource: FactoriesFactoryAgentResource) => void;
  onDisconnect: (resource: FactoriesFactoryAgentResource) => void;
  onToggleEnabled: (resource: FactoriesFactoryAgentResource, enabled: boolean) => void;
  onConnect: (resource: FactoriesFactoryAgentResource) => void;
}) {
  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.loading}</p>;
  }
  if (isError) {
    return <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.loadError}</p>;
  }
  if (resources.length === 0) {
    return (
      <FactorySettingsCard data-testid="agent-resources-connections-empty">
        <p className="text-[13px] font-medium text-foreground">{AGENT_RESOURCES_COPY.emptyConnectionsTitle}</p>
        <p className="mt-1 text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.emptyConnectionsBody}</p>
        <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button type="button" className="mt-4" onClick={onAdd} disabled={!canUpdate}>
            {AGENT_RESOURCES_COPY.addConnection}
          </Button>
        </PermissionTooltip>
      </FactorySettingsCard>
    );
  }

  return (
    <FactorySettingsCard data-testid="agent-resources-connections-list">
      <ul className="divide-y divide-border">
        {resources.map((resource) => (
          <ConnectionRow
            key={resource.id ?? resource.name}
            resource={resource}
            canUpdate={canUpdate}
            onEdit={() => onEdit(resource)}
            onDelete={() => onDelete(resource)}
            onDisconnect={() => onDisconnect(resource)}
            onToggleEnabled={(enabled) => onToggleEnabled(resource, enabled)}
            onConnect={() => onConnect(resource)}
          />
        ))}
      </ul>
    </FactorySettingsCard>
  );
}

function ConnectionRow({
  resource,
  canUpdate,
  onEdit,
  onDelete,
  onDisconnect,
  onToggleEnabled,
  onConnect,
}: {
  resource: FactoriesFactoryAgentResource;
  canUpdate: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onDisconnect: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onConnect: () => void;
}) {
  const name = resource.name?.trim() || AGENT_RESOURCES_COPY.unnamedResource;
  const status = connectionStatusLabel(resource);
  const needsOAuth = connectionNeedsOAuthAction(resource);
  const reconnect =
    resource.oauthStatus === "OAUTH_STATUS_NEEDS_RECONNECT" || resource.oauthStatus === "OAUTH_STATUS_VENDOR_REJECTED";

  return (
    <li
      className="flex flex-wrap items-center gap-3 py-3 first:pt-0 last:pb-0"
      data-testid={`agent-resource-row-${resource.id}`}
    >
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
        <p className="truncate text-[12px] text-muted-foreground">{resource.url}</p>
        {resource.oauthError ? <p className="mt-1 text-[12px] text-destructive">{resource.oauthError}</p> : null}
      </div>
      <span className="text-[12px] text-muted-foreground">{connectionAuthLabel(resource.auth)}</span>
      <span className={cn("text-[12px]", reconnect ? "text-destructive" : "text-muted-foreground")}>{status}</span>
      <Switch
        checked={resource.enabled !== false}
        disabled={!canUpdate}
        onCheckedChange={onToggleEnabled}
        aria-label={`Enable ${name}`}
        data-testid={`agent-resource-enabled-${resource.id}`}
      />
      {needsOAuth ? (
        <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button type="button" size="sm" variant="outline" onClick={onConnect} disabled={!canUpdate}>
            {reconnect ? AGENT_RESOURCES_COPY.reconnect : AGENT_RESOURCES_COPY.connect}
          </Button>
        </PermissionTooltip>
      ) : null}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground"
            aria-label={`${name} menu`}
            data-testid={`agent-resource-menu-${resource.id}`}
          >
            <MoreHorizontal className="size-3.5" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem disabled={!canUpdate} onClick={onEdit}>
            {AGENT_RESOURCES_COPY.edit}
          </DropdownMenuItem>
          {resource.auth === "AUTH_OAUTH" && resource.oauthStatus === "OAUTH_STATUS_CONNECTED" ? (
            <DropdownMenuItem disabled={!canUpdate} onClick={onDisconnect}>
              {AGENT_RESOURCES_COPY.disconnect}
            </DropdownMenuItem>
          ) : null}
          <DropdownMenuItem disabled={!canUpdate} className="text-destructive" onClick={onDelete}>
            {AGENT_RESOURCES_COPY.delete}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </li>
  );
}

function SkillsPanel({
  canUpdate,
  isLoading,
  isError,
  resources,
}: {
  canUpdate: boolean;
  isLoading: boolean;
  isError: boolean;
  resources: FactoriesFactoryAgentResource[];
}) {
  if (isLoading) {
    return <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.loading}</p>;
  }
  if (isError) {
    return <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.loadError}</p>;
  }
  if (resources.length === 0) {
    return (
      <FactorySettingsCard data-testid="agent-resources-skills-empty">
        <p className="text-[13px] font-medium text-foreground">{AGENT_RESOURCES_COPY.emptySkillsTitle}</p>
        <p className="mt-1 text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.emptySkillsBody}</p>
        <Button type="button" className="mt-4" disabled title={AGENT_RESOURCES_COPY.skillsUnavailable}>
          {AGENT_RESOURCES_COPY.addSkill}
        </Button>
      </FactorySettingsCard>
    );
  }

  return (
    <FactorySettingsCard data-testid="agent-resources-skills-list">
      <ul className="divide-y divide-border">
        {resources.map((resource) => {
          const name = resource.name?.trim() || "skill";
          const source = skillSourceLabel(resource);
          return (
            <li
              key={resource.id ?? resource.name}
              className="flex flex-wrap items-center gap-3 py-3 first:pt-0 last:pb-0"
              data-testid={`agent-resource-skill-${resource.id}`}
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
                <p className="truncate text-[12px] text-muted-foreground">{source}</p>
              </div>
              <span className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.statusReady}</span>
              <Switch checked={resource.enabled !== false} disabled={!canUpdate} aria-label={`Enable ${name}`} />
            </li>
          );
        })}
      </ul>
    </FactorySettingsCard>
  );
}
