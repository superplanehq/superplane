import type { FactoriesFactoryLine, FactoriesWorkOrder } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryApps, useFactoryWorkOrders, useUpdateFactoryLine } from "@/hooks/useFactoryData";
import { useWorkOrderChecks } from "@/hooks/useWorkOrderChecks";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Clock, Layers, MoreHorizontal, Pencil, Plus } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useLocation, useNavigate, useParams } from "react-router";
import { ClickToRename } from "../layout/ClickToRename";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import {
  buildLinePhaseBoard,
  collectLineBacklogOrders,
  findBacklogAutomationApp,
  findClosureAutomationApp,
  isDoneLineColumn,
  LINE_PHASE_RUNS_PAGE_SIZE,
  resolveColumnGlyph,
  resolvePhaseRunStatus,
  type LinePhaseColumn,
  type LinePhaseRunCard,
  type PhaseGlyphKind,
} from "../lib/linePhaseRuns";
import { isQueuedStepRow } from "../lib/workOrderExecutions";
import { latestDispatchForLine } from "../lib/workOrderNumberResolution";
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
  factoryHomePath,
  factoryLineDetailPath,
  isIntakeSearchOpen,
  linesPath,
  workOrderDetailPath,
} from "../lib/factoryPagePaths";
import { humanizeLineName } from "../lib/humanizeLineName";
import {
  factoryKanbanPageClassName,
  factorySectionBodyClassName,
  factorySectionHeaderClassName,
  factoryWorkOrdersBodyClassName,
} from "./factoryPageLayoutStyles";
import { replaceLineStepParallelism } from "../lib/factoryLineFormShared";
import { BacklogSettingsDialog } from "./BacklogSettingsDialog";
import { ColumnLaneMenu } from "./ColumnLaneMenu";
import { ParallelismSettingsDialog } from "./ParallelismSettingsDialog";
import { LineIntakeDrawer } from "./LineIntakeDrawer";
import { LineListCard } from "./LineListCard";
import { lineBoardColumnLaneClassName, type LineBoardColumnColorId } from "./lineBoardColumnColors";
import { descriptionForLine, toLineListMetrics } from "./lineListMetricsMockData";
import { useLineCardMutations } from "./useLineCardMutations";

const LIST_SUBTITLE = "Last 30 days. Success rate, completions per day, duration, and cost per merged work order.";

export function LinesPage() {
  const { organizationId, factoryId, factoryKey, factory, openCreateWorkOrder } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { lineId: routeLineId } = useParams<{ lineId: string }>();
  const { search } = useLocation();
  const navigate = useNavigate();
  const intakeOpen = isIntakeSearchOpen(search);
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const { data: factoryApps = [] } = useFactoryApps(organizationId, factoryId);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);

  const canUpdate = canAct("factories", "update");
  const canUpdateWorkOrders = canAct("work_orders", "update");
  const canCreateWorkOrder = canAct("work_orders", "create");
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
      <div className="flex h-full min-h-0 min-w-0 w-full" data-testid="lines-detail-page">
        {intakeOpen ? (
          <LineIntakeDrawer onClose={() => navigate(factoryHomePath(organizationId, factoryKey, selectedLine.id))} />
        ) : null}
        <div className={factoryKanbanPageClassName}>
          <div className="shrink-0">
            <LineDetailHeader
              organizationId={organizationId}
              factoryId={factoryId}
              factoryKey={factoryKey}
              line={selectedLine}
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
              canCreateWorkOrder={canCreateWorkOrder || permissionsLoading}
              canUpdate={canUpdate}
              onCreateWorkOrder={openCreateWorkOrder}
              workOrderCardContext={{
                organizationId,
                factoryKey,
                factoryLines: lines,
                canDispatch: canUpdateWorkOrders,
                preferredLineName: selectedLine.name,
                canAssign: canUpdateWorkOrders,
                ...cardActions,
              }}
            />
          </div>
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
  factoryId,
  factoryKey,
  line,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  canUpdate: boolean;
}) {
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const title = humanizeLineName(line.name);
  const editHref = line.id ? editFactoryLinePath(organizationId, factoryKey, line.id) : "#";

  const handleRename = async (name: string) => {
    if (!line.id) {
      return;
    }
    try {
      await updateLine.mutateAsync({ lineId: line.id, name });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to rename line"));
    }
  };

  return (
    <WorkspacePageHeader
      className={factorySectionHeaderClassName}
      title={
        <ClickToRename
          value={title}
          onSave={(name) => void handleRename(name)}
          canEdit={canUpdate && Boolean(line.id)}
          busy={updateLine.isPending}
          testId="lines-board-title"
          ariaLabel="Line name"
          inputClassName="font-medium text-[length:var(--workspace-page-title-size)] leading-[var(--workspace-page-title-line-height)] tracking-[var(--workspace-page-title-tracking)]"
        />
      }
      actions={
        canUpdate && line.id ? (
          <ColumnConfigureMenu title={title} href={editHref} testId="lines-edit-menu" />
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
  canCreateWorkOrder,
  canUpdate,
  onCreateWorkOrder,
  workOrderCardContext,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  workOrders: FactoriesWorkOrder[];
  canCreateWorkOrder: boolean;
  canUpdate: boolean;
  onCreateWorkOrder: () => void;
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
          factoryId={factoryId}
          factoryKey={factoryKey}
          line={line}
          backlogOrders={backlogOrders}
          columns={board}
          canCreateWorkOrder={canCreateWorkOrder}
          canRename={canUpdate}
          onCreateWorkOrder={onCreateWorkOrder}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={setPeekOrderId}
        />
      )}
      {peekOrderId ? (
        <LineBoardSplitRunPopup
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          lineId={line.id}
          lineName={line.name}
          peekOrderId={peekOrderId}
          peekOrder={peekOrder}
          canvasEditHref={canvasEditHref}
          canvasExpandHref={canvasExpandHref}
          canDispatch={workOrderCardContext.canDispatch}
          isDispatching={workOrderCardContext.isDispatching}
          onDispatch={workOrderCardContext.onDispatch}
          onClose={() => setPeekOrderId(null)}
        />
      ) : null}
    </div>
  );
}

