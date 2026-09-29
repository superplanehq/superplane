import type { FactoriesFactoryAgentResource } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources, useFactoryAgentResourceTools } from "@/hooks/useFactoryAgentResources";
import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { ConnectionStatusDot } from "./settings/ConnectionStatusDot";
import { connectionIsEstablished } from "./settings/agentResourceDisplay";
import { AGENT_RESOURCES_COPY } from "./settings/agentResourceCopy";
import { skillDisplayTitle, skillListDescription } from "./settings/skillFrontmatter";
import { MCPToolsList } from "./settings/MCPToolsList";
import { enabledToolCount, mcpToolItems, nextDisabledTools, workspaceDisabledTools } from "./settings/mcpTools";
import { PLANNING_REVIEW_RESOURCES_COPY } from "./planningReviewResourcesCopy";

export function AgentResourcesEditor({
  organizationId,
  factoryId,
  factoryKey,
  disabledIds,
  disabledTools,
  onDisabledIdsChange,
  onDisabledToolsChange,
  compact = false,
}: {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  disabledIds: string[];
  disabledTools: Record<string, string[]>;
  onDisabledIdsChange: (ids: string[]) => void;
  onDisabledToolsChange: (tools: Record<string, string[]>) => void;
  compact?: boolean;
}) {
  const features = useExperimentalFeature(organizationId);
  const showMcp = Boolean(organizationId && factoryId && factoryKey && features.has(FEATURE_WORKSPACE_MCP));
  const showSkills = Boolean(organizationId && factoryId && factoryKey && features.has(FEATURE_WORKSPACE_SKILLS));
  if (!showMcp && !showSkills) {
    return null;
  }

  return (
    <div className="flex flex-col gap-4" data-testid="planning-review-resources">
      {showMcp ? (
        <MCPAutomationSection
          organizationId={organizationId!}
          factoryId={factoryId!}
          factoryKey={factoryKey!}
          disabledIds={disabledIds}
          disabledTools={disabledTools}
          onDisabledIdsChange={onDisabledIdsChange}
          onDisabledToolsChange={onDisabledToolsChange}
          compact={compact}
        />
      ) : null}
      {showSkills ? (
        <SkillsAutomationSection
          organizationId={organizationId!}
          factoryId={factoryId!}
          factoryKey={factoryKey!}
          disabledIds={disabledIds}
          onDisabledIdsChange={onDisabledIdsChange}
          compact={compact}
        />
      ) : null}
    </div>
  );
}

function MCPAutomationSection({
  organizationId,
  factoryId,
  factoryKey,
  disabledIds,
  disabledTools,
  onDisabledIdsChange,
  onDisabledToolsChange,
  compact,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  disabledIds: string[];
  disabledTools: Record<string, string[]>;
  onDisabledIdsChange: (ids: string[]) => void;
  onDisabledToolsChange: (tools: Record<string, string[]>) => void;
  compact: boolean;
}) {
  const connections = useFactoryAgentResources(organizationId, factoryId, "KIND_MCP_SERVER");
  const settingsHref = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "mcp");
  const resources = connections.data ?? [];
  return (
    <section
      className={cn("overflow-hidden rounded-xl border border-border bg-card shadow-sm", compact && "shadow-none")}
    >
      <SectionHeader title={AGENT_RESOURCES_COPY.mcpTitle} count={resources.length} settingsHref={settingsHref} />
      {connections.isLoading ? (
        <p className="px-5 py-6 text-sm text-muted-foreground">{PLANNING_REVIEW_RESOURCES_COPY.loading}</p>
      ) : connections.isError ? (
        <p className="px-5 py-6 text-sm text-destructive">{PLANNING_REVIEW_RESOURCES_COPY.loadError}</p>
      ) : resources.length === 0 ? (
        <EmptySettingsNote settingsHref={settingsHref} kind="mcp" />
      ) : (
        <ul className="divide-y divide-border">
          {resources.map((resource) => (
            <MCPAutomationRow
              key={resource.id || resource.name}
              organizationId={organizationId}
              factoryId={factoryId}
              resource={resource}
              disabledIds={disabledIds}
              disabledTools={disabledTools}
              onDisabledIdsChange={onDisabledIdsChange}
              onDisabledToolsChange={onDisabledToolsChange}
            />
          ))}
        </ul>
      )}
    </section>
  );
}

