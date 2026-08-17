import type { FactoriesFactory, FactoriesWorkOrder, FactoryApp } from "@/api-client";
import { Link } from "@/components/Link/link";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { useCallback, useEffect, useMemo, useRef, useState, type RefObject } from "react";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import {
  findWorkOrderForAutomationRun,
  listFactoryAutomationRuns,
  resolveFactoryAutomationStatusFromCanvasRuns,
  type FactoryAutomationRunCard,
} from "../lib/factoryAutomationStatus";
import { automationsPath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { type AutomationCardActions } from "./automationCardActions";
import { AutomationHeaderActions, DeleteAutomationDialog, StatusTick } from "./automationsPageParts";
import { factoryContentBodyClassName } from "./factoryPageLayoutStyles";
import { LineVelocityPanel } from "./LineVelocityPanel";

export function AutomationDetail({
  organizationId,
  factoryKey,
  app,
  actions,
  factory,
  workOrders,
  workOrderCardContext,
}: {
  organizationId: string;
  factoryKey: string;
  app: FactoryApp;
  actions: AutomationCardActions;
  factory: FactoriesFactory | null | undefined;
  workOrders: FactoriesWorkOrder[];
  workOrderCardContext: WorkOrderCardContext;
}) {
  const canvasId = app.id ?? "";
  const {
    data: runsPages,
    isLoading: runsLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteCanvasRuns(canvasId, {}, Boolean(canvasId));
  const canvasRuns = useMemo(() => runsPages?.pages.flatMap((page) => page?.runs ?? []) ?? [], [runsPages]);
  const status = resolveFactoryAutomationStatusFromCanvasRuns(canvasRuns);
  const runs = useMemo(() => listFactoryAutomationRuns(canvasRuns), [canvasRuns]);

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

  const [deleteOpen, setDeleteOpen] = useState(false);
  const automationName = app.name?.trim() || "Unnamed automation";
  const description = app.description?.trim();
  const subtitleParts = [status.label, description].filter(Boolean);
  const subtitle = subtitleParts.length > 0 ? subtitleParts.join(" · ") : undefined;

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="automations-detail">
      <WorkspacePageHeader
        variant="entity"
        backHref={automationsPath(organizationId, factoryKey)}
        backLabel="Automations"
        backTestId="automations-detail-back"
        title={automationName}
        subtitle={subtitle}
        actions={
          <AutomationHeaderActions
            name={automationName}
            actions={actions}
            onRequestDelete={() => setDeleteOpen(true)}
          />
        }
      />

      <DeleteAutomationDialog
        open={deleteOpen}
        name={automationName}
        canDelete={actions.canDelete}
        isDeleting={Boolean(actions.isDeleting)}
        onClose={() => setDeleteOpen(false)}
        onConfirm={actions.onDelete}
      />

      <div
        className={cn(factoryContentBodyClassName, "flex min-h-0 flex-1 flex-col overflow-hidden")}
        data-testid="automations-detail-body"
      >
        <AutomationDetailTabs
          organizationId={organizationId}
          factoryKey={factoryKey}
          canvasId={canvasId}
          factory={factory}
          workOrders={workOrders}
          workOrderCardContext={workOrderCardContext}
          runs={runs}
          runsLoading={runsLoading}
          isFetchingNextPage={isFetchingNextPage}
          runsScrollRef={runsScrollRef}
          loadMoreRunsIfNeeded={loadMoreRunsIfNeeded}
        />
      </div>
    </div>
  );
}

function AutomationDetailTabs({
  organizationId,
  factoryKey,
  canvasId,
  factory,
  workOrders,
  workOrderCardContext,
  runs,
  runsLoading,
  isFetchingNextPage,
  runsScrollRef,
  loadMoreRunsIfNeeded,
}: {
  organizationId: string;
  factoryKey: string;
  canvasId: string;
  factory: FactoriesFactory | null | undefined;
  workOrders: FactoriesWorkOrder[];
  workOrderCardContext: WorkOrderCardContext;
  runs: FactoryAutomationRunCard[];
  runsLoading: boolean;
  isFetchingNextPage: boolean;
  runsScrollRef: RefObject<HTMLUListElement | null>;
  loadMoreRunsIfNeeded: (element: HTMLElement | null) => void;
}) {
  return (
    <Tabs defaultValue="runs" className="flex min-h-0 min-w-0 flex-1 flex-col gap-3">
      <TabsList>
        <TabsTrigger value="runs" data-testid="automations-tab-runs">
          Runs
        </TabsTrigger>
        <TabsTrigger value="velocity" data-testid="automations-tab-velocity">
          Velocity
        </TabsTrigger>
      </TabsList>
      <TabsContent value="runs" className="mt-0 flex min-h-0 min-w-0 flex-1 flex-col outline-none">
        <ul
          ref={runsScrollRef}
          className="flex min-h-[12rem] min-w-0 max-h-[calc(100dvh-16rem)] flex-1 flex-col gap-2 overflow-y-auto overscroll-contain p-0.5 [scrollbar-width:thin]"
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
                <li key={run.runId} className="w-full min-w-0">
                  <AutomationRunCard
                    organizationId={organizationId}
                    factoryKey={factoryKey}
                    appId={canvasId}
                    factory={factory}
                    workOrders={workOrders}
                    workOrderCardContext={workOrderCardContext}
                    run={run}
                  />
                </li>
              ))}
              {isFetchingNextPage ? (
                <li className="py-2 text-center text-[12px] text-muted-foreground">Loading more…</li>
              ) : null}
            </>
          )}
        </ul>
      </TabsContent>
      <TabsContent
        value="velocity"
        className="mt-0 min-h-0 max-h-[calc(100dvh-16rem)] flex-1 overflow-y-auto outline-none [scrollbar-width:thin]"
      >
        <LineVelocityPanel intro="Run throughput for this automation." />
      </TabsContent>
    </Tabs>
  );
}

function AutomationRunCard({
  organizationId,
  factoryKey,
  appId,
  factory,
  workOrders,
  workOrderCardContext,
  run,
}: {
  organizationId: string;
  factoryKey: string;
  appId: string;
  factory: FactoriesFactory | null | undefined;
  workOrders: FactoriesWorkOrder[];
  workOrderCardContext: WorkOrderCardContext;
  run: FactoryAutomationRunCard;
}) {
  const href = factoryAppRunPath(organizationId, factoryKey, appId, run.runId, { from: "automations" });
  const order = findWorkOrderForAutomationRun(workOrders, run.runId);
  if (order) {
    const entry = buildWorkOrderListEntry(order, factory);
    return <WorkOrderCard {...workOrderCardContext} entry={entry} href={href} />;
  }

  return <AutomationRunRow href={href} run={run} />;
}

function AutomationRunRow({ href, run }: { href: string; run: FactoryAutomationRunCard }) {
  const timestamp = run.updatedAt ?? run.finishedAt ?? run.createdAt;
  const timeLabel =
    run.tick === "queued" && !timestamp ? "next" : timestamp ? formatTimeAgo(new Date(timestamp), false) : null;

  return (
    <Link
      href={href}
      className="block w-full rounded-md border border-border bg-background px-3 py-2.5 text-left transition-colors hover:border-foreground/25 hover:bg-accent/40"
      data-testid={`automations-run-${run.runId}`}
    >
      <div className="truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">{run.title}</div>
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
