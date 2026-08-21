import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps, useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Clock, Inbox, Layers, MoreHorizontal, Pencil, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import {
  buildLinePhaseBoard,
  collectLineBacklogOrders,
  findBacklogAutomationApp,
  LINE_PHASE_RUNS_PAGE_SIZE,
  resolveColumnGlyph,
  resolvePhaseRunStatus,
  type LinePhaseColumn,
  type LinePhaseRunCard,
  type PhaseGlyphKind,
} from "../lib/linePhaseRuns";
import { isQueuedStepRow } from "../lib/workOrderExecutions";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import {
  WorkOrderBoardLane,
  WorkOrderKanbanBoard,
  workOrderKanbanLaneScrollClassName,
  workOrderKanbanLaneSizeClassName,
  type BoardLaneTone,
} from "../workOrders/WorkOrderBoardChrome";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { WorkOrderSplitRunPopup } from "./work-order-split-run/WorkOrderSplitRunPopup";
import type { SplitRunCanvasKey } from "./work-order-split-run/splitRunCanvases";
import { splitRunFixtureForWorkOrder } from "./work-order-split-run/splitRunMocks";
import {
  createFactoryLinePath,
  editFactoryLinePath,
  factoryAppConfigurePath,
  factoryAppSplitRunPath,
  factoryLineDetailPath,
  linesPath,
} from "../lib/factoryPagePaths";
import { formatLinePhaseDescription, humanizeLineName } from "../lib/humanizeLineName";
import {
  factoryKanbanPageClassName,
  factorySectionBodyClassName,
  factorySectionHeaderClassName,
  factoryWorkOrdersBodyClassName,
} from "./factoryPageLayoutStyles";
import { LineListCard } from "./LineListCard";
import { descriptionForLine, toLineListMetrics } from "./lineListMetricsMockData";
import { PhaseGlyph } from "./linePhaseGlyph";
import { useLineCardMutations } from "./useLineCardMutations";

const LIST_SUBTITLE = "Last 30 days. Success rate, completions per day, duration, and cost per merged work order.";

