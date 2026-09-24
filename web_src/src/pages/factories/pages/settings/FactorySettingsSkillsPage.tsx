import { BookOpen, MoreHorizontal } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Switch } from "@/components/ui/switch";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useDeleteFactoryAgentResource,
  useFactoryAgentResources,
  useUpdateFactoryAgentResource,
} from "@/hooks/useFactoryAgentResources";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { skillSourceLabel } from "./agentResourceDisplay";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsSkillsPage() {
  const { organizationId, factoryId, factory } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canUpdate = canAct("factories", "update") && !permissionsLoading;
  const navigate = useNavigate();
  const factoryKey = factory.key ?? "";
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");
  const updateResource = useUpdateFactoryAgentResource(organizationId, factoryId);
  const deleteResource = useDeleteFactoryAgentResource(organizationId, factoryId);
  const [pendingDelete, setPendingDelete] = useState<FactoriesFactoryAgentResource | undefined>();
  usePageTitle([AGENT_RESOURCES_COPY.skillsTitle, "Settings", factory.name ?? "Workspace"]);

  const editorPath = (resourceId?: string) => {
    const base = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "skills");
    return resourceId ? `${base}/${resourceId}` : `${base}/new`;
  };

  const toggleEnabled = (resource: FactoriesFactoryAgentResource, enabled: boolean) => {
    if (!resource.id) {
      return;
    }
    void updateResource.mutateAsync({ resourceId: resource.id, enabled }).catch((error) => {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.skillUpdateFailed));
    });
  };

  const confirmDelete = async () => {
    if (!pendingDelete?.id) {
      return;
    }
    try {
      await deleteResource.mutateAsync(pendingDelete.id);
      showSuccessToast(AGENT_RESOURCES_COPY.skillDeleted);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.skillDeleteFailed));
      throw error;
    }
  };

  return (
    <FactorySettingsPageFrame
      title={AGENT_RESOURCES_COPY.skillsTitle}
      subtitle={AGENT_RESOURCES_COPY.skillsHelper}
      wide
      actions={
        <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button
            type="button"
            onClick={() => navigate(editorPath())}
            disabled={!canUpdate}
            data-testid="agent-resources-add-skill"
          >
            {AGENT_RESOURCES_COPY.addSkill}
          </Button>
        </PermissionTooltip>
      }
    >
      <div data-testid="factory-settings-skills">
        {skills.isLoading ? (
          <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.skillsLoading}</p>
        ) : skills.isError ? (
          <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.skillsLoadError}</p>
        ) : (skills.data ?? []).length === 0 ? (
          <FactorySettingsCard data-testid="agent-resources-skills-empty">
            <Empty className="border-0 p-6 md:p-10">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <BookOpen />
                </EmptyMedia>
                <EmptyTitle>{AGENT_RESOURCES_COPY.emptySkillsTitle}</EmptyTitle>
                <EmptyDescription>{AGENT_RESOURCES_COPY.emptySkillsBody}</EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
                  <Button type="button" onClick={() => navigate(editorPath())} disabled={!canUpdate}>
                    {AGENT_RESOURCES_COPY.addSkill}
                  </Button>
                </PermissionTooltip>
              </EmptyContent>
            </Empty>
          </FactorySettingsCard>
        ) : (
          <FactorySettingsCard data-testid="agent-resources-skills-list">
            <ul className="divide-y divide-border">
              {(skills.data ?? []).map((resource) => (
                <SkillRow
                  key={resource.id || resource.name}
                  resource={resource}
                  canUpdate={canUpdate}
                  onEdit={() => navigate(editorPath(resource.id))}
                  onDelete={() => setPendingDelete(resource)}
                  onToggleEnabled={(enabled) => toggleEnabled(resource, enabled)}
                />
              ))}
            </ul>
          </FactorySettingsCard>
        )}
      </div>
      <FactoryDeleteDialog
        open={Boolean(pendingDelete)}
        factoryName={pendingDelete?.name?.trim() || AGENT_RESOURCES_COPY.unnamedSkill}
        title={`Delete "${pendingDelete?.name?.trim() || AGENT_RESOURCES_COPY.unnamedSkill}"?`}
        description={AGENT_RESOURCES_COPY.deleteSkillDescription}
        canDelete={canUpdate}
        isDeleting={deleteResource.isPending}
        onClose={() => setPendingDelete(undefined)}
        onConfirm={confirmDelete}
      />
    </FactorySettingsPageFrame>
  );
}

function SkillRow({
  resource,
  canUpdate,
  onEdit,
  onDelete,
  onToggleEnabled,
}: {
  resource: FactoriesFactoryAgentResource;
  canUpdate: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onToggleEnabled: (enabled: boolean) => void;
}) {
  const name = resource.name?.trim() || AGENT_RESOURCES_COPY.unnamedSkill;
  return (
    <li
      className="flex flex-wrap items-center gap-3 py-3 first:pt-0 last:pb-0"
      data-testid={`agent-resource-row-${resource.id}`}
    >
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
        <p className="truncate text-[12px] text-muted-foreground">{skillSourceLabel(resource)}</p>
      </div>
      <Switch
        checked={resource.enabled !== false}
        disabled={!canUpdate}
        onCheckedChange={onToggleEnabled}
        aria-label={`Enable ${name}`}
        data-testid={`agent-resource-enabled-${resource.id}`}
      />
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
          <DropdownMenuItem disabled={!canUpdate} className="text-destructive" onClick={onDelete}>
            {AGENT_RESOURCES_COPY.delete}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </li>
  );
}
