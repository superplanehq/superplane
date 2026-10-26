import { durationLabelMs, formatMinutesSecondsDuration, isSubSecondDurationLabel } from "@/lib/duration";

import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { SPLIT_RUN_CLOSURE_PHASE_ID, type SplitRunPhaseStatus } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";

export type ConsoleDisplayStatus = SplitRunPhaseStatus | "queued";

export function consoleHeaderStatus(
  runs: Array<Pick<AutomationStage, "id" | "status">>,
  queuedRunIds: ReadonlySet<string>,
): ConsoleDisplayStatus {
  if (runs.some((run) => run.status === "running" && !queuedRunIds.has(run.id))) {
    return "running";
  }
  if (runs.some((run) => run.status === "waiting")) {
    return "waiting";
  }
  const live = runs.filter((run) => run.status === "running");
  if (live.length > 0 && live.every((run) => queuedRunIds.has(run.id))) {
    return "queued";
  }
  return runs.at(-1)?.status ?? "pending";
}

export function consoleStopRunId(
  runs: Array<Pick<AutomationStage, "id" | "status">>,
  queuedRunIds: ReadonlySet<string>,
): string | undefined {
  const working = runs.find((run) => run.status === "running" && !queuedRunIds.has(run.id));
  if (working) {
    return working.id;
  }
  return runs.find((run) => run.status === "running" && queuedRunIds.has(run.id))?.id;
}

export function consoleCanStop(
  status: ConsoleDisplayStatus,
  runs: Array<Pick<AutomationStage, "id" | "status">>,
  queuedRunIds: ReadonlySet<string>,
): boolean {
  if (status === "running" || status === "queued") {
    return true;
  }
  return status === "waiting" && Boolean(consoleStopRunId(runs, queuedRunIds));
}

/**
 * Sum of run durations, including setup and waits within each run.
 * Concurrent runs count separately. A currently queued run does not add time.
 */
export function runMetaLine(
  stage: AutomationStage,
  runs: AutomationStage[] = [stage],
  queuedRunIds: ReadonlySet<string> = new Set(),
): string {
  const counted = runs.filter((run) => !(run.status === "running" && queuedRunIds.has(run.id)));
  if (counted.length === 0) {
    return "";
  }
  const labels = counted.map((run) => run.duration ?? "");
  const totalMs = labels.reduce((sum, label) => sum + durationLabelMs(label), 0);
  return (
    formatMinutesSecondsDuration(totalMs) ||
    (labels.some((label) => isSubSecondDurationLabel(label)) ? "<1s" : "") ||
    (counted.length <= 1 ? (labels[0] ?? "") : "")
  );
}

export function outputCountLabel(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`;
}

/**
 * Footer of the open card. Start stamp stays here. Model and spend are
 * rendered beside this lead so spend can stay a live control.
 */
export function runFooterLine(stage: AutomationStage): string {
  const stamp = formatWorkOrderDateTime(new Date(stage.startedAt ?? ""));
  if (stage.id === SPLIT_RUN_CLOSURE_PHASE_ID) {
    return stamp ? `Closed ${stamp}` : "";
  }
  return stamp;
}

/** Dollar amount and compact token count for the run footer. */
export function runFooterSpendLabel(cost?: string, tokens?: string): string {
  return [cost, tokens].filter(Boolean).join(" · ");
}
