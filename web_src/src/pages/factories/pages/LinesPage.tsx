import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoryLineStep } from "@/api-client";
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
import { Layers, MoreHorizontal, Pencil, Plus, Workflow } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import {
  buildLinePhaseBoard,
  LINE_PHASE_RUNS_PAGE_SIZE,
  linePhaseRunHref,
  resolveColumnGlyph,
  type LinePhaseColumn,
  type LinePhaseRunCard,
  type LinePhaseTick,
  type PhaseGlyphKind,
} from "../lib/linePhaseRuns";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import {
  WorkOrderBoardLane,
  WorkOrderKanbanBoard,
  workOrderKanbanLaneScrollClassName,
  workOrderKanbanLaneSizeClassName,
  type BoardLaneTone,
} from "../workOrders/WorkOrderBoardChrome";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import {
  createFactoryLinePath,
  editFactoryLinePath,
  factoryAppConfigurePath,
  factoryLineDetailPath,
  linesPath,
} from "../lib/factoryPagePaths";
import { automationNameForLineStep } from "../lib/factoryLineFormShared";
import { formatLinePhaseDescription, humanizeLineName } from "../lib/humanizeLineName";
import {
  factoryKanbanPageClassName,
  factorySectionBodyClassName,
  factorySectionHeaderClassName,
  factoryWorkOrdersBodyClassName,
} from "./factoryPageLayoutStyles";
import { PhaseGlyph } from "./linePhaseGlyph";

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
        subtitle="Factory lines specialize how work moves through the workspace. Each phase is backed by a canvas that runs work orders."
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
          <ul className="flex flex-col gap-2" data-testid="lines-list">
            {lines.map((line) => {
              if (!line.id) {
                return null;
              }
              const board = buildLinePhaseBoard(line, workOrders, factoryApps);
              return (
                <li key={line.id}>
                  <LineCard
                    line={line}
                    apps={factoryApps}
                    href={factoryLineDetailPath(organizationId, factoryKey, line.id)}
                    ticks={board.map((column) => column.tick)}
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
      variant="entity"
      backHref={linesPath(organizationId, factoryKey)}
      backLabel="Lines"
      backTestId="lines-back-to-list"
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
  factoryKey,
  line,
  apps,
  workOrders,
  workOrderCardContext,
}: {
  organizationId: string;
  factoryKey: string;
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  workOrders: FactoriesWorkOrder[];
  workOrderCardContext: WorkOrderCardContext;
}) {
  const steps = line.steps ?? [];
  const board = useMemo(() => buildLinePhaseBoard(line, workOrders ?? [], apps), [line, workOrders, apps]);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="lines-detail">
      {steps.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">No phases yet. Edit this line to add app-driven phases.</p>
      ) : (
        <PhaseBoard
          organizationId={organizationId}
          factoryKey={factoryKey}
          lineId={line.id}
          columns={board}
          workOrderCardContext={workOrderCardContext}
        />
      )}
    </div>
  );
}

function LineCard({
  line,
  apps,
  href,
  ticks,
}: {
  line: FactoriesFactoryLine;
  apps: Array<{ id?: string; name?: string }>;
  href: string;
  ticks: LinePhaseTick[];
}) {
  const navigate = useNavigate();
  const steps = line.steps ?? [];
  const description = formatLinePhaseDescription(steps, apps);

  return (
    <div
      role="button"
      tabIndex={0}
      className={cn(
        "group/line w-full cursor-pointer rounded-lg border border-border bg-background px-3.5 py-3 text-left transition-colors",
        "hover:border-foreground/25 hover:bg-accent/40",
      )}
      onClick={() => navigate(href)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          navigate(href);
        }
      }}
      data-testid={`lines-card-${line.id}`}
    >
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Workflow className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
            <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">
              {humanizeLineName(line.name)}
            </span>
          </div>
          {description ? <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{description}</p> : null}
        </div>
      </div>
      {steps.length > 0 ? <PhaseStrip steps={steps} apps={apps} ticks={ticks} /> : null}
    </div>
  );
}

function PhaseStrip({
  steps,
  apps,
  ticks,
}: {
  steps: FactoryLineStep[];
  apps: Array<{ id?: string; name?: string }>;
  ticks: LinePhaseTick[];
}) {
  return (
    <ol className="mt-3.5 flex w-full items-start" aria-label="Phases">
      {steps.map((step, index) => (
        <li
          key={`${step.app?.app ?? "step"}-${index}`}
          className="relative flex min-w-0 flex-1 flex-col items-center text-center"
        >
          {index < steps.length - 1 ? (
            <span
              className="absolute top-[7px] left-[calc(50%+8px)] right-[calc(-50%+8px)] h-px bg-border"
              aria-hidden
            />
          ) : null}
          <span className="relative z-[1] flex h-3.5 items-center justify-center">
            <PhaseTickDot tick={ticks[index] ?? null} />
          </span>
          <span className="mt-1.5 max-w-full truncate px-1 text-[12px] leading-tight text-muted-foreground">
            {automationNameForLineStep(step, apps, index)}
          </span>
        </li>
      ))}
    </ol>
  );
}

function PhaseBoard({
  organizationId,
  factoryKey,
  lineId,
  columns,
  workOrderCardContext,
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  columns: LinePhaseColumn[];
  workOrderCardContext: WorkOrderCardContext;
}) {
  return (
    <WorkOrderKanbanBoard testId="lines-phase-board">
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
          />
        </div>
      ))}
    </WorkOrderKanbanBoard>
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
}: {
  organizationId: string;
  factoryKey: string;
  lineId?: string;
  column: LinePhaseColumn;
  workOrderCardContext: WorkOrderCardContext;
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
            <PhaseRunCard
              run={run}
              lineId={lineId}
              stepAppId={column.appId}
              workOrderCardContext={workOrderCardContext}
            />
          </li>
        ))}
      </ul>
    </WorkOrderBoardLane>
  );
}

function PhaseRunCard({
  run,
  lineId,
  stepAppId,
  workOrderCardContext,
}: {
  run: LinePhaseRunCard;
  lineId?: string;
  stepAppId?: string;
  workOrderCardContext: WorkOrderCardContext;
}) {
  const { factory } = useFactoriesLayout();
  const entry = useMemo(() => buildWorkOrderListEntry(run.order, factory), [run.order, factory]);
  const href = linePhaseRunHref(
    workOrderCardContext.organizationId,
    workOrderCardContext.factoryKey,
    lineId,
    run,
    stepAppId,
  );
  return <WorkOrderCard {...workOrderCardContext} entry={entry} href={href} />;
}

function PhaseTickDot({ tick }: { tick: LinePhaseTick }) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full bg-[#c4c4c4]",
        tick === "running" && "bg-[#3b82f6] animate-pulse",
        tick === "waiting" && "bg-[#f59e0b]",
        tick === "failed" && "bg-[#ef4444]",
        tick === "queued" && "bg-[#a3a3a3]",
      )}
      aria-hidden
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
