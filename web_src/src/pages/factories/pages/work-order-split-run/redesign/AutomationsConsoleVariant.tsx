import {
  Timeline,
  TimelineContent,
  TimelineHeader,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from "@/components/reui/timeline";
import { Badge } from "@/components/reui/badge";
import { cn } from "@/lib/utils";
import { Check, Circle, Clock, LoaderCircle, X } from "lucide-react";
import { type ReactNode, useMemo } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";

import { type SplitRunFixture, type SplitRunPhase } from "../splitRunMocks";
import type { SplitRunSource } from "../splitRunSource";
import { ConsoleSummaryPanel } from "./ConsoleSummaryPanel";
import { timingsForConsoleColumns, type ColumnTiming } from "./columnTiming";
import { ConsoleAutomationCard } from "./consoleAutomationCard";
import { ColumnTimingMeta } from "./consoleColumnTiming";
import { IntakeTimelineEvent } from "./consoleIntake";
import {
  allStages,
  automationsFromStages,
  isConsoleCreationStage,
  outcomeSummary,
  stagesByConsoleColumn,
  stagesFromFixture,
  type ConsoleAutomation,
  type ConsoleColumnId,
} from "./automationsViewModel";
import { META_TEXT_CLASSNAME } from "./redesignFormat";
import { fixtureWithStartedReruns, useStageAutomationRerun } from "./useStageAutomationRerun";

/**
 * Variant B: automation console. Backlog, Implement, Verify, and Done sit
 * on a timeline. Each card is the latest run of one automation, seen
 * through its agent. The header carries status and the summed duration. The
 * body is the same marker list on a finished run and on a running one. A
 * running step is the live row. The footer carries start time, live
 * spend, model, and Retry or Stop. Canvas nodes are not shown on this tab.
 * Columns the task has not reached read "Not started". A sticky Frame
 * holds the task status, spend, checks, and outputs.
 */
type AutomationsConsoleVariantProps = {
  fixture: SplitRunFixture;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  /** Task description markdown shown in the Intake event. */
  taskDescription?: string;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  source?: SplitRunSource;
  files?: FilesFile[];
  /** Pull requests tracked on this task, for the summary panel. */
  pullRequests?: FactoriesFactoryPullRequest[];
  /** Every artifact on this task, for the summary panel. */
  artifacts?: FactoriesWorkOrderArtifact[];
  /** Decision note and actions for the summary panel. */
  panelReview?: ReactNode;
  /** True when this person can cancel a live canvas run. */
  canStopRun?: boolean;
  actionBusy?: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  /** Reruns a failed line step through line dispatch. */
  onRerunStep?: (phase: SplitRunPhase) => void;
};

export function AutomationsConsoleVariant({
  fixture,
  organizationId,
  factoryId,
  orderId,
  factoryKey,
  orderNumber,
  taskDescription,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  source,
  files,
  pullRequests,
  artifacts,
  panelReview,
  canStopRun = false,
  actionBusy = false,
  onStopRun,
  onRerunStep,
}: AutomationsConsoleVariantProps) {
  const knownRunIds = useMemo(
    () => fixture.phases.flatMap((phase) => (phase.runId ? [phase.runId] : [])),
    [fixture.phases],
  );
  const stageRerun = useStageAutomationRerun(organizationId, factoryId, orderId, knownRunIds);
  const shownFixture = useMemo(
    () => fixtureWithStartedReruns(fixture, stageRerun.attempts),
    [fixture, stageRerun.attempts],
  );
  const outcome = outcomeSummary(shownFixture);
  const groups = stagesFromFixture(shownFixture);
  const stages = allStages(groups);
  const columns = consoleColumns(groups, shownFixture.footer.run?.appId);
  const currentColumn = reachedColumns(columns);
  const markers = columns.map((column, index) => columnMarker(column, index, currentColumn));
  const anyLive = hasLiveAutomation(columns);
  const expandIdleCards = !anyLive && (shownFixture.lineStatus === "pending" || shownFixture.footerTone === "draft");
  const liveRun = liveRunTarget(shownFixture);
  const stopLiveRun = canStopRun && onStopRun && liveRun ? () => onStopRun(liveRun) : undefined;
  const showIntake = Boolean(source) || Boolean(taskDescription?.trim()) || canEditDescription;
  const rerunAutomation = (phase: SplitRunPhase) => {
    if (!phase.appId || !phase.runId) {
      return;
    }
    void stageRerun.rerun({ sourcePhaseId: phase.id, appId: phase.appId, runId: phase.runId });
  };

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_340px]" data-testid="redesign-console-variant">
      <ConsoleTimeline
        fixture={shownFixture}
        columns={columns}
        markers={markers}
        currentColumn={currentColumn}
        expandIdleCards={expandIdleCards}
        showIntake={showIntake}
        source={source}
        taskDescription={taskDescription}
        canEditDescription={canEditDescription}
        descriptionBusy={descriptionBusy}
        onDescriptionSave={onDescriptionSave}
        files={files}
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
        canStopRun={canStopRun}
        actionBusy={actionBusy || stageRerun.pending}
        onStopRun={onStopRun}
        onRerunStep={onRerunStep}
        onRerunAutomation={rerunAutomation}
      />
      <ConsoleSummaryPanel
        fixture={fixture}
        outcome={outcome}
        stages={stages}
        pullRequests={pullRequests}
        artifacts={artifacts}
        panelReview={panelReview}
        source={source}
        actionBusy={actionBusy}
        onStopLiveRun={stopLiveRun}
      />
    </div>
  );
}

