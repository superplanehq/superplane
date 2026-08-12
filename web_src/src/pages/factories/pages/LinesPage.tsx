import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoryLineStep } from "@/api-client";
import { Link } from "@/components/Link/link";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Heading } from "@/components/Heading/heading";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { useAutoLoadMoreOnScroll } from "@/components/CanvasToolSidebar/useAutoLoadMoreOnScroll";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ArrowLeft, Layers, MoreHorizontal, Pencil, Plus, Workflow } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router";
import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  buildLinePhaseBoard,
  LINE_PHASE_RUNS_PAGE_SIZE,
  resolvePhaseRunStatus,
  type LinePhaseColumn,
  type LinePhaseRunCard,
  type LinePhaseTick,
} from "../lib/linePhaseRuns";
import {
  createFactoryLinePath,
  editFactoryLinePath,
  factoryAppConfigurePath,
  factoryAppRunPath,
  factoryLineDetailPath,
  linesPath,
  workOrderDetailPath,
} from "../lib/factoryPagePaths";
import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "./factoryPageLayoutStyles";

export function LinesPage() {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { lineId: routeLineId } = useParams<{ lineId: string }>();
  const { data: workOrders = [] } = useFactoryWorkOrders(organizationId, factoryId);

  const canUpdate = canAct("factories", "update");
  const lines = useMemo(() => factory?.lines ?? [], [factory?.lines]);
  const selectedLine = useMemo(
    () => (routeLineId ? (lines.find((line) => line.id === routeLineId) ?? null) : null),
    [lines, routeLineId],
  );

  if (routeLineId && factory && !selectedLine) {
    return <Navigate to={linesPath(organizationId, factoryId)} replace />;
  }

  return (
    <>
      <header className={factoryContentHeaderClassName}>
        <div>
          <Heading level={1} className={cn("!text-[22px]", factoryPageTitleClassName)}>
            Lines
          </Heading>
          <p className={cn("mt-1", factoryPageSubtitleClassName)}>
            Factory lines specialize how work moves through the workspace. Each phase is backed by a canvas that runs
            work orders.
          </p>
        </div>
        {selectedLine == null ? (
          <PermissionTooltip
            allowed={canUpdate || permissionsLoading}
            message="You don't have permission to create lines."
          >
            <Button type="button" variant="outline" asChild disabled={!canUpdate} data-testid="lines-create-button">
              <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryId) : "#"}>
                <Plus className="h-3.5 w-3.5" aria-hidden />
                Add line
              </Link>
            </Button>
          </PermissionTooltip>
        ) : null}
      </header>

      <div className={factoryContentBodyClassName}>
        {selectedLine ? (
          <LineDetail
            organizationId={organizationId}
            factoryId={factoryId}
            line={selectedLine}
            workOrders={workOrders}
            canUpdate={canUpdate}
          />
        ) : lines.length === 0 ? (
          <EmptyLinesState
            organizationId={organizationId}
            factoryId={factoryId}
            canUpdate={canUpdate || permissionsLoading}
          />
        ) : (
          <ul className="flex flex-col gap-2" data-testid="lines-list">
            {lines.map((line) => {
              if (!line.id) {
                return null;
              }
              const board = buildLinePhaseBoard(line, workOrders);
              return (
                <li key={line.id}>
                  <LineCard
                    line={line}
                    href={factoryLineDetailPath(organizationId, factoryId, line.id)}
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

function LineDetail({
  organizationId,
  factoryId,
  line,
  workOrders,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  workOrders: FactoriesWorkOrder[];
  canUpdate: boolean;
}) {
  const steps = line.steps ?? [];
  const editHref = line.id ? editFactoryLinePath(organizationId, factoryId, line.id) : "#";
  const board = useMemo(() => buildLinePhaseBoard(line, workOrders ?? []), [line, workOrders]);

  return (
    <div data-testid="lines-detail">
      <Link
        href={linesPath(organizationId, factoryId)}
        className="mb-3 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
        data-testid="lines-back-to-list"
      >
        <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
        All lines
      </Link>

      <section className={cn("w-full rounded-lg border border-foreground bg-background px-3.5 py-3")}>
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <Workflow className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
              <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">
                {line.name || "Unnamed line"}
              </h2>
            </div>
            <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{formatPhaseCount(steps.length)}</p>
          </div>
          {canUpdate ? (
            <Link
              href={editHref}
              className="inline-flex shrink-0 items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
              data-testid="lines-edit-button"
            >
              <Pencil className="h-3.5 w-3.5" aria-hidden />
              Edit
            </Link>
          ) : null}
        </div>
        {steps.length > 0 ? <PhaseStrip steps={steps} ticks={board.map((column) => column.tick)} /> : null}
      </section>

      {steps.length === 0 ? (
        <p className="mt-6 text-[13px] text-muted-foreground">
          No phases yet. Edit this line to add app-driven phases.
        </p>
      ) : (
        <PhaseBoard organizationId={organizationId} factoryId={factoryId} lineId={line.id} columns={board} />
      )}
    </div>
  );
}

function LineCard({ line, href, ticks }: { line: FactoriesFactoryLine; href: string; ticks: LinePhaseTick[] }) {
  const navigate = useNavigate();
  const steps = line.steps ?? [];

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
              {line.name || "Unnamed line"}
            </span>
          </div>
          <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{formatPhaseCount(steps.length)}</p>
        </div>
      </div>
      {steps.length > 0 ? <PhaseStrip steps={steps} ticks={ticks} /> : null}
    </div>
  );
}

function PhaseStrip({ steps, ticks }: { steps: FactoryLineStep[]; ticks: LinePhaseTick[] }) {
  return (
    <ol className="mt-3.5 flex w-full items-start" aria-label="Phases">
      {steps.map((step, index) => (
        <li key={`${step.name}-${index}`} className="relative flex min-w-0 flex-1 flex-col items-center text-center">
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
            {step.name || `Phase ${index + 1}`}
          </span>
        </li>
      ))}
    </ol>
  );
}

/** Max phase boxes per row; extra steps wrap to the next row. */
const MAX_PHASE_COLUMNS_PER_ROW = 3;

function PhaseBoard({
  organizationId,
  factoryId,
  lineId,
  columns,
}: {
  organizationId: string;
  factoryId: string;
  lineId?: string;
  columns: LinePhaseColumn[];
}) {
  const columnsPerRow = Math.min(columns.length, MAX_PHASE_COLUMNS_PER_ROW) || 1;

  return (
    <div
      className="mt-6 grid w-full gap-3"
      style={{ gridTemplateColumns: `repeat(${columnsPerRow}, minmax(0, 1fr))` }}
      data-testid="lines-phase-board"
    >
      {columns.map((column) => (
        <PhaseColumn
          key={`${column.stepIndex}-${column.stepName}`}
          organizationId={organizationId}
          factoryId={factoryId}
          lineId={lineId}
          column={column}
        />
      ))}
    </div>
  );
}

function PhaseColumn({
  organizationId,
  factoryId,
  lineId,
  column,
}: {
  organizationId: string;
  factoryId: string;
  lineId?: string;
  column: LinePhaseColumn;
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
    ? factoryAppConfigurePath(organizationId, factoryId, column.appId, { from: "lines", lineId })
    : null;

  return (
    <section
      className="flex min-w-0 flex-col rounded-lg border border-border bg-background"
      aria-label={`${column.stepName} stage`}
      data-testid={`lines-phase-column-${column.stepIndex}`}
    >
      <div className="flex h-9 items-center justify-between gap-2 border-b border-border px-3">
        <h3 className="min-w-0 truncate text-[12px] font-medium tracking-[-0.01em] text-foreground">
          {column.stepName}
        </h3>
        {configureHref ? (
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
        ) : null}
      </div>
      <ul
        ref={scrollRef}
        className="flex max-h-[14rem] min-h-[120px] flex-col gap-2 overflow-y-auto overscroll-contain p-2 [scrollbar-width:thin]"
        onScroll={(event) => loadMoreIfNeeded(event.currentTarget)}
        data-testid={`lines-phase-column-scroll-${column.stepIndex}`}
      >
        {totalRuns === 0 ? (
          <li className="px-2 py-4 text-[12px] text-muted-foreground">No work orders</li>
        ) : (
          visibleRuns.map((run) => (
            <li key={run.executionId}>
              <PhaseRunCard organizationId={organizationId} factoryId={factoryId} lineId={lineId} run={run} />
            </li>
          ))
        )}
      </ul>
    </section>
  );
}

function PhaseRunCard({
  organizationId,
  factoryId,
  lineId,
  run,
}: {
  organizationId: string;
  factoryId: string;
  lineId?: string;
  run: LinePhaseRunCard;
}) {
  const status = resolvePhaseRunStatus(run.execution);
  const appId = run.execution.run?.appId;
  const runId = run.execution.run?.id;
  const href =
    appId && runId
      ? factoryAppRunPath(organizationId, factoryId, appId, runId, { from: "lines", lineId })
      : workOrderDetailPath(organizationId, factoryId, run.workOrderId);
  const timestamp = run.execution.updatedAt ?? run.execution.createdAt;
  const timeLabel = timestamp ? formatTimeAgo(new Date(timestamp), false) : null;

  return (
    <Link
      href={href}
      className="block w-full rounded-md border border-border bg-background px-3 py-2.5 text-left transition-colors hover:border-foreground/25 hover:bg-accent/40"
      data-testid={`lines-phase-run-${run.executionId}`}
    >
      <div className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{run.title}</div>
      <div className="mt-1.5 flex items-center gap-1.5 text-[12px] text-muted-foreground">
        <PhaseTickDot tick={status.kind === "idle" ? null : status.kind} />
        <span>
          {status.label}
          {timeLabel ? ` · ${timeLabel}` : ""}
        </span>
      </div>
    </Link>
  );
}

function PhaseTickDot({ tick }: { tick: LinePhaseTick }) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full bg-[#c4c4c4]",
        tick === "running" && "bg-[#3b82f6] animate-pulse",
        tick === "waiting" && "bg-[#f59e0b]",
        tick === "queued" && "bg-[#a3a3a3]",
      )}
      aria-hidden
    />
  );
}

function EmptyLinesState({
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
      data-testid="lines-empty-state"
    >
      <Layers className="h-8 w-8 text-muted-foreground" aria-hidden />
      <p className="mt-3 text-[15px] font-medium text-foreground">No lines yet</p>
      <p className="mt-1 max-w-md text-[13px] text-muted-foreground">
        Lines define how work orders flow through your apps.
      </p>
      <Button type="button" asChild className="mt-6" disabled={!canUpdate}>
        <Link href={canUpdate ? createFactoryLinePath(organizationId, factoryId) : "#"}>
          <Plus className="h-3.5 w-3.5" aria-hidden />
          Add line
        </Link>
      </Button>
    </div>
  );
}

function formatPhaseCount(count: number) {
  return `${count} ${count === 1 ? "phase" : "phases"}`;
}
