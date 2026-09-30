import { BookOpen, Pencil } from "lucide-react";
import { Link, useNavigate } from "react-router";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { cn } from "@/lib/utils";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AgentSettingsSectionEmpty } from "./AgentSettingsSectionEmpty";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { skillDisplayTitle } from "./skillFrontmatter";

export function WorkspaceSkillsSection({
  organizationId,
  factoryId,
  factoryKey,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canUpdate: boolean;
}) {
  const navigate = useNavigate();
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");

  const editorPath = (resourceId?: string) => {
    const base = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "skills");
    return resourceId ? `${base}/${resourceId}` : `${base}/new`;
  };

  const addSkillAction = (
    <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
      <Button
        type="button"
        size="sm"
        onClick={() => navigate(editorPath())}
        disabled={!canUpdate}
        data-testid="agent-resources-add-skill"
      >
        {AGENT_RESOURCES_COPY.addSkill}
      </Button>
    </PermissionTooltip>
  );

  return (
    <div data-testid="factory-settings-skills">
      {skills.isLoading ? (
        <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.skillsLoading}</p>
      ) : skills.isError ? (
        <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.skillsLoadError}</p>
      ) : (skills.data ?? []).length === 0 ? (
        <FactorySettingsCard
          title={AGENT_RESOURCES_COPY.skillsTitle}
          description={AGENT_RESOURCES_COPY.skillsHelper}
          action={addSkillAction}
          data-testid="agent-resources-skills-empty"
        >
          <AgentSettingsSectionEmpty
            icon={BookOpen}
            title={AGENT_RESOURCES_COPY.emptySkillsTitle}
            description={AGENT_RESOURCES_COPY.emptySkillsBody}
          />
        </FactorySettingsCard>
      ) : (
        <FactorySettingsCard
          title={AGENT_RESOURCES_COPY.skillsTitle}
          description={AGENT_RESOURCES_COPY.skillsHelper}
          action={addSkillAction}
          attachedList
          data-testid="agent-resources-skills-list"
        >
          <ul className="divide-y divide-border border-t border-border">
            {(skills.data ?? []).map((resource) => (
              <SkillRow
                key={resource.id || resource.name}
                resource={resource}
                canUpdate={canUpdate}
                configureHref={resource.id ? editorPath(resource.id) : "#"}
              />
            ))}
          </ul>
        </FactorySettingsCard>
      )}
    </div>
  );
}

function SkillRow({
  resource,
  canUpdate,
  configureHref,
}: {
  resource: FactoriesFactoryAgentResource;
  canUpdate: boolean;
  configureHref: string;
}) {
  const name = skillDisplayTitle(resource) || AGENT_RESOURCES_COPY.unnamedSkill;
  const disabled = resource.enabled === false;

  return (
    <li
      className={cn("flex min-w-0 items-center gap-4 py-2", disabled && "opacity-60")}
      data-testid={`agent-resource-row-${resource.id}`}
      aria-disabled={disabled || undefined}
    >
      <Link
        to={configureHref}
        className="flex min-w-0 flex-1 items-center gap-2.5 overflow-hidden text-left"
        data-testid={`agent-resource-edit-${resource.id}`}
      >
        <BookOpen className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
        <span className="min-w-0 truncate text-[13px] font-medium text-foreground">{name}</span>
      </Link>
      <div className="flex shrink-0 items-center gap-1" data-testid={`agent-resource-actions-${resource.id}`}>
        {disabled ? (
          <span
            className={cn("whitespace-nowrap text-[12px] text-muted-foreground")}
            data-testid={`agent-resource-status-${resource.id}`}
          >
            {AGENT_RESOURCES_COPY.statusSkillDisabled}
          </span>
        ) : null}
        <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8 shrink-0 text-muted-foreground"
            asChild
            disabled={!canUpdate}
          >
            <Link
              to={configureHref}
              aria-label={AGENT_RESOURCES_COPY.editSkill}
              data-testid={`agent-resource-configure-${resource.id}`}
            >
              <Pencil className="size-4" strokeWidth={1.75} aria-hidden />
            </Link>
          </Button>
        </PermissionTooltip>
      </div>
    </li>
  );
}