function LineBoardSplitRunPopup({
  organizationId,
  factoryId,
  factoryKey,
  lineId,
  lineName,
  peekOrderId,
  peekOrder,
  canvasEditHref,
  canvasExpandHref,
  canDispatch,
  isDispatching,
  onDispatch,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId: string | undefined;
  lineName: string | undefined;
  peekOrderId: string;
  peekOrder: FactoriesWorkOrder | undefined;
  canvasEditHref: (key: SplitRunCanvasKey) => string | undefined;
  canvasExpandHref: (key: SplitRunCanvasKey) => string | undefined;
  canDispatch: boolean;
  isDispatching: boolean;
  onDispatch: (orderId: string, input: { lineName: string }) => Promise<void>;
  onClose: () => void;
}) {
  const { data: peekChecks = [] } = useWorkOrderChecks(organizationId, factoryId, peekOrderId);
  const resolvedLineName = lineName?.trim();
  const detailHref =
    peekOrder?.number !== undefined ? workOrderDetailPath(organizationId, factoryKey, peekOrder.number) : undefined;
  return (
    <WorkOrderSplitRunPopup
      key={peekOrderId}
      organizationId={organizationId}
      fixture={splitRunFixtureForWorkOrder(peekOrder, { checks: peekChecks, lineId, detailHref })}
      canvasEditHref={canvasEditHref}
      canvasExpandHref={canvasExpandHref}
      detailHref={detailHref}
      canDispatch={canDispatch && Boolean(resolvedLineName)}
      isDispatching={isDispatching}
      onDispatch={resolvedLineName ? () => onDispatch(peekOrderId, { lineName: resolvedLineName }) : undefined}
      onClose={onClose}
      fixed
    />
  );
}

function firstCanvasAppId(appIdByCanvas: Record<SplitRunCanvasKey, string | undefined>): string | undefined {
  return (
    appIdByCanvas.planning ??
    appIdByCanvas.implementation ??
    appIdByCanvas.risk ??
    appIdByCanvas.closure ??
    appIdByCanvas.intake
  );
}

function canvasAppIdsForLine(
  line: FactoriesFactoryLine,
  apps: Array<{ id?: string; name?: string }>,
): Record<SplitRunCanvasKey, string | undefined> {
  const backlog = findBacklogAutomationApp(apps);
  const closure = findClosureAutomationApp(apps);
  const steps = line.steps ?? [];
  return {
    intake: backlog?.id,
    planning: steps[0]?.app?.app,
    implementation: steps[1]?.app?.app,
    risk: steps[2]?.app?.app,
    closure: steps[3]?.app?.app ?? closure?.id,
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
    const appId = appIdByCanvas[key] ?? firstCanvasAppId(appIdByCanvas) ?? apps.find((app) => app.id)?.id;
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
      runId: executionRunIdForCanvas(order, key, line.id),
      orderNumber: order?.number,
      canvas: key,
    });
  };
}