export function LinesPage() {
  const { organizationId, factoryId, factoryKey, factory } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { lineId: routeLineId } = useParams<{ lineId: string }>();
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const { data: factoryApps = [] } = useFactoryApps(organizationId, factoryId);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);

  const canUpdate = canAct("factories", "update");
  const canUpdateWorkOrders = canAct("work_orders", "update");
  const lines = useMemo(() => factory?.lines ?? [], [factory?.lines]);
  const { actionsForLine } = useLineCardMutations({
    organizationId,
    factoryId,
    factoryKey,
    lines,
    canUpdate,
  });
  const selectedLine = useMemo(
    () => (routeLineId ? (lines.find((line) => line.id === routeLineId) ?? null) : null),
    [lines, routeLineId],
  );

  // Above the list/detail branching below (hooks can't be conditional): this
  // single call covers both the Lines list and the in-page detail view for a
  // selected line, re-firing when the selection changes without an
  // unmount/mount.
  usePageTitle([selectedLine ? humanizeLineName(selectedLine.name) : "Lines", factory?.name ?? "Workspace"]);

  if (routeLineId && factory && !selectedLine) {
    return <Navigate to={linesPath(organizationId, factoryKey)} replace />;
  }

  // The phase board is a Kanban surface: it claims the full viewport height so
  // the lanes read as columns rather than as boxes around their cards.
  if (selectedLine) {
    return (
      <div className={factoryKanbanPageClassName} data-testid="lines-detail-page">
        <div className="shrink-0">
          <LineDetailHeader
            organizationId={organizationId}
            factoryKey={factoryKey}
            line={selectedLine}
            apps={factoryApps}
            canUpdate={canUpdate}
          />
        </div>

        <div className={factoryWorkOrdersBodyClassName}>
          <LineDetail
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={factoryKey}
            line={selectedLine}
            apps={factoryApps}
            workOrders={workOrders}
            workOrderCardContext={{
              organizationId,
              factoryKey,
              factoryLines: lines,
              canDispatch: canUpdateWorkOrders,
              canAssign: canUpdateWorkOrders,
              ...cardActions,
            }}
          />
        </div>
      </div>
    );
  }

  return (
    <>
      <WorkspacePageHeader
        className={factorySectionHeaderClassName}
        title="Lines"
        subtitle={LIST_SUBTITLE}
        actions={
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to create lines."
          >
            <Button type="button" size="sm" asChild disabled={!canUpdate} data-testid="lines-create-button">
              <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryKey) : "#"}>
                <Plus className="size-3.5" aria-hidden />
                New line
              </Link>
            </Button>
          </PermissionTooltip>
        }
      />

      <div className={factorySectionBodyClassName}>
        {lines.length === 0 ? (
          <EmptyLinesState
            organizationId={organizationId}
            factoryKey={factoryKey}
            canUpdate={canUpdate || permissionsLoading}
          />
        ) : (
          <ul className="flex flex-col gap-4" data-testid="lines-list">
            {lines.map((line) => {
              if (!line.id) {
                return null;
              }
              return (
                <li key={line.id}>
                  <LineListCard
                    line={line}
                    href={factoryLineDetailPath(organizationId, factoryKey, line.id)}
                    metrics={toLineListMetrics(line.metrics)}
                    description={descriptionForLine(line.id)}
                    actions={actionsForLine(line)}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </>
  );
}

function LineDetailHeader({
  organizationId,
  factoryKey,
  line,
  apps,
  canUpdate,
}: {
  organizationId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  canUpdate: boolean;
}) {
  const editHref = line.id ? editFactoryLinePath(organizationId, factoryKey, line.id) : "#";
  return (
    <WorkspacePageHeader
      className={factorySectionHeaderClassName}
      title={humanizeLineName(line.name)}
      subtitle={formatLinePhaseDescription(line.steps, apps)}
      actions={
        canUpdate ? (
          <Button type="button" variant="outline" size="sm" asChild data-testid="lines-edit-button">
            <Link href={editHref}>
              <Pencil className="size-3.5" aria-hidden />
              Edit
            </Link>
          </Button>
        ) : undefined
      }
    />
  );
}

function LineDetail({
  organizationId,
  factoryId,
  factoryKey,
  line,
  apps,
  workOrders,
  workOrderCardContext,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  workOrders: FactoriesWorkOrder[];
  workOrderCardContext: WorkOrderCardContext;
}) {
  const steps = line.steps ?? [];
  const board = useMemo(() => buildLinePhaseBoard(line, workOrders ?? [], apps), [line, workOrders, apps]);
  const backlogOrders = useMemo(() => collectLineBacklogOrders(workOrders ?? []), [workOrders]);
  const [peekOrderId, setPeekOrderId] = useState<string | null>(null);
  const peekOrder = workOrders.find((order) => order.id === peekOrderId);
  const canvasEditHref = useMemo(
    () => canvasEditHrefForLine(organizationId, factoryKey, line, apps),
    [organizationId, factoryKey, line, apps],
  );
  const canvasExpandHref = useMemo(
    () => canvasExpandHrefForLine(organizationId, factoryKey, line, apps, peekOrder),
    [organizationId, factoryKey, line, apps, peekOrder],
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="lines-detail">
      {steps.length === 0 && backlogOrders.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">No phases yet. Edit this line to add app-driven phases.</p>
      ) : (
        <PhaseBoard
          organizationId={organizationId}
          factoryKey={factoryKey}
          lineId={line.id}
          apps={apps}
          backlogOrders={backlogOrders}
          columns={board}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={setPeekOrderId}
        />
      )}
      {peekOrderId ? (
        <WorkOrderSplitRunPopup
          key={peekOrderId}
          fixture={splitRunFixtureForWorkOrder(workOrders.find((order) => order.id === peekOrderId))}
          canvasEditHref={canvasEditHref}
          canvasExpandHref={canvasExpandHref}
          onClose={() => setPeekOrderId(null)}
          fixed
        />
      ) : null}
    </div>
  );
}

function canvasAppIdsForLine(
  line: FactoriesFactoryLine,
  apps: Array<{ id?: string; name?: string }>,
): Record<SplitRunCanvasKey, string | undefined> {
  const backlog = findBacklogAutomationApp(apps);
  const steps = line.steps ?? [];
  return {
    intake: backlog?.id,
    planning: steps[0]?.app?.app,
    implementation: steps[1]?.app?.app,
    risk: steps[2]?.app?.app,
    closure: steps[3]?.app?.app,
  };
}

function canvasEditHrefForLine(
  organizationId: string,
  factoryKey: string,
  line: FactoriesFactoryLine,
  apps: Array<{ id?: string; name?: string }>,
): (key: SplitRunCanvasKey) => string | undefined {
  const appIdByCanvas = canvasAppIdsForLine(line, apps);

  return (key) => {
    const appId = appIdByCanvas[key];
    if (!appId) {
      return undefined;
    }
    return factoryAppConfigurePath(organizationId, factoryKey, appId, { from: "lines", lineId: line.id });
  };
}

function canvasExpandHrefForLine(
  organizationId: string,
  factoryKey: string,
  line: FactoriesFactoryLine,
  apps: Array<{ id?: string; name?: string }>,
  order: FactoriesWorkOrder | undefined,
): (key: SplitRunCanvasKey) => string | undefined {
  const appIdByCanvas = canvasAppIdsForLine(line, apps);

  return (key) => {
    const appId = appIdByCanvas[key];
    if (!appId) {
      return undefined;
    }
    return factoryAppSplitRunPath(organizationId, factoryKey, appId, {
      from: "lines",
      lineId: line.id,
      runId: executionRunIdForCanvas(order, key),
      orderNumber: order?.number,
      canvas: key,
    });
  };
}

function executionRunIdForCanvas(order: FactoriesWorkOrder | undefined, key: SplitRunCanvasKey): string | undefined {
  const stepIndex: Partial<Record<SplitRunCanvasKey, number>> = {
    planning: 0,
    implementation: 1,
    risk: 2,
    closure: 3,
  };
  const index = stepIndex[key];
  if (index == null) {
    return undefined;
  }
  const executions = order?.lineDispatches?.[0]?.stepExecutions ?? [];
  return executions.find((execution) => execution.stepIndex === index)?.run?.id;
}

function PhaseBoard({
  organizationId,
  factoryKey,
  lineId,
  apps,
  backlogOrders,
  columns,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  apps: Array<{ id?: string; name?: string }>;
  backlogOrders: FactoriesWorkOrder[];
  columns: LinePhaseColumn[];
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const backlogApp = findBacklogAutomationApp(apps);
  const backlogConfigureHref = backlogApp
    ? factoryAppConfigurePath(organizationId, factoryKey, backlogApp.id, { from: "lines", lineId })
    : null;

  return (
    <WorkOrderKanbanBoard testId="lines-phase-board">
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <BacklogColumn
          orders={backlogOrders}
          configureHref={backlogConfigureHref}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
        />
      </div>
      {columns.map((column, index) => (
        <div
          key={`${column.stepIndex}-${column.stepName}`}
          className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}
        >
          {index < columns.length - 1 ? (
            <span className="absolute top-[21px] left-full z-[1] h-px w-3 bg-border" aria-hidden />
          ) : null}
          <PhaseColumn
            organizationId={organizationId}
            factoryKey={factoryKey}
            lineId={lineId}
            column={column}
            workOrderCardContext={workOrderCardContext}
            onOpenWorkOrder={onOpenWorkOrder}
          />
        </div>
      ))}
    </WorkOrderKanbanBoard>
  );
}

function BacklogColumn({
  orders,
  configureHref,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  orders: FactoriesWorkOrder[];
  configureHref: string | null;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  return (
    <WorkOrderBoardLane
      title="Backlog"
      label="Backlog"
      count={orders.length}
      tone="neutral"
      emptyDescription="No work orders in the backlog."
      className="bg-muted"
      leading={<Inbox className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
      actions={
        configureHref ? <ColumnConfigureMenu title="Backlog" href={configureHref} testId="lines-backlog-menu" /> : null
      }
      testId="lines-backlog-column"
    >
      <ul className={workOrderKanbanLaneScrollClassName} data-testid="lines-backlog-column-scroll">
        {orders.map((order) => (
          <li key={order.id}>
            <LineBoardOrderCard
              order={order}
              workOrderCardContext={workOrderCardContext}
              onOpenWorkOrder={onOpenWorkOrder}
            />
          </li>
        ))}
      </ul>
    </WorkOrderBoardLane>
  );
}

function ColumnConfigureMenu({ title, href, testId }: { title: string; href: string; testId: string }) {
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`${title} menu`}
          className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          data-testid={testId}
        >
          <MoreHorizontal className="size-3.5" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuItem onClick={() => navigate(href)} data-testid={`${testId}-edit`}>
          <Pencil className="h-3.5 w-3.5" aria-hidden />
          Edit
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Phase lanes borrow the Work Orders lane tints: blue in flight, grey once closed. */
const PHASE_LANE_TONE: Record<PhaseGlyphKind, BoardLaneTone> = {
  running: "running",
  waiting: "running",
  queued: "running",
  failed: "done",
  passed: "done",
  pending: "neutral",
};

function PhaseColumn({
  organizationId,
  factoryKey,
  lineId,
  column,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  column: LinePhaseColumn;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const scrollRef = useRef<HTMLUListElement>(null);
  const [visibleCount, setVisibleCount] = useState(LINE_PHASE_RUNS_PAGE_SIZE);
  const totalRuns = column.runs.length;
  const hasMore = visibleCount < totalRuns;

  const loadMore = useCallback(() => {
    setVisibleCount((current) => Math.min(current + LINE_PHASE_RUNS_PAGE_SIZE, totalRuns));
  }, [totalRuns]);

  const loadMoreIfNeeded = useAutoLoadMoreOnScroll({
    hasMore,
    onLoadMore: loadMore,
  });

  // When the window is short enough that the first page does not overflow,
  // pull the next page so a scrollbar can appear (same pattern as versions tab).
  useEffect(() => {
    loadMoreIfNeeded(scrollRef.current);
  }, [visibleCount, loadMoreIfNeeded]);

  const navigate = useNavigate();
  const visibleRuns = column.runs.slice(0, Math.min(visibleCount, totalRuns));
  const configureHref = column.appId
    ? factoryAppConfigurePath(organizationId, factoryKey, column.appId, { from: "lines", lineId })
    : null;
  const glyph = resolveColumnGlyph(column);

  return (
    <WorkOrderBoardLane
      title={column.stepName}
      label={`${column.stepName} phase`}
      count={totalRuns}
      tone={PHASE_LANE_TONE[glyph]}
      emptyDescription="No work orders in this phase."
      leading={<PhaseGlyph kind={glyph} />}
      testId={`lines-phase-column-${column.stepIndex}`}
      actions={
        configureHref ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                aria-label={`${column.stepName} menu`}
                className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                data-testid={`lines-phase-menu-${column.stepIndex}`}
              >
                <MoreHorizontal className="size-3.5" aria-hidden />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40">
              <DropdownMenuItem
                onClick={() => navigate(configureHref)}
                data-testid={`lines-phase-edit-${column.stepIndex}`}
              >
                <Pencil className="h-3.5 w-3.5" aria-hidden />
                Edit
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null
      }
    >
      <ul
        ref={scrollRef}
        className={workOrderKanbanLaneScrollClassName}
        onScroll={(event) => loadMoreIfNeeded(event.currentTarget)}
        data-testid={`lines-phase-column-scroll-${column.stepIndex}`}
      >
        {visibleRuns.map((run) => (
          <li key={run.executionId}>
            <PhaseRunCard run={run} workOrderCardContext={workOrderCardContext} onOpenWorkOrder={onOpenWorkOrder} />
          </li>
        ))}
      </ul>
    </WorkOrderBoardLane>
  );
}

function PhaseRunCard({
  run,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  run: LinePhaseRunCard;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const { factory } = useFactoriesLayout();
  const entry = useMemo(() => buildWorkOrderListEntry(run.order, factory), [run.order, factory]);
  const queuedLabel = isQueuedStepRow(run.execution) ? resolvePhaseRunStatus(run.execution).label : null;

  return (
    <div data-testid={`lines-phase-run-${run.executionId}`}>
      <WorkOrderCard
        {...workOrderCardContext}
        entry={entry}
        onOpen={() => {
          if (run.workOrderId) {
            onOpenWorkOrder(run.workOrderId);
          }
        }}
      />
      {queuedLabel ? (
        <p className="mt-1 flex items-center gap-1 px-0.5 text-[11px] text-muted-foreground">
          <Clock className="size-3 shrink-0" aria-hidden />
          {queuedLabel} — waiting for a free slot
        </p>
      ) : null}
    </div>
  );
}

function LineBoardOrderCard({
  order,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  order: FactoriesWorkOrder;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const { factory } = useFactoriesLayout();
  const entry = useMemo(() => buildWorkOrderListEntry(order, factory), [order, factory]);

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      onOpen={() => {
        if (order.id) {
          onOpenWorkOrder(order.id);
        }
      }}
    />
  );
}

function EmptyLinesState({
  organizationId,
  factoryKey,
  canUpdate,
}: {
  organizationId: string;
  factoryKey: string;
  canUpdate: boolean;
}) {
  return (
    <div
      className="flex flex-col items-center rounded-lg border border-dashed border-border bg-card px-6 py-12 text-center"
      data-testid="lines-empty-state"
    >
      <Layers className="h-8 w-8 text-muted-foreground" aria-hidden />
      <p className="mt-3 text-[15px] font-medium text-foreground">No lines yet</p>
      <p className="mt-1 max-w-md text-[13px] text-muted-foreground">
        Lines define how work orders flow through your apps.
      </p>
      <Button type="button" size="sm" asChild className="mt-6" disabled={!canUpdate}>
        <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryKey) : "#"}>
          <Plus className="size-3.5" aria-hidden />
          New line
        </Link>
      </Button>
    </div>
  );
}
