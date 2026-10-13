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
import { formatCompactDuration } from "@/lib/duration";
import { Check, Circle, Clock, LoaderCircle, X } from "lucide-react";
import { useMemo } from "react";

import { formatWorkOrderDateTime } from "../lib/workOrderDateTime";
import { LiveHeaderSpendProvider } from "../pages/work-order-split-run/liveHeaderSpendContext";
import { phasesWithRunArtifacts } from "../pages/work-order-split-run/attachStreamArtifacts";
import { ConsoleAutomationCard } from "../pages/work-order-split-run/redesign/consoleAutomationCard";
import {
  timingsForConsoleColumns,
  type ColumnTiming,
  type ColumnTimingId,
} from "../pages/work-order-split-run/redesign/columnTiming";
import {
  automationsFromStages,
  isConsoleCreationStage,
  stagesByConsoleColumn,
  stagesFromFixture,
  type ConsoleAutomation,
  type ConsoleColumnId,
} from "../pages/work-order-split-run/redesign/automationsViewModel";
import { META_TEXT_CLASSNAME } from "../pages/work-order-split-run/redesign/redesignFormat";
import { useSplitRunStreamArtifacts } from "../pages/work-order-split-run/useSplitRunStreamArtifacts";
import { SpecificModelIdsProvider } from "../pages/work-order-split-run/specificModelIds";
import type { SplitRunFixture, SplitRunPhase, SplitRunPhaseStatus } from "../pages/work-order-split-run/splitRunMocks";
import { MOBILE_TASK_COPY } from "./mobileCopy";

const CONSOLE_COLUMNS = [
  { id: "backlog", title: "Backlog" },
  { id: "implement", title: "Implement" },
  { id: "verify", title: "Verify" },
  { id: "done", title: "Done" },
] as const;

type MobileConsoleColumn = {
  id: ConsoleColumnId;
  title: string;
  automations: ConsoleAutomation[];
};

type MobileTaskActivityProps = {
  organizationId: string;
  factoryId: string;
  orderId: string;
  fixture: SplitRunFixture;
  factoryKey?: string;
  orderNumber?: string;
  canStopRun?: boolean;
  actionBusy?: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  onRerunStep?: (phase: SplitRunPhase) => void;
};

/**
 * Automation runs on the task, with the same cards as the desktop task
 * popup. The timeline stacks in one column for a narrow screen: only
 * columns that ran show up, and each card opens the same collapsible
 * agent steps (Clone Repo, Implementation, Commit and Push, Generate PR
 * title and description) with the same run footer and Stop control.
 */