const CONSOLE_COLUMNS = [
  { id: "backlog", title: "Backlog" },
  { id: "implement", title: "Implement" },
  { id: "verify", title: "Verify" },
  { id: "done", title: "Done" },
] as const;

interface ConsoleColumn {
  id: ConsoleColumnId;
  title: string;
  automations: ConsoleAutomation[];
}

/**
 * Task stages sit in the column named after them. A column app uses the
 * column it is installed on. The create-task stage is omitted; the Intake
 * event shows it. Pull request activity sits in Verify, except factory
 * PR Closure and an app installed on another column.
 */
function consoleColumns(groups: ReturnType<typeof stagesFromFixture>, closerAppId?: string): ConsoleColumn[] {
  const byColumn = stagesByConsoleColumn(groups, closerAppId);
  return CONSOLE_COLUMNS.map((column) => ({
    id: column.id,
    title: column.title,
    automations: automationsFromStages(byColumn[column.id].filter((stage) => !isConsoleCreationStage(stage))),
  }));
}

function ConsoleTimeline({
  fixture,
  columns,
  markers,
  currentColumn,
  expandIdleCards,
  showIntake,
  source,
  taskDescription,
  canEditDescription,
  descriptionBusy,
  onDescriptionSave,
  files,
  organizationId,
  factoryId,
  orderId,
  factoryKey,
  orderNumber,
  canStopRun,
  actionBusy,
  onStopRun,
  onRerunStep,
  onRerunAutomation,
}: {
  fixture: SplitRunFixture;
  columns: ConsoleColumn[];
  markers: ColumnMarker[];
  currentColumn: number;
  expandIdleCards: boolean;
  showIntake: boolean;
  source?: SplitRunSource;
  taskDescription?: string;
  canEditDescription: boolean;
  descriptionBusy: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  factoryKey?: string;
  orderNumber?: string;
  canStopRun: boolean;
  actionBusy: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  onRerunStep?: (phase: SplitRunPhase) => void;
  onRerunAutomation?: (phase: SplitRunPhase) => void;
}) {
  const offset = showIntake ? 1 : 0;
  const timings = timingsForConsoleColumns(fixture);
  return (
    <section className="flex min-w-0 flex-col gap-4">
      <Timeline value={reachedColumnCount(markers) + offset} className="pl-1">
        {showIntake ? (
          <IntakeTimelineEvent
            source={source}
            description={taskDescription}
            canEdit={canEditDescription}
            busy={descriptionBusy}
            onSave={onDescriptionSave}
            files={files}
            organizationId={organizationId}
            factoryId={factoryId}
            orderId={orderId}
            timing={timings.intake}
          />
        ) : null}
        {columns.map((column, index) => (
          <ColumnTimelineItem
            key={column.id}
            column={column}
            index={index}
            step={index + 1 + offset}
            marker={markers[index]}
            currentColumn={currentColumn}
            timing={timings[column.id]}
            fixture={fixture}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
            expandIdle={expandIdleCards && index + 1 === currentColumn}
            canStopRun={canStopRun}
            actionBusy={actionBusy}
            onStopRun={onStopRun}
            onRerunStep={onRerunStep}
            onRerunAutomation={onRerunAutomation}
          />
        ))}
      </Timeline>
    </section>
  );
}

function ColumnTimelineItem({
  column,
  index,
  step,
  marker,
  currentColumn,
  timing,
  fixture,
  organizationId,
  factoryKey,
  orderNumber,
  expandIdle,
  canStopRun,
  actionBusy,
  onStopRun,
  onRerunStep,
  onRerunAutomation,
}: {
  column: ConsoleColumn;
  index: number;
  step: number;
  marker: ColumnMarker;
  currentColumn: number;
  timing?: ColumnTiming;
  fixture: SplitRunFixture;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  expandIdle: boolean;
  canStopRun: boolean;
  actionBusy: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  onRerunStep?: (phase: SplitRunPhase) => void;
  onRerunAutomation?: (phase: SplitRunPhase) => void;
}) {
  const emptyCopy = emptyColumnCopy(column.id, index, currentColumn, fixture);
  return (
    <TimelineItem step={step} className="group/column" data-testid={`redesign-console-column-${column.id}`}>
      <TimelineHeader className="flex w-full items-center gap-2">
        <TimelineSeparator />
        <ColumnStatusIndicator marker={marker} columnId={column.id} />
        <TimelineTitle className="font-semibold">{column.title}</TimelineTitle>
        {column.automations.length > 0 ? (
          <Badge variant="secondary" className="h-5 min-w-5 px-1.5">
            {column.automations.length}
          </Badge>
        ) : null}
        <ColumnTimingMeta columnId={column.id} timing={timing} />
      </TimelineHeader>
      <TimelineContent className="mt-2 flex flex-col gap-3 text-foreground">
        {column.automations.length === 0 ? (
          emptyCopy ? (
            <span className={META_TEXT_CLASSNAME}>{emptyCopy}</span>
          ) : null
        ) : (
          column.automations.map((automation) => (
            <ConsoleAutomationCard
              key={automation.id}
              automation={automation}
              phase={fixture.phases.find((entry) => entry.id === automation.latest.id)}
              phases={fixture.phases}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              expandIdle={expandIdle}
              canStopRun={canStopRun}
              actionBusy={actionBusy}
              onStopRun={onStopRun}
              onRerunStep={onRerunStep}
              onRerunAutomation={onRerunAutomation}
            />
          ))
        )}
      </TimelineContent>
    </TimelineItem>
  );
}