function executionRunIdForCanvas(
  order: FactoriesWorkOrder | undefined,
  key: SplitRunCanvasKey,
  lineId: string | undefined,
): string | undefined {
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
  const executions = latestDispatchForLine(order, lineId)?.stepExecutions ?? [];
  return executions.find((execution) => execution.stepIndex === index)?.run?.id;
}

function PhaseBoard({
  organizationId,
  factoryId,
  factoryKey,
  line,
  backlogOrders,
  columns,
  canCreateWorkOrder,
  canRename,
  onCreateWorkOrder,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  backlogOrders: FactoriesWorkOrder[];
  columns: LinePhaseColumn[];
  canCreateWorkOrder: boolean;
  canRename: boolean;
  onCreateWorkOrder: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const [columnColors, setColumnColors] = useState<Record<string, LineBoardColumnColorId | null>>({});
  const [columnTitles, setColumnTitles] = useState<Record<string, string>>({});
  const [backlogSize, setBacklogSize] = useState<number | null>(null);
  const [backlogSettingsOpen, setBacklogSettingsOpen] = useState(false);
  const [parallelismByStep, setParallelismByStep] = useState<Record<number, number>>({});
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const lineId = line.id;

  const setColumnColor = useCallback((columnKey: string, colorId: LineBoardColumnColorId | null) => {
    setColumnColors((current) => ({ ...current, [columnKey]: colorId }));
  }, []);

  const setColumnTitle = useCallback((columnKey: string, title: string) => {
    setColumnTitles((current) => ({ ...current, [columnKey]: title }));
  }, []);

  const saveParallelism = useCallback(
    async (stepIndex: number, value: number) => {
      setParallelismByStep((current) => ({ ...current, [stepIndex]: value }));
      if (!line.id) {
        return;
      }
      try {
        await updateLine.mutateAsync({
          lineId: line.id,
          steps: replaceLineStepParallelism(line.steps, stepIndex, value),
        });
      } catch (error) {
        setParallelismByStep((current) => {
          const next = { ...current };
          delete next[stepIndex];
          return next;
        });
        showErrorToast(getApiErrorMessage(error, "Failed to update parallelism"));
      }
    },
    [line.id, line.steps, updateLine],
  );

  return (
    <WorkOrderKanbanBoard testId="lines-phase-board">
      <div className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}>
        <BacklogColumn
          orders={backlogOrders}
          title={columnTitles.backlog ?? "Backlog"}
          size={backlogSize}
          settingsOpen={backlogSettingsOpen}
          onOpenSettings={() => setBacklogSettingsOpen(true)}
          onCloseSettings={() => setBacklogSettingsOpen(false)}
          onSaveSettings={({ name, size }) => {
            setColumnTitle("backlog", name);
            setBacklogSize(size);
            setBacklogSettingsOpen(false);
          }}
          colorId={columnColors.backlog ?? null}
          onColorChange={(colorId) => setColumnColor("backlog", colorId)}
          canCreateWorkOrder={canCreateWorkOrder}
          canRename={canRename}
          onRename={(title) => setColumnTitle("backlog", title)}
          onCreateWorkOrder={onCreateWorkOrder}
          workOrderCardContext={workOrderCardContext}
          onOpenWorkOrder={onOpenWorkOrder}
        />
      </div>
      {columns.map((column, index) => {
        const columnKey = `phase-${column.stepIndex}`;
        return (
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
              title={columnTitles[columnKey] ?? column.stepName}
              parallelism={parallelismByStep[column.stepIndex] ?? column.maxParallelism}
              onSaveParallelism={(value) => void saveParallelism(column.stepIndex, value)}
              colorId={columnColors[columnKey] ?? null}
              onColorChange={(colorId) => setColumnColor(columnKey, colorId)}
              canRename={canRename}
              onRename={(title) => setColumnTitle(columnKey, title)}
              workOrderCardContext={workOrderCardContext}
              onOpenWorkOrder={onOpenWorkOrder}
            />
          </div>
        );
      })}
    </WorkOrderKanbanBoard>
  );
}

