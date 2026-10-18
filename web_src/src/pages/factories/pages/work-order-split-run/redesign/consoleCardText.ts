import { durationLabelMs, formatMinutesSecondsDuration, isSubSecondDurationLabel } from "@/lib/duration";

import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { SPLIT_RUN_CLOSURE_PHASE_ID, type SplitRunPhaseStatus } from "../splitRunMocks";
import type { AutomationStage } from "./automationsViewModel";

export type ConsoleDisplayStatus = SplitRunPhaseStatus | "queued";

export type ConsoleRunTiming = {
  queued?: boolean;
  workStartedAt?: string;
  workEndedAt?: string;
  completedWorkMs?: number;
  openStartedAt?: string[];
};

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
 * Duration on the collapsed row: working runs on the card, added together.
 * A queued agent does not add time. Spend and model sit on the open-card
 * footer, where they can update live.
 */
export function runMetaLine(
  stage: AutomationStage,
  runs: AutomationStage[] = [stage],
  timingByRunId?: ReadonlyMap<string, ConsoleRunTiming>,
  now = Date.now(),
): string {
  const counted = timingByRunId
    ? runs.filter((run) => !(run.status === "running" && timingByRunId.get(run.id)?.queued))
    : runs;
  if (counted.length === 0) {
    return "";
  }
  const labels = counted.map((run) => durationLabelForRun(run, timingByRunId?.get(run.id), now));
  const totalMs = labels.reduce((sum, label) => sum + durationLabelMs(label), 0);
  return (
    formatMinutesSecondsDuration(totalMs) ||
    (labels.some((label) => isSubSecondDurationLabel(label)) ? "<1s" : "") ||
    (counted.length <= 1 ? (labels[0] ?? "") : "")
  );
}

function durationLabelForRun(run: AutomationStage, timing: ConsoleRunTiming | undefined, now: number): string {
  const worked = workedMs(run, timing, now);
  if (worked === undefined) {
    return run.duration ?? "";
  }
  return formatMinutesSecondsDuration(worked) || "<1s";
}

function workedMs(run: AutomationStage, timing: ConsoleRunTiming | undefined, now: number): number | undefined {
  const completed = timing?.completedWorkMs ?? 0;
  const open = openWorkMs(timing, run.status, now);
  if (open !== undefined) {
    return open + completed;
  }
  const closed = closedWindowMs(timing);
  if (closed !== undefined) {
    return closed + completed;
  }
  return completed > 0 ? completed : undefined;
}

function openWorkMs(
  timing: ConsoleRunTiming | undefined,
  status: AutomationStage["status"],
  now: number,
): number | undefined {
  const starts = liveOpenStarts(timing);
  if (starts.length === 0 || status !== "running") {
    return undefined;
  }
  return starts.reduce((sum, value) => sum + elapsedMs(value, now), 0);
}

function closedWindowMs(timing: ConsoleRunTiming | undefined): number | undefined {
  const start = Date.parse(timing?.workStartedAt ?? "");
  const ended = Date.parse(timing?.workEndedAt ?? "");
  if (!Number.isFinite(start) || !Number.isFinite(ended)) {
    return undefined;
  }
  return Math.max(0, ended - start);
}

function elapsedMs(value: string, now: number): number {
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? Math.max(0, now - parsed) : 0;
}

function liveOpenStarts(timing: ConsoleRunTiming | undefined): string[] {
  if (timing?.workEndedAt) {
    return [];
  }
  if (timing?.openStartedAt && timing.openStartedAt.length > 0) {
    return timing.openStartedAt;
  }
  return timing?.workStartedAt ? [timing.workStartedAt] : [];
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
