import type { FactoriesFactoryLine, FactoryApp, FactoryLineStep } from "@/api-client";
import { Icon } from "@/components/Icon";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps } from "@/hooks/useFactoryData";
import { useCanvas, useCreateCanvas } from "@/hooks/useCanvasData";
import { factoryAppsKey } from "@/hooks/useFactoryData";
import { appPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { ChevronRight, ExternalLink, MoreHorizontal, Pencil, Plus, Workflow } from "lucide-react";
import { useMemo, useState } from "react";
import { NavLink, useNavigate, useParams } from "react-router-dom";
import { CreateFactoryAppDialog } from "../CreateFactoryAppDialog";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { listTriggerNodes } from "../lib/factoryCanvasTriggers";
import { createFactoryLinePath, editFactoryLinePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  factoryCardClassName,
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";

export function AutomationsPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();

  const { data: apps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);
  const { lineId: routeLineId } = useParams<{ lineId: string }>();

  const canUpdate = canAct("factories", "update");
  const canCreateApp = canAct("canvases", "create");

  const lines = useMemo(() => factory?.lines ?? [], [factory?.lines]);
  const selectedLineId = routeLineId ?? lines[0]?.id ?? undefined;
  const selectedLine = useMemo(
    () => lines.find((line) => line.id === selectedLineId) ?? lines[0],
    [lines, selectedLineId],
  );

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            Automations
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            Lines specialize how work moves through the workspace. Each phase runs an app.
          </p>
        </div>
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You don't have permission to create lines."
        >
          <Button type="button" asChild disabled={!canUpdate} data-testid="automations-create-line-button">
            <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryId) : "#"}>
              <Icon name="plus" />
              New Line
            </Link>
          </Button>
        </PermissionTooltip>
      </header>

      <div className={factoryContentBodyClassName}>
        <div className="grid gap-6 lg:grid-cols-[minmax(0,220px)_minmax(0,1fr)]">
          <LinesList
            organizationId={organizationId}
            factoryId={factoryId}
            lines={lines}
            selectedLineId={selectedLine?.id}
            canUpdate={canUpdate}
          />
          <div className="space-y-6">
            {selectedLine ? (
              <SelectedLineCard
                organizationId={organizationId}
                factoryId={factoryId}
                line={selectedLine}
                apps={apps}
                canUpdate={canUpdate}
              />
            ) : (
              <EmptyLinesCard
                organizationId={organizationId}
                factoryId={factoryId}
                canUpdate={canUpdate || permissionsLoading}
              />
            )}
            <AppsCard
              organizationId={organizationId}
              factoryId={factoryId}
              apps={apps}
              isLoading={appsLoading}
              canCreate={canCreateApp}
              permissionsLoading={permissionsLoading}
            />
          </div>
        </div>
      </div>
    </>
  );
}

function LinesList({
  organizationId,
  factoryId,
  lines,
  selectedLineId,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  lines: FactoriesFactoryLine[];
  selectedLineId?: string;
  canUpdate: boolean;
}) {
  return (
    <aside data-testid="automations-lines-list">
      <div className="flex items-center justify-between px-1 pb-2">
        <p className="text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">Lines</p>
        {canUpdate ? (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            asChild
            className="h-6 w-6 p-0 text-muted-foreground"
            aria-label="Create line"
            data-testid="automations-lines-list-create"
          >
            <Link href={createFactoryLinePath(organizationId, factoryId)}>
              <Plus className="h-3.5 w-3.5" aria-hidden />
            </Link>
          </Button>
        ) : null}
      </div>

      {lines.length === 0 ? (
        <p className="px-1 py-4 text-[13px] text-muted-foreground">No lines yet.</p>
      ) : (
        <ul className="flex flex-col gap-0.5">
          {lines.map((line) => {
            if (!line.id) {
              return null;
            }
            return (
              <LinesListRow
                key={line.id}
                line={line}
                isActive={line.id === selectedLineId}
                href={factoryLineDetailPath(organizationId, factoryId, line.id)}
              />
            );
          })}
        </ul>
      )}
    </aside>
  );
}

