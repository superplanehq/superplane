import type { FactoryApp } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { useCreateCanvas, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { factoryAppsKey, useFactoryApps, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Plus, Workflow } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { CreateFactoryAppDialog } from "../CreateFactoryAppDialog";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  listFactoryAutomationRuns,
  resolveFactoryAutomationStatus,
  resolveFactoryAutomationStatusFromCanvasRuns,
  type FactoryAutomationRunCard,
  type FactoryAutomationTick,
} from "../lib/factoryAutomationStatus";
import {
  automationDetailPath,
  automationsPath,
  factoryAppConfigurePath,
  factoryAppRunPath,
  factoryLineDetailPath,
} from "../lib/factoryPagePaths";
import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";

export function AutomationsPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { appId: routeAppId } = useParams<{ appId: string }>();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: apps = [], isLoading: appsLoading } = useFactoryApps(organizationId, factoryId);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const canCreateApp = canAct("canvases", "create");

  const [createOpen, setCreateOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createCanvas = useCreateCanvas(organizationId);

  const selectedApp = useMemo(
    () => (routeAppId ? (apps.find((app) => app.id === routeAppId) ?? null) : null),
    [apps, routeAppId],
  );

  const legacyLine =
    routeAppId && factory?.lines ? (factory.lines.find((line) => line.id === routeAppId) ?? null) : null;

  const handleCreateAutomation = async (input: { name: string; description: string }) => {
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
        showSuccessToast("Automation created.");
        navigate(factoryAppConfigurePath(organizationId, factoryId, canvasId, { from: "automations" }));
      }
    } catch (error) {
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create automation"));
      throw error;
    }
  };

  if (routeAppId && !appsLoading && !selectedApp) {
    // Old /automations/:lineId bookmarks redirect to Lines once factory lines load.
    if (!factory) {
      return (
        <>
          <header className={factoryContentHeaderClassName}>
            <div>
              <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
                Automations
              </Heading>
            </div>
          </header>
          <div className={factoryContentBodyClassName}>
            <p className="text-[13px] text-muted-foreground">Loading…</p>
          </div>
        </>
      );
    }
    if (legacyLine?.id) {
      return <Navigate to={factoryLineDetailPath(organizationId, factoryId, legacyLine.id)} replace />;
    }
    return <Navigate to={automationsPath(organizationId, factoryId)} replace />;
  }

  return (
    <div
      className={cn(selectedApp && "flex h-full min-h-0 flex-col")}
      data-testid={selectedApp ? "automations-detail-page" : "automations-list-page"}
    >
      <header className={cn(factoryContentHeaderClassName, selectedApp && "shrink-0")}>
        <div>
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            Automations
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            Automations are one-step lines. Each one listens for a trigger and runs a canvas when it fires.
          </p>
        </div>
        {selectedApp == null ? (
          <PermissionTooltip
            allowed={canCreateApp || permissionsLoading}
            message="You don't have permission to create automations."
          >
            <Button
              type="button"
              variant="outline"
              asChild={false}
              disabled={!canCreateApp}
              onClick={() => setCreateOpen(true)}
              data-testid="automations-create-button"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden />
              Add automation
            </Button>
          </PermissionTooltip>
        ) : null}
      </header>

      <div
        className={cn(
          factoryContentBodyClassName,
          selectedApp && "flex min-h-0 flex-1 flex-col overflow-hidden py-6",
        )}
      >
        {appsLoading && !selectedApp ? (
          <p className="text-[13px] text-muted-foreground">Loading automations…</p>
        ) : selectedApp ? (
          <AutomationDetail organizationId={organizationId} factoryId={factoryId} app={selectedApp} />
        ) : apps.length === 0 ? (
          <EmptyAutomationsState canCreate={canCreateApp || permissionsLoading} onCreate={() => setCreateOpen(true)} />
        ) : (
          <ul className="flex flex-col gap-2" data-testid="automations-list">
            {apps.map((app) => {
              if (!app.id) {
                return null;
              }
              const status = resolveFactoryAutomationStatus(app.id, workOrders);
              return (
                <li key={app.id}>
                  <AutomationCard
                    app={app}
                    href={automationDetailPath(organizationId, factoryId, app.id)}
                    tick={status.tick}
                    statusLabel={status.label}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <CreateFactoryAppDialog
        open={createOpen}
        isSaving={createCanvas.isPending}
        onClose={() => setCreateOpen(false)}
        onCreate={handleCreateAutomation}
      />
    </div>
  );
}

function AutomationDetail({
  organizationId,
  factoryId,
  app,
}: {
  organizationId: string;
  factoryId: string;
  app: FactoryApp;
}) {
  const canvasId = app.id ?? "";
  const {
    data: runsPages,
    isLoading: runsLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteCanvasRuns(canvasId, {}, Boolean(canvasId));
  const canvasRuns = useMemo(
    () => runsPages?.pages.flatMap((page) => page?.runs ?? []) ?? [],
    [runsPages],
  );
  const status = resolveFactoryAutomationStatusFromCanvasRuns(canvasRuns);
  const runs = useMemo(() => listFactoryAutomationRuns(canvasRuns), [canvasRuns]);
  const configureHref = canvasId
    ? factoryAppConfigurePath(organizationId, factoryId, canvasId, { from: "automations" })
    : "#";

  const runsScrollRef = useRef<HTMLUListElement>(null);
  const loadMoreRuns = useCallback(() => {
    if (!hasNextPage || isFetchingNextPage) {
      return;
    }
    void fetchNextPage();
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);
  const loadMoreRunsIfNeeded = useAutoLoadMoreOnScroll({
    hasMore: Boolean(hasNextPage),
    isLoading: isFetchingNextPage,
    onLoadMore: loadMoreRuns,
  });

  useEffect(() => {
    loadMoreRunsIfNeeded(runsScrollRef.current);
  }, [runs.length, loadMoreRunsIfNeeded]);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="automations-detail">
      <Link
        href={automationsPath(organizationId, factoryId)}
        className="mb-3 inline-flex shrink-0 items-center gap-1.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground"
        data-testid="automations-detail-back"
      >
        <ArrowLeft className="size-3.5" strokeWidth={1.75} aria-hidden />
        All automations
      </Link>

      <div className="shrink-0">
        <AutomationCard app={app} tick={status.tick} statusLabel={status.label} emphasized />
      </div>

      <section
        className="mt-6 flex min-h-[12rem] min-w-0 max-h-[calc(100dvh-16rem)] flex-1 flex-col overflow-hidden rounded-lg border border-border bg-background"
        aria-label="Runs"
      >
        <div className="flex h-9 shrink-0 items-center justify-between gap-2 border-b border-border px-3">
          <h2 className="truncate text-[12px] font-medium tracking-[-0.01em] text-foreground">Runs</h2>
          <Link
            href={configureHref}
            className="cursor-pointer text-[13px] text-muted-foreground transition-colors hover:text-foreground"
            data-testid="automations-configure"
          >
            Configure
          </Link>
        </div>
        <ul
          ref={runsScrollRef}
          className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto overscroll-contain p-2 [scrollbar-width:thin]"
          onScroll={(event) => loadMoreRunsIfNeeded(event.currentTarget)}
          data-testid="automations-runs-scroll"
        >
          {runsLoading && runs.length === 0 ? (
            <li className="px-2 py-4 text-[12px] text-muted-foreground">Loading runs…</li>
          ) : runs.length === 0 ? (
            <li className="px-2 py-4 text-[12px] text-muted-foreground">No runs</li>
          ) : (
            <>
              {runs.map((run) => (
                <li key={run.runId}>
                  <AutomationRunCard
                    organizationId={organizationId}
                    factoryId={factoryId}
                    appId={canvasId}
                    run={run}
                  />
                </li>
              ))}
              {isFetchingNextPage ? (
                <li className="px-2 py-2 text-center text-[12px] text-muted-foreground">Loading more…</li>
              ) : null}
            </>
          )}
        </ul>
      </section>
    </div>
  );
}

function AutomationRunCard({
  organizationId,
  factoryId,
  appId,
  run,
}: {
  organizationId: string;
  factoryId: string;
  appId: string;
  run: FactoryAutomationRunCard;
}) {
  const href = factoryAppRunPath(organizationId, factoryId, appId, run.runId, { from: "automations" });
  const timestamp = run.updatedAt ?? run.finishedAt ?? run.createdAt;
  const timeLabel =
    run.tick === "queued" && !timestamp
      ? "next"
      : timestamp
        ? formatTimeAgo(new Date(timestamp), false)
        : null;

  return (
    <Link
      href={href}
      className="block w-full rounded-md border border-border bg-background px-3 py-2.5 text-left transition-colors hover:border-foreground/25 hover:bg-accent/40"
      data-testid={`automations-run-${run.runId}`}
    >
      <div className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{run.title}</div>
      <div className="mt-1.5 flex items-center gap-1.5 text-[12px] text-muted-foreground">
        <StatusTick tick={run.tick} size="sm" />
        <span>
          {run.label}
          {timeLabel ? ` · ${timeLabel}` : ""}
        </span>
      </div>
    </Link>
  );
}

function AutomationCard({
  app,
  href,
  tick,
  statusLabel,
  emphasized,
}: {
  app: FactoryApp;
  href?: string;
  tick: FactoryAutomationTick;
  statusLabel: string;
  emphasized?: boolean;
}) {
  const navigate = useNavigate();
  const description = app.description?.trim() || "Factory automation canvas.";
  const interactive = Boolean(href);

  return (
    <div
      role={interactive ? "button" : undefined}
      tabIndex={interactive ? 0 : undefined}
      className={cn(
        "group/automation w-full rounded-lg border px-3.5 py-3 text-left transition-colors",
        emphasized ? "border-foreground bg-background" : "border-border bg-background",
        interactive && "cursor-pointer hover:border-foreground/25 hover:bg-accent/40",
      )}
      onClick={interactive && href ? () => navigate(href) : undefined}
      onKeyDown={
        interactive && href
          ? (event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                navigate(href);
              }
            }
          : undefined
      }
      data-testid={emphasized ? `automations-detail-card-${app.id}` : `automations-app-${app.id}`}
    >
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Workflow className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
            <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">
              {app.name?.trim() || "Unnamed automation"}
            </span>
          </div>
          <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{description}</p>
        </div>
      </div>
      <div className="mt-3.5 flex items-center gap-1.5">
        <StatusTick tick={tick} />
        <span className="text-[12px] leading-tight text-muted-foreground">{statusLabel}</span>
      </div>
    </div>
  );
}

function StatusTick({ tick, size = "md" }: { tick: FactoryAutomationTick; size?: "sm" | "md" }) {
  return (
    <span
      className={cn(
        size === "sm" ? "size-1.5" : "size-2",
        "shrink-0 rounded-full bg-[#c4c4c4]",
        tick === "running" && "bg-[#3b82f6] animate-pulse",
        (tick === "waiting" || tick === "failed") && "bg-[#f59e0b]",
        tick === "queued" && "bg-[#a3a3a3]",
        tick === "passed" && "bg-[#22c55e]",
        tick === "cancelled" && "bg-[#a3a3a3]",
      )}
      aria-hidden
    />
  );
}

function EmptyAutomationsState({ canCreate, onCreate }: { canCreate: boolean; onCreate: () => void }) {
  return (
    <div
      className="flex flex-col items-center rounded-lg border border-dashed border-border bg-card px-6 py-12 text-center"
      data-testid="automations-empty-state"
    >
      <Workflow className="h-8 w-8 text-muted-foreground" aria-hidden />
      <p className="mt-3 text-[15px] font-medium text-foreground">No automations yet</p>
      <p className="mt-1 max-w-md text-[13px] text-muted-foreground">
        Automations listen for triggers and run canvases that power line phases.
      </p>
      <Button type="button" className="mt-6" disabled={!canCreate} onClick={onCreate}>
        <Plus className="h-3.5 w-3.5" aria-hidden />
        Add automation
      </Button>
    </div>
  );
}