function MCPAutomationRow({
  organizationId,
  factoryId,
  resource,
  disabledIds,
  disabledTools,
  onDisabledIdsChange,
  onDisabledToolsChange,
}: {
  organizationId: string;
  factoryId: string;
  resource: FactoriesFactoryAgentResource;
  disabledIds: string[];
  disabledTools: Record<string, string[]>;
  onDisabledIdsChange: (ids: string[]) => void;
  onDisabledToolsChange: (tools: Record<string, string[]>) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const id = resource.id ?? "";
  const name = resource.name?.trim() || AGENT_RESOURCES_COPY.unnamedResource;
  const workspaceOff = resource.enabled === false;
  const checked = !workspaceOff && !disabledIds.includes(id);
  const showTools = connectionIsEstablished(resource);
  const toolsQuery = useFactoryAgentResourceTools(organizationId, factoryId, id, showTools && !workspaceOff);
  const workspaceDisabled = workspaceDisabledTools(resource);
  const automationDisabled = disabledTools[id] ?? [];
  const tools = mcpToolItems(toolsQuery.data);
  const effectiveDisabled = [...new Set([...workspaceDisabled, ...automationDisabled])];
  const count =
    showTools && !workspaceOff && !toolsQuery.isLoading && !toolsQuery.isError && tools.length > 0
      ? AGENT_RESOURCES_COPY.toolsCount(enabledToolCount(tools, effectiveDisabled), tools.length)
      : null;

  return (
    <li>
      <div className="flex flex-wrap items-center gap-3 px-5 py-3">
        <ConnectionStatusDot resource={resource} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
          <p className="truncate text-[12px] text-muted-foreground">{PLANNING_REVIEW_RESOURCES_COPY.kindMcp}</p>
          {workspaceOff ? (
            <p className="truncate text-[12px] text-muted-foreground">
              {PLANNING_REVIEW_RESOURCES_COPY.offForWorkspace}
            </p>
          ) : null}
        </div>
        {count ? (
          <span className="text-[12px] tabular-nums text-muted-foreground" data-testid={`mcp-tools-count-${id}`}>
            {count}
          </span>
        ) : null}
        {showTools && !workspaceOff ? (
          <Button type="button" size="sm" variant="ghost" onClick={() => setExpanded((open) => !open)}>
            <ChevronDown className={cn("size-3.5 transition-transform", expanded && "rotate-180")} aria-hidden />
            {AGENT_RESOURCES_COPY.toolsToggle}
          </Button>
        ) : null}
        <Switch
          checked={checked}
          disabled={workspaceOff || !id}
          onCheckedChange={(next) => onDisabledIdsChange(nextDisabledIds(disabledIds, id, next))}
          aria-label={PLANNING_REVIEW_RESOURCES_COPY.enableLabel(name)}
          data-testid={`planning-review-resource-${id}`}
        />
      </div>
      {expanded && showTools && !workspaceOff ? (
        <div className="px-5 pb-3">
          <MCPToolsList
            tools={toolsQuery.data ?? []}
            isLoading={toolsQuery.isLoading}
            isError={toolsQuery.isError}
            disabledTools={automationDisabled}
            lockedTools={workspaceDisabled}
            canUpdate
            onToggleTool={(toolName, enabled) =>
              onDisabledToolsChange(
                nextResourceTools(disabledTools, id, nextDisabledTools(automationDisabled, toolName, enabled)),
              )
            }
          />
        </div>
      ) : null}
    </li>
  );
}

function SkillsAutomationSection({
  organizationId,
  factoryId,
  factoryKey,
  disabledIds,
  onDisabledIdsChange,
  compact,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  disabledIds: string[];
  onDisabledIdsChange: (ids: string[]) => void;
  compact: boolean;
}) {
  const skills = useFactoryAgentResources(organizationId, factoryId, "KIND_SKILL");
  const settingsHref = factorySettingsSectionPath(organizationId, factoryKey, "workspace", "skills");
  const resources = skills.data ?? [];
  return (
    <section
      className={cn("overflow-hidden rounded-xl border border-border bg-card shadow-sm", compact && "shadow-none")}
    >
      <SectionHeader title={AGENT_RESOURCES_COPY.skillsTitle} count={resources.length} settingsHref={settingsHref} />
      {skills.isLoading ? (
        <p className="px-5 py-6 text-sm text-muted-foreground">{PLANNING_REVIEW_RESOURCES_COPY.loading}</p>
      ) : skills.isError ? (
        <p className="px-5 py-6 text-sm text-destructive">{PLANNING_REVIEW_RESOURCES_COPY.loadError}</p>
      ) : resources.length === 0 ? (
        <EmptySettingsNote settingsHref={settingsHref} kind="skills" />
      ) : (
        <ul className="divide-y divide-border">
          {resources.map((resource) => (
            <SkillAutomationRow
              key={resource.id || resource.name}
              resource={resource}
              disabledIds={disabledIds}
              onDisabledIdsChange={onDisabledIdsChange}
            />
          ))}
        </ul>
      )}
    </section>
  );
}

function SkillAutomationRow({
  resource,
  disabledIds,
  onDisabledIdsChange,
}: {
  resource: FactoriesFactoryAgentResource;
  disabledIds: string[];
  onDisabledIdsChange: (ids: string[]) => void;
}) {
  const id = resource.id ?? "";
  const name = skillDisplayTitle(resource) || AGENT_RESOURCES_COPY.unnamedSkill;
  const description = skillListDescription(resource);
  const workspaceOff = resource.enabled === false;
  const checked = !workspaceOff && !disabledIds.includes(id);
  return (
    <li className="flex flex-wrap items-center gap-3 px-5 py-3">
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
        {description ? <p className="truncate text-[12px] text-muted-foreground">{description}</p> : null}
        {workspaceOff ? (
          <p className="truncate text-[12px] text-muted-foreground">{PLANNING_REVIEW_RESOURCES_COPY.offForWorkspace}</p>
        ) : null}
      </div>
      <Switch
        checked={checked}
        disabled={workspaceOff || !id}
        onCheckedChange={(next) => onDisabledIdsChange(nextDisabledIds(disabledIds, id, next))}
        aria-label={PLANNING_REVIEW_RESOURCES_COPY.enableLabel(name)}
        data-testid={`planning-review-resource-${id}`}
      />
    </li>
  );
}

function SectionHeader({ title, count, settingsHref }: { title: string; count: number; settingsHref: string }) {
  return (
    <header className="flex items-center gap-2.5 border-b border-border px-5 py-3.5">
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      {count > 0 ? (
        <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium tabular-nums text-muted-foreground">
          {count}
        </span>
      ) : null}
      <span className="flex-1" />
      {count > 0 ? (
        <Button asChild variant="outline" size="sm">
          <Link href={settingsHref} data-testid="planning-review-resources-manage">
            {PLANNING_REVIEW_RESOURCES_COPY.manage}
          </Link>
        </Button>
      ) : null}
    </header>
  );
}

function EmptySettingsNote({ settingsHref, kind }: { settingsHref: string; kind: "mcp" | "skills" }) {
  return (
    <p
      className="px-5 py-10 text-center text-sm text-muted-foreground"
      data-testid={`planning-review-resources-empty-${kind}`}
    >
      Add {kind === "mcp" ? "MCP servers" : "skills"} on the{" "}
      <Link
        href={settingsHref}
        className="font-medium text-foreground underline underline-offset-2 hover:text-primary"
        data-testid="planning-review-resources-settings"
      >
        {PLANNING_REVIEW_RESOURCES_COPY.settingsLink}
      </Link>
      .
    </p>
  );
}

function nextDisabledIds(disabledIds: string[], id: string, enabled: boolean): string[] {
  if (!id) {
    return disabledIds;
  }
  if (enabled) {
    return disabledIds.filter((entry) => entry !== id);
  }
  if (disabledIds.includes(id)) {
    return disabledIds;
  }
  return [...disabledIds, id];
}

function nextResourceTools(
  current: Record<string, string[]>,
  resourceId: string,
  names: string[],
): Record<string, string[]> {
  const next = { ...current };
  if (names.length === 0) {
    delete next[resourceId];
    return next;
  }
  next[resourceId] = names;
  return next;
}