function LinesListRow({ line, isActive, href }: { line: FactoriesFactoryLine; isActive: boolean; href: string }) {
  const stepsCount = line.steps?.length ?? 0;
  return (
    <li>
      <NavLink
        to={href}
        className={cn(
          "flex items-center gap-2.5 rounded-md px-2 py-2 text-[13px] text-foreground/80 transition-colors hover:bg-accent hover:text-foreground",
          isActive && "bg-accent text-foreground",
        )}
        data-testid={`automations-line-${line.id}`}
      >
        <Workflow className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className={cn("truncate text-[13px]", isActive ? "font-medium" : "font-normal")}>
            {line.name || "Unnamed line"}
          </p>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            {stepsCount} {stepsCount === 1 ? "phase" : "phases"}
          </p>
        </div>
      </NavLink>
    </li>
  );
}

function SelectedLineCard({
  organizationId,
  factoryId,
  line,
  apps,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  apps: FactoryApp[];
  canUpdate: boolean;
}) {
  const editHref = line.id ? editFactoryLinePath(organizationId, factoryId, line.id) : "#";
  const steps = line.steps ?? [];
  const appNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const app of apps) {
      if (app.id && app.name) {
        map.set(app.id, app.name);
      }
    }
    return map;
  }, [apps]);

  return (
    <section className={cn("overflow-hidden", factoryCardClassName)} data-testid="automations-selected-line">
      <div className="flex items-center justify-between gap-4 border-b border-border px-5 py-4">
        <div className="flex min-w-0 items-center gap-2">
          <Workflow className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
          <h2 className="truncate text-[14px] font-medium tracking-[-0.01em] text-foreground">
            {line.name || "Unnamed line"}
          </h2>
        </div>
        {canUpdate ? (
          <Link
            href={editHref}
            className="inline-flex shrink-0 items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
            data-testid="automations-line-edit"
          >
            <Pencil className="h-3.5 w-3.5" aria-hidden />
            Edit
          </Link>
        ) : null}
      </div>

      <div className="px-5 py-5">
        {steps.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">No phases yet. Edit this line to add app-driven phases.</p>
        ) : (
          <ol className="flex flex-col gap-4" data-testid="automations-line-steps">
            {steps.map((step, index) => (
              <LineStepRow
                key={`${step.name}-${index}`}
                organizationId={organizationId}
                index={index + 1}
                step={step}
                appName={resolveStepAppName(step, appNameById)}
              />
            ))}
          </ol>
        )}
      </div>
    </section>
  );
}

function resolveStepAppName(step: FactoryLineStep, appNameById: Map<string, string>): string | undefined {
  const appId = step.app?.app;
  if (!appId) {
    return undefined;
  }
  return appNameById.get(appId);
}

/**
 * Resolves the human-readable trigger label for a step by fetching the app's
 * canvas and looking up the trigger node whose id matches `step.app.entrypoint`.
 * `useCanvas` is keyed on (organizationId, appId), so react-query dedupes when
 * multiple steps share the same app.
 */
function useStepTriggerName(organizationId: string, step: FactoryLineStep): string | undefined {
  const appId = step.app?.app;
  const entrypointId = step.app?.entrypoint;
  const { data: canvas } = useCanvas(organizationId, appId ?? "", {
    enabled: Boolean(appId && entrypointId),
    staleTime: 60_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });
  if (!entrypointId) {
    return undefined;
  }
  const trigger = listTriggerNodes(canvas).find((node) => node.id === entrypointId);
  return trigger?.name || trigger?.id;
}

