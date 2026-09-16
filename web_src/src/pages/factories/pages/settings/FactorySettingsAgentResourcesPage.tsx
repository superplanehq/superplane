import { useState } from "react";
import { MoreHorizontal } from "lucide-react";
import { useSearchParams } from "react-router";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryAgentResource,
  useDeleteFactoryAgentResource,
  useDisconnectFactoryAgentResourceOAuth,
  useFactoryAgentResources,
  useStartFactoryAgentResourceOAuth,
  useUpdateFactoryAgentResource,
} from "@/hooks/useFactoryAgentResources";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Switch } from "@/ui/switch";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { AgentResourceConnectionDialog, type AgentResourceConnectionDraft } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import {
  connectionAuthLabel,
  connectionNeedsOAuthAction,
  connectionStatusLabel,
  skillSourceLabel,
} from "./agentResourceDisplay";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

type AgentResourcesTab = "connections" | "skills";

function readTab(searchParams: URLSearchParams): AgentResourcesTab {
  return searchParams.get("tab") === "skills" ? "skills" : "connections";
}

export function FactorySettingsAgentResourcesPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const [searchParams, setSearchParams] = useSearchParams();
  const tab = readTab(searchParams);
  const addDialogOpen = searchParams.get("dialog") === "add";
  const [editResource, setEditResource] = useState<FactoriesFactoryAgentResource | undefined>();
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactoryAgentResource | undefined>();

  const connections = useFactoryAgentResources(organizationId, factoryId, "KIND_MCP_SERVER");
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");
  const createResource = useCreateFactoryAgentResource(organizationId, factoryId);
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const startOAuth = useStartFactoryAgentResourceOAuth(organizationId, factoryId);
  const disconnectOAuth = useDisconnectFactoryAgentResourceOAuth(organizationId, factoryId);

  usePageTitle([AGENT_RESOURCES_COPY.title, "Settings", factory.name ?? "Workspace"]);

  const setTab = (nextTab: string) => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (nextTab === "skills") {
          next.set("tab", "skills");
        } else {
          next.delete("tab");
        }
        next.delete("dialog");
        return next;
      },
      { replace: true },
    );
  };

  const setAddDialogOpen = (open: boolean) => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (open) {
          next.set("dialog", "add");
        } else {
          next.delete("dialog");
        }
        return next;
      },
      { replace: true },
    );
  };

  const startOAuthRedirect = async (resource: FactoriesFactoryAgentResource) => {
    if (!resource.id) {
      return;
    }
    try {
      const result = await startOAuth.mutateAsync(resource.id);
      if (result.authorizationUrl) {
        window.location.assign(result.authorizationUrl);
      }
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.connectFailed));
    }
  };

  const saveConnection = async (draft: AgentResourceConnectionDraft) => {
    try {
      if (editResource?.id) {
        await updateResource.mutateAsync({
          resourceId: editResource.id,
          name: draft.name,
          url: draft.url,
          auth: draft.auth,
          headers: draft.headers,
        });
        showSuccessToast(AGENT_RESOURCES_COPY.updated);
        setEditResource(undefined);
        return;
      }
      await createResource.mutateAsync({
        kind: "KIND_MCP_SERVER",
        name: draft.name,
        enabled: true,
        url: draft.url,
        auth: draft.auth,
        headers: draft.headers,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.created);
      setAddDialogOpen(false);
    } catch (error) {
      showErrorToast(
        getApiErrorMessage(error, editResource ? AGENT_RESOURCES_COPY.updateFailed : AGENT_RESOURCES_COPY.createFailed),
      );
      throw error;
    }
  };

  const connectionDialogOpen = addDialogOpen || Boolean(editResource);
  const isSaving = createResource.isPending || updateResource.isPending;

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
      actions={
        tab === "skills" ? (
          <Button
            type="button"
            disabled
            title={AGENT_RESOURCES_COPY.skillsUnavailable}
            data-testid="agent-resources-add-skill"
          >
            {AGENT_RESOURCES_COPY.addSkill}
          </Button>
        ) : (
          <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button
              type="button"
              onClick={() => setAddDialogOpen(true)}
              disabled={!canUpdate}
              data-testid="agent-resources-add-connection"
            >
              {AGENT_RESOURCES_COPY.addConnection}
            </Button>
          </PermissionTooltip>
        )
      }
    >
      <Tabs value={tab} onValueChange={setTab} data-testid="factory-settings-agent-resources">
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
            canUpdate={canUpdate}
            isLoading={connections.isLoading}
            isError={connections.isError}
            resources={connections.data ?? []}
            onAdd={() => setAddDialogOpen(true)}
            onEdit={setEditResource}
            onDelete={setPendingDelete}
            onDisconnect={(resource) => {
              if (!resource.id) {
                return;
              }
              void disconnectOAuth.mutateAsync(resource.id).then(
                () => showSuccessToast(AGENT_RESOURCES_COPY.disconnected),
                (error) => showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.disconnectFailed)),
              );
            }}
            onToggleEnabled={(resource, enabled) => {
              if (!resource.id) {
                return;
              }
              void updateResource.mutateAsync({ resourceId: resource.id, enabled }).catch((error) => {
                showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.updateFailed));
              });
            }}
            onConnect={(resource) => void startOAuthRedirect(resource)}
          />
        </TabsContent>
        <TabsContent value="skills" className="mt-4">
          <SkillsPanel
            canUpdate={canUpdate}
            isLoading={skills.isLoading}
            isError={skills.isError}
            resources={skills.data ?? []}
          />
        </TabsContent>
      </Tabs>

      <AgentResourceConnectionDialog
        open={connectionDialogOpen}
        organizationId={organizationId}
        resource={editResource}
        isSaving={isSaving}
        onClose={() => {
          setEditResource(undefined);
          if (addDialogOpen) {
            setAddDialogOpen(false);
          }
        }}
        onSave={saveConnection}
      />

      <FactoryDeleteDialog
        open={Boolean(pendingDelete)}
        factoryName={pendingDelete?.name ?? ""}
        title={`Delete "${pendingDelete?.name ?? "connection"}"?`}
        description="This removes the connection for every agent in this workspace."
        canDelete={canUpdate}
        isDeleting={deleteResource.isPending}
        onClose={() => setPendingDelete(undefined)}
        onConfirm={async () => {
          if (!pendingDelete?.id) {
            return;
          }
          try {
            await deleteResource.mutateAsync(pendingDelete.id);
            showSuccessToast(AGENT_RESOURCES_COPY.deleted);
          } catch (error) {
            showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.deleteFailed));
            throw error;
          }
        }}
      />
    </FactorySettingsPageFrame>
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
  const name = resource.name?.trim() || "connection";
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