function BacklogColumn({
  orders,
  title,
  size,
  settingsOpen,
  onOpenSettings,
  onCloseSettings,
  onSaveSettings,
  colorId,
  onColorChange,
  canCreateWorkOrder,
  canRename,
  onRename,
  onCreateWorkOrder,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  orders: FactoriesWorkOrder[];
  title: string;
  size: number | null;
  settingsOpen: boolean;
  onOpenSettings: () => void;
  onCloseSettings: () => void;
  onSaveSettings: (settings: { name: string; size: number | null }) => void;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canCreateWorkOrder: boolean;
  canRename: boolean;
  onRename: (title: string) => void;
  onCreateWorkOrder: () => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);
  const atCapacity = size != null && orders.length >= size;
  const canAdd = canCreateWorkOrder && !atCapacity;

  return (
    <>
      <WorkOrderBoardLane
        title={title}
        label={title}
        canRename={canRename}
        onRename={onRename}
        titleTestId="lines-column-title-backlog"
        count={orders.length}
        tone="neutral"
        surfaceClassName={surfaceClassName}
        emptyDescription="No work orders in the backlog."
        className={surfaceClassName ? undefined : "bg-muted"}
        actions={
          <div className="flex shrink-0 items-center gap-0.5">
            <PermissionTooltip allowed={canCreateWorkOrder} message="You don't have permission to create work orders.">
              <button
                type="button"
                onClick={() => {
                  if (canAdd) {
                    onCreateWorkOrder();
                  }
                }}
                disabled={!canAdd}
                aria-label="Create work order"
                title={atCapacity ? "The backlog is full." : "Create work order"}
                data-testid="lines-backlog-create"
                className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:pointer-events-none disabled:opacity-60"
              >
                <Plus className="size-3.5" aria-hidden />
              </button>
            </PermissionTooltip>
            <ColumnLaneMenu
              title={title}
              testId="lines-backlog-menu"
              onEdit={onOpenSettings}
              colorId={colorId}
              onColorChange={onColorChange}
            />
          </div>
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
      <BacklogSettingsDialog
        open={settingsOpen}
        name={title}
        size={size}
        onSave={onSaveSettings}
        onClose={onCloseSettings}
      />
    </>
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
  title,
  parallelism,
  onSaveParallelism,
  colorId,
  onColorChange,
  canRename,
  onRename,
  workOrderCardContext,
  onOpenWorkOrder,
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  column: LinePhaseColumn;
  title: string;
  parallelism: number;
  onSaveParallelism: (value: number) => void;
  colorId: LineBoardColumnColorId | null;
  onColorChange: (colorId: LineBoardColumnColorId | null) => void;
  canRename: boolean;
  onRename: (title: string) => void;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string) => void;
}) {
  const scrollRef = useRef<HTMLUListElement>(null);
  const [visibleCount, setVisibleCount] = useState(LINE_PHASE_RUNS_PAGE_SIZE);
  const [parallelismOpen, setParallelismOpen] = useState(false);
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

  const visibleRuns = column.runs.slice(0, Math.min(visibleCount, totalRuns));
  const configureHref =
    !isDoneLineColumn(column) && column.appId
      ? factoryAppConfigurePath(organizationId, factoryKey, column.appId, { from: "lines", lineId })
      : null;
  const glyph = resolveColumnGlyph(column);
  const surfaceClassName = lineBoardColumnLaneClassName(colorId);

  return (
    <>
      <WorkOrderBoardLane
        title={title}
        label={`${title} phase`}
        count={totalRuns}
        tone={PHASE_LANE_TONE[glyph]}
        surfaceClassName={surfaceClassName}
        emptyDescription="No work orders in this phase."
        canRename={canRename}
        onRename={onRename}
        titleTestId={`lines-column-title-phase-${column.stepIndex}`}
        testId={`lines-phase-column-${column.stepIndex}`}
        actions={
          <ColumnLaneMenu
            title={title}
            testId={`lines-phase-menu-${column.stepIndex}`}
            editHref={configureHref}
            editLabel={configureHref ? "Edit Automation" : undefined}
            onSetParallelism={configureHref ? () => setParallelismOpen(true) : undefined}
            parallelism={parallelism}
            colorId={colorId}
            onColorChange={onColorChange}
          />
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
      <ParallelismSettingsDialog
        open={parallelismOpen}
        value={parallelism}
        onSave={(value) => {
          onSaveParallelism(value);
          setParallelismOpen(false);
        }}
        onClose={() => setParallelismOpen(false)}
      />
    </>
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