function LineStepRow({
  organizationId,
  index,
  step,
  appName,
}: {
  organizationId: string;
  index: number;
  step: FactoryLineStep;
  appName?: string;
}) {
  const triggerName = useStepTriggerName(organizationId, step);
  const subtitleParts = [appName, triggerName].filter((part): part is string => Boolean(part));

  return (
    <li className="flex items-start gap-3">
      <span
        className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-muted text-[11px] font-medium text-muted-foreground"
        aria-hidden
      >
        {index}
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-medium text-foreground">{step.name || "Unnamed step"}</p>
        {subtitleParts.length > 0 ? (
          <p className="mt-0.5 truncate text-[12px] text-muted-foreground">{subtitleParts.join(" · ")}</p>
        ) : null}
      </div>
    </li>
  );
}

function EmptyLinesCard({
  organizationId,
  factoryId,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  canUpdate: boolean;
}) {
  return (
    <div
      className="flex flex-col items-center rounded-lg border border-dashed border-border bg-card px-6 py-12 text-center"
      data-testid="automations-empty-state"
    >
      <Workflow className="h-8 w-8 text-muted-foreground" aria-hidden />
      <p className="mt-3 text-[15px] font-medium text-foreground">No lines yet</p>
      <p className="mt-1 max-w-md text-[13px] text-muted-foreground">
        Lines define how work orders flow through your apps.
      </p>
      <Button type="button" asChild className="mt-6" disabled={!canUpdate}>
        <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryId) : "#"}>
          <Icon name="plus" />
          Create line
        </Link>
      </Button>
    </div>
  );
}

function AppsCard({
  organizationId,
  factoryId,
  apps,
  isLoading,
  canCreate,
  permissionsLoading,
}: {
  organizationId: string;
  factoryId: string;
  apps: FactoryApp[];
  isLoading: boolean;
  canCreate: boolean;
  permissionsLoading: boolean;
}) {
  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);

  const handleCreateApp = async (input: { name: string; description: string }) => {
    try {
      const result = await createCanvas.mutateAsync({
        name: input.name,
        description: input.description,
        factoryId,
        method: "ui",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      setCreateOpen(false);
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      if (canvasId) {
        showSuccessToast("App created.");
        navigate(appPath(organizationId, canvasId, "?edit=1"));
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create app"));
      throw error;
    }
  };

  return (
    <section className={cn("overflow-hidden", factoryCardClassName)} data-testid="automations-apps-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <MoreHorizontal className="h-4 w-4 text-muted-foreground" aria-hidden />
          <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Apps</h2>
          <span
            className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground"
            data-testid="automations-apps-count"
          >
            {apps.length}
          </span>
        </div>
        <PermissionTooltip
          allowed={canCreate || permissionsLoading}
          message="You don't have permission to create apps."
        >
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={() => setCreateOpen(true)}
            disabled={!canCreate}
            className="text-muted-foreground"
            data-testid="automations-app-create-button"
          >
            <Plus className="h-3.5 w-3.5" aria-hidden />
            New App
          </Button>
        </PermissionTooltip>
      </div>

      {isLoading ? (
        <p className="px-4 py-4 text-[13px] text-muted-foreground">Loading apps…</p>
      ) : apps.length === 0 ? (
        <p className="px-4 py-4 text-[13px] text-muted-foreground">No apps yet. Apps power line phases.</p>
      ) : (
        <ul>
          {apps.map((app) => (
            <li key={app.id} className="border-b border-border/60 last:border-b-0">
              <Link
                href={appPath(organizationId, app.id ?? "")}
                className="group flex items-center gap-3 px-4 py-3 text-[13px] text-foreground/80 hover:bg-accent hover:text-foreground"
                data-testid={`automations-app-${app.id}`}
              >
                <p className="min-w-0 flex-1 truncate font-medium text-foreground">{app.name}</p>
                <ExternalLink className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden />
                <ChevronRight
                  className="h-4 w-4 shrink-0 text-muted-foreground transition group-hover:text-foreground"
                  aria-hidden
                />
              </Link>
            </li>
          ))}
        </ul>
      )}

      <CreateFactoryAppDialog
        open={createOpen}
        isSaving={createCanvas.isPending}
        onClose={() => setCreateOpen(false)}
        onCreate={handleCreateApp}
      />
    </section>
  );
}