/** Empty Backlog after Intake is not skipped; the creation stage already ran. */
function emptyColumnCopy(
  columnId: ConsoleColumnId,
  index: number,
  currentColumn: number,
  fixture: SplitRunFixture,
): string | undefined {
  if (columnId === "backlog" && fixture.phases.some((phase) => isConsoleCreationStage(phase))) {
    return undefined;
  }
  return index + 1 < currentColumn ? "Skipped" : "Not started";
}

/** Timeline steps to fill: through the last column that has a run. */
function reachedColumns(columns: ConsoleColumn[]): number {
  const reached = columns.map((column) => column.automations.length > 0).lastIndexOf(true);
  return reached + 1;
}

/** The run the summary-panel Stop cancels: the running phase, else the footer's live run. */
function liveRunTarget(fixture: SplitRunFixture): { appId: string; runId: string } | undefined {
  const phase = fixture.phases.find((entry) => entry.status === "running" && entry.appId && entry.runId);
  if (phase?.appId && phase.runId) {
    return { appId: phase.appId, runId: phase.runId };
  }
  return fixture.footer.kind === "running" ? fixture.footer.run : undefined;
}

function hasLiveAutomation(columns: ConsoleColumn[]): boolean {
  return columns.some((column) =>
    column.automations.some((automation) => {
      const status = automation.latest.status;
      return status === "running" || status === "waiting";
    }),
  );
}

type ColumnMarker = "completed" | "running" | "waiting" | "failed" | "cancelled" | "pending";

/**
 * Rail marker for one console column. Composition matches ReUI
 * solution-agents-3: size-5 disk, size-3 icon, indicator in the header.
 */
function columnMarker(column: ConsoleColumn, index: number, currentColumn: number): ColumnMarker {
  if (column.automations.length === 0) {
    return index + 1 < currentColumn ? "completed" : "pending";
  }
  const statuses = column.automations.map((automation) => automation.latest.status);
  if (statuses.some((status) => status === "running")) return "running";
  if (statuses.some((status) => status === "waiting")) return "waiting";
  if (statuses.some((status) => status === "failed")) return "failed";
  if (statuses.some((status) => status === "cancelled")) return "cancelled";
  if (statuses.every((status) => status === "pending")) return "pending";
  return "completed";
}

/** Fill the rail through every started column, including failed and running. */
function reachedColumnCount(markers: ColumnMarker[]): number {
  const firstPending = markers.findIndex((marker) => marker === "pending");
  return firstPending === -1 ? markers.length : firstPending;
}

const COLUMN_MARKER_LABEL: Record<ColumnMarker, string> = {
  completed: "Completed",
  running: "Running",
  waiting: "Waiting",
  failed: "Failed",
  cancelled: "Canceled",
  pending: "Not started",
};

function ColumnStatusIndicator({ marker, columnId }: { marker: ColumnMarker; columnId: string }) {
  return (
    <TimelineIndicator
      aria-hidden={false}
      className={cn(
        "flex size-5 items-center justify-center border-none",
        marker === "pending"
          ? "bg-muted text-muted-foreground"
          : "bg-foreground text-background group-data-completed/timeline-item:bg-foreground group-data-completed/timeline-item:text-background",
      )}
      data-testid={`redesign-console-column-marker-${columnId}`}
      data-status={marker}
    >
      <span className="sr-only">{COLUMN_MARKER_LABEL[marker]}</span>
      {marker === "completed" ? <Check className="size-3" aria-hidden /> : null}
      {marker === "running" ? <LoaderCircle className="size-3 animate-spin" aria-hidden /> : null}
      {marker === "waiting" ? <Clock className="size-3" aria-hidden /> : null}
      {marker === "failed" || marker === "cancelled" ? <X className="size-3" aria-hidden /> : null}
      {marker === "pending" ? <Circle className="size-3" aria-hidden /> : null}
    </TimelineIndicator>
  );
}
