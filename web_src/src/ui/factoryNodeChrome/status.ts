import { DEFAULT_EVENT_STATE_MAP } from "../componentBase/defaultEventStateMap";
import type { EventStateMap, EventStateStyle } from "../componentBase/eventState";
import type { FactoryNodeStatus } from "./types";

const PASSED_EVENT_STATES = new Set([
  "passed",
  "created",
  "success",
  "finished",
  // If / loop channel outcomes — successful finish, not "still pending".
  "true",
  "false",
  "done",
]);

const RUNNING_EVENT_STATES = new Set([
  "running",
  // Loop still iterating or waiting between iterations.
  "next",
  "waiting",
]);

const CANCELLED_EVENT_STATES = new Set(["cancelled", "rejected"]);

const UNSET_EVENT_STATES = new Set(["pending", "neutral"]);

const ICON_TO_FACTORY_STATUS: Record<string, FactoryNodeStatus> = {
  "circle-check": "passed",
  "circle-x": "failed",
  "triangle-alert": "error",
  "refresh-cw": "running",
  "loader-circle": "running",
  "circle-slash-2": "cancelled",
  "circle-slash": "cancelled",
  "circle-stop": "cancelled",
  "circle-dashed": "queued",
};

function isUnsetEventState(status: string | undefined): boolean {
  return status === undefined || UNSET_EVENT_STATES.has(status);
}

/** Map classic ComponentBase / mapper eventState → factory footer status. */
export function normalizeFactoryNodeStatus(status: string | undefined, stateMap?: EventStateMap): FactoryNodeStatus {
  if (status === undefined || UNSET_EVENT_STATES.has(status)) {
    return "pending";
  }
  if (PASSED_EVENT_STATES.has(status)) {
    return "passed";
  }
  if (RUNNING_EVENT_STATES.has(status)) {
    return "running";
  }
  if (CANCELLED_EVENT_STATES.has(status)) {
    return "cancelled";
  }
  if (status === "failed") {
    return "failed";
  }
  if (status === "error") {
    return "error";
  }
  if (status === "cancelling") {
    return "cancelling";
  }
  if (status === "queued") {
    return "queued";
  }
  if (status === "triggered") {
    return "triggered";
  }

  const mappedStatus = factoryStatusFromEventStyle(stateMap?.[status]);
  if (mappedStatus !== undefined) {
    return mappedStatus;
  }

  // No map entry: mapper success aliases (buildActionStateRegistry) are not in the whitelist.
  return "passed";
}

/**
 * Factory run inspection: keep Pending only while the run can still progress.
 * Finished run + no execution → Did not run. Channel/classic states map as usual.
 */
export function resolveFactoryRuntimeStatus({
  eventState,
  runIsActive = true,
  stateMap,
}: {
  eventState: string | undefined;
  runIsActive?: boolean;
  stateMap?: EventStateMap;
}): FactoryNodeStatus {
  if (isUnsetEventState(eventState)) {
    return runIsActive ? "pending" : "did_not_run";
  }
  return normalizeFactoryNodeStatus(eventState, stateMap);
}

function factoryStatusFromEventStyle(style: EventStateStyle | undefined): FactoryNodeStatus | undefined {
  if (style === undefined) {
    return undefined;
  }

  for (const [eventState, defaultStyle] of Object.entries(DEFAULT_EVENT_STATE_MAP)) {
    if (style.icon === defaultStyle.icon && style.badgeColor === defaultStyle.badgeColor) {
      return normalizeFactoryNodeStatus(eventState);
    }
  }

  for (const [eventState, defaultStyle] of Object.entries(DEFAULT_EVENT_STATE_MAP)) {
    if (style.badgeColor === defaultStyle.badgeColor) {
      return normalizeFactoryNodeStatus(eventState);
    }
  }

  return ICON_TO_FACTORY_STATUS[style.icon];
}

export function factoryNodeStatusLabel(status: FactoryNodeStatus): string {
  switch (status) {
    case "passed":
      return "Passed";
    case "failed":
      return "Failed";
    case "error":
      return "Error";
    case "running":
      return "Running";
    case "cancelling":
      return "Cancelling";
    case "queued":
      return "Queued";
    case "cancelled":
      return "Cancelled";
    case "triggered":
      return "Triggered";
    case "pending":
      return "Pending";
    case "did_not_run":
      return "Did not run";
  }
}

/** Footer strip — Storybook tones for primary states; classic hues for the rest. */
export function factoryNodeStatusStripClass(status: FactoryNodeStatus): string {
  switch (status) {
    case "passed":
      return "border-[#dcfce7] bg-[#f0fdf4] text-[#166534] dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300";
    case "failed":
    case "error":
      return "border-[#fecaca] bg-[#fef2f2] text-[#991b1b] dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300";
    case "running":
      return "border-[#bfdbfe] bg-[#eff6ff] text-[#1d4ed8] dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-300";
    case "cancelling":
      return "border-[#fde68a] bg-[#fffbeb] text-[#b45309] dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300";
    case "queued":
      return "border-[#fed7aa] bg-[#fff7ed] text-[#c2410c] dark:border-orange-500/20 dark:bg-orange-500/10 dark:text-orange-300";
    case "triggered":
      return "border-[#ddd6fe] bg-[#f5f3ff] text-[#6d28d9] dark:border-violet-500/20 dark:bg-violet-500/10 dark:text-violet-300";
    case "cancelled":
      return "border-[#e5e5e5] bg-[#f5f5f5] text-[#525252] dark:border-border dark:bg-muted dark:text-muted-foreground";
    case "pending":
    case "did_not_run":
      return "border-[#e5e5e5] bg-[#fafafa] text-[#737373] dark:border-border dark:bg-muted dark:text-muted-foreground";
  }
}

/** Compact duration: `7s`, `1m 16s`, `9m so far`. */
export function formatFactoryNodeDuration(ms: number, options?: { soFar?: boolean }): string {
  const totalSec = Math.max(0, Math.floor(ms / 1000));
  const hours = Math.floor(totalSec / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;

  let text: string;
  if (hours > 0) {
    text = minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  } else if (minutes > 0) {
    text = seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
  } else {
    text = `${seconds}s`;
  }

  return options?.soFar ? `${text} so far` : text;
}
