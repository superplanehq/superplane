import type { FactoriesFactoryAgentResource } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";
import { FEATURE_WORKSPACE_AGENT_RESOURCES } from "@/lib/experimentalFeatures";

import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { PLANNING_REVIEW_RESOURCES_COPY } from "./planningReviewResourcesCopy";

export function PlanningReviewResourcesCard({
  organizationId,
  factoryId,
  factoryKey,
  disabledIds,
  onDisabledIdsChange,
}: {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  disabledIds: string[];
  onDisabledIdsChange: (ids: string[]) => void;
}) {
  const state = usePlanningReviewResources(organizationId, factoryId, factoryKey);
  if (!state) {
    return null;
  }

  return (
    <section
      className="overflow-hidden rounded-xl border border-border bg-card shadow-sm"
      aria-label={PLANNING_REVIEW_RESOURCES_COPY.title}
      data-testid="planning-review-resources"
    >
      <ResourcesCardHeader count={state.resources.length} settingsHref={state.settingsHref} />
      <ResourcesCardBody
        resources={state.resources}
        settingsHref={state.settingsHref}
        isLoading={state.isLoading}
        isError={state.isError}
        disabledIds={disabledIds}
        onDisabledIdsChange={onDisabledIdsChange}
      />
    </section>
  );
}

function usePlanningReviewResources(organizationId?: string, factoryId?: string, factoryKey?: string) {
  const features = useExperimentalFeature(organizationId);
  const enabled = Boolean(organizationId && factoryId && factoryKey && features.has(FEATURE_WORKSPACE_AGENT_RESOURCES));
  const connections = useFactoryAgentResources(organizationId ?? "", factoryId ?? "", "KIND_MCP_SERVER", enabled);
  const skills = useFactoryAgentResources(organizationId ?? "", factoryId ?? "", "KIND_SKILL", enabled);
  if (!enabled) {
    return null;
  }
  return {
    resources: [...(connections.data ?? []), ...(skills.data ?? [])],
    settingsHref: factorySettingsSectionPath(organizationId ?? "", factoryKey ?? "", "workspace", "agent-resources"),
    isLoading: connections.isLoading || skills.isLoading,
    isError: connections.isError || skills.isError,
  };
}

function ResourcesCardHeader({ count, settingsHref }: { count: number; settingsHref: string }) {
  return (
    <header className="flex items-center gap-2.5 border-b border-border px-5 py-3.5">
      <h3 className="text-sm font-semibold text-foreground">{PLANNING_REVIEW_RESOURCES_COPY.title}</h3>
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

function ResourcesCardBody({
  resources,
  settingsHref,
  isLoading,
  isError,
  disabledIds,
  onDisabledIdsChange,
}: {
  resources: FactoriesFactoryAgentResource[];
  settingsHref: string;
  isLoading: boolean;
  isError: boolean;
  disabledIds: string[];
  onDisabledIdsChange: (ids: string[]) => void;
}) {
  if (isLoading) {
    return <p className="px-5 py-6 text-sm text-muted-foreground">{PLANNING_REVIEW_RESOURCES_COPY.loading}</p>;
  }
  if (isError) {
    return <p className="px-5 py-6 text-sm text-destructive">{PLANNING_REVIEW_RESOURCES_COPY.loadError}</p>;
  }
  if (resources.length === 0) {
    return (
      <p className="px-5 py-10 text-center text-sm text-muted-foreground" data-testid="planning-review-resources-empty">
        Add MCP servers and skills on the{" "}
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

  return (
    <ul className="divide-y divide-border">
      {resources.map((resource) => (
        <ResourceRow
          key={resource.id || resource.name}
          resource={resource}
          disabledIds={disabledIds}
          onDisabledIdsChange={onDisabledIdsChange}
        />
      ))}
    </ul>
  );
}

function ResourceRow({
  resource,
  disabledIds,
  onDisabledIdsChange,
}: {
  resource: FactoriesFactoryAgentResource;
  disabledIds: string[];
  onDisabledIdsChange: (ids: string[]) => void;
}) {
  const id = resource.id ?? "";
  const name = resource.name?.trim() || (resource.kind === "KIND_SKILL" ? "skill" : "MCP server");
  const workspaceOff = resource.enabled === false;
  const checked = !workspaceOff && !disabledIds.includes(id);
  return (
    <li className="flex flex-wrap items-center gap-3 px-5 py-3">
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
        <p className="truncate text-[12px] text-muted-foreground">
          {resource.kind === "KIND_SKILL"
            ? PLANNING_REVIEW_RESOURCES_COPY.kindSkill
            : PLANNING_REVIEW_RESOURCES_COPY.kindMcp}
        </p>
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