export function MobileTaskActivity({
  organizationId,
  factoryId,
  orderId,
  fixture,
  factoryKey,
  orderNumber,
  canStopRun = false,
  actionBusy = false,
  onStopRun,
  onRerunStep,
}: MobileTaskActivityProps) {
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);
  const consoleFixture = useMemo(() => {
    const phases = phasesWithRunArtifacts(fixture.phases, artifactIndex);
    return phases === fixture.phases ? fixture : { ...fixture, phases };
  }, [artifactIndex, fixture]);
  const columns = useMemo(() => mobileConsoleColumns(consoleFixture), [consoleFixture]);
  const timings = useMemo(() => timingsForConsoleColumns(consoleFixture), [consoleFixture]);
  const currentColumn = reachedColumns(columns);
  const markers = columns.map((column, index) => columnMarker(column, index, currentColumn));
  const anyLive = hasLiveAutomation(columns);
  const expandIdleCards = !anyLive && (fixture.lineStatus === "pending" || fixture.footerTone === "draft");

  if (columns.length === 0) {
    return <p className="text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.noActivity}</p>;
  }
  return (
    <SpecificModelIdsProvider organizationId={organizationId}>
      <LiveHeaderSpendProvider>
        <section aria-label={MOBILE_TASK_COPY.activity} data-testid="mobile-task-activity">
          <Timeline value={columns.length} className="pl-1">
            {columns.map((column, index) => (
              <TimelineItem
                key={column.id}
                step={index + 1}
                className="group/column"
                data-testid={`mobile-task-column-${column.id}`}
              >
                <TimelineHeader className="flex w-full items-center gap-2">
                  <TimelineSeparator />
                  <ColumnStatusIndicator marker={markers[index] ?? "pending"} columnId={column.id} />
                  <TimelineTitle className="font-semibold">{column.title}</TimelineTitle>
                  {column.automations.length > 0 ? (
                    <Badge variant="secondary" className="h-5 min-w-5 px-1.5">
                      {column.automations.length}
                    </Badge>
                  ) : null}
                  <MobileColumnTiming columnId={column.id} timing={timings[column.id] as ColumnTiming | undefined} />
                </TimelineHeader>
                <TimelineContent className="mt-2 flex min-w-0 flex-col gap-3 text-foreground">
                  {column.automations.map((automation) => (
                    <ConsoleAutomationCard
                      key={automation.id}
                      automation={automation}
                      phase={consoleFixture.phases.find((entry) => entry.id === automation.latest.id)}
                      phases={consoleFixture.phases}
                      organizationId={organizationId}
                      factoryKey={factoryKey}
                      orderNumber={orderNumber}
                      expandIdle={expandIdleCards && index + 1 === currentColumn}
                      canStopRun={canStopRun}
                      actionBusy={actionBusy}
                      onStopRun={onStopRun}
                      onRerunStep={onRerunStep}
                    />
                  ))}
                </TimelineContent>
              </TimelineItem>
            ))}
          </Timeline>
        </section>
      </LiveHeaderSpendProvider>
    </SpecificModelIdsProvider>
  );
}

/**
 * Task stages sit in the column named after them, like the desktop
 * console. The phone skips empty columns and the synthetic create-task
 * stage so the run log starts at the first automation that acted.
 */
function mobileConsoleColumns(fixture: SplitRunFixture): MobileConsoleColumn[] {
  const groups = stagesFromFixture(fixture);
  const byColumn = stagesByConsoleColumn(groups, fixture.footer.run?.appId);
  return CONSOLE_COLUMNS.flatMap((column) => {
    const automations = automationsFromStages(byColumn[column.id].filter((stage) => !isConsoleCreationStage(stage)));
    if (automations.length === 0) {
      return [];
    }
    return [{ id: column.id, title: column.title, automations }];
  });
}

/** Timeline steps to fill: through the last column that has a run. */
function reachedColumns(columns: MobileConsoleColumn[]): number {
  return columns.length;
}

function hasLiveAutomation(columns: MobileConsoleColumn[]): boolean {
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
function columnMarker(column: MobileConsoleColumn, index: number, currentColumn: number): ColumnMarker {
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
      data-testid={`mobile-task-column-marker-${columnId}`}
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

/**
 * Arrival and dwell for one column. Desktop shows this on hover; a phone
 * has no hover, so it stays visible in a compact badge row that wraps
 * under the column title on narrow screens.
 */
function MobileColumnTiming({ columnId, timing }: { columnId: ColumnTimingId; timing?: ColumnTiming }) {
  if (!timing) {
    return null;
  }
  const arrived = formatWorkOrderDateTime(new Date(timing.enteredAt));
  const spent = timing.durationMs != null ? formatCompactDuration(timing.durationMs) : "";
  if (!arrived && !spent) {
    return null;
  }
  return (
    <span
      className="ml-auto inline-flex max-w-full flex-wrap items-center justify-end gap-1"
      data-testid={`mobile-task-column-timing-${columnId}`}
    >
      {arrived ? (
        <Badge variant="secondary" size="sm" radius="full" className="font-normal tabular-nums">
          <span className="sr-only">Arrived </span>
          {arrived}
        </Badge>
      ) : null}
      {spent ? (
        <Badge variant="secondary" size="sm" radius="full" className="font-normal tabular-nums">
          <span className="sr-only">Spent </span>
          {spent}
        </Badge>
      ) : null}
    </span>
  );
}

export function MobileActivityEmptyHint() {
  return <p className={META_TEXT_CLASSNAME}>{MOBILE_TASK_COPY.noActivity}</p>;
}

export type { SplitRunPhaseStatus };
