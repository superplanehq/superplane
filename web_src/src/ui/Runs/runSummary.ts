import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { formatCompactDuration } from "@/lib/duration";
import { getExecutionEventState } from "./runNodeDetailModel";

/**
 * A step's activity bucket, derived from its execution. Note that "error" is the
 * only alarm state (the execution could not complete); normal branching outcomes
 * such as a "false"/"rejected" result are considered "done", not errors.
 */
export type StepActivity = "error" | "running" | "waiting" | "done";

/** The subset of step activities users actively filter for. */
export type RunStepFilter = Exclude<StepActivity, "done">;

/** Panel-local step filters. Distinct from the runs-list status filters. */
export const RUN_STEP_FILTERS: { id: RunStepFilter; label: string; dotClassName: string }[] = [
  { id: "error", label: "Errors", dotClassName: "bg-red-500" },
  { id: "running", label: "Running", dotClassName: "bg-blue-500" },
  { id: "waiting", label: "Waiting", dotClassName: "bg-amber-500" },
];

/** An errored step is one whose execution itself failed to produce a result. */
export function isErrorExecution(execution: CanvasesCanvasNodeExecution): boolean {
  return execution.result === "RESULT_FAILED";
}

/** Event states that mean "blocked, waiting on time or input" (e.g. approval, timegate). */
const WAITING_EVENT_STATES = new Set(["waiting", "queued", "pending"]);

export function getStepActivity(
  node: ComponentsNode | undefined,
  execution: CanvasesCanvasNodeExecution,
): StepActivity {
  if (isErrorExecution(execution)) return "error";
  const eventState = getExecutionEventState(node, execution);
  if (WAITING_EVENT_STATES.has(eventState)) return "waiting";
  if (eventState === "running" || execution.state === "STATE_STARTED") return "running";
  return "done";
}

export interface RunStepSummary {
  /** Number of executed steps (excludes the trigger, which has no execution). */
  total: number;
  errors: number;
  running: number;
  waiting: number;
  /** Steps that have finished executing (includes errored steps). */
  done: number;
  /** Steps still in flight: running + waiting. */
  inFlight: number;
}

export function getRunStepSummary(
  executions: CanvasesCanvasNodeExecution[],
  workflowNodes: ComponentsNode[],
): RunStepSummary {
  const nodeById = new Map<string, ComponentsNode>();
  for (const node of workflowNodes) {
    if (node.id) nodeById.set(node.id, node);
  }

  const summary: RunStepSummary = {
    total: executions.length,
    errors: 0,
    running: 0,
    waiting: 0,
    done: 0,
    inFlight: 0,
  };

  for (const execution of executions) {
    const activity = getStepActivity(nodeById.get(execution.nodeId || ""), execution);
    if (activity === "error") {
      summary.errors += 1;
      summary.done += 1;
    } else if (activity === "running") {
      summary.running += 1;
      summary.inFlight += 1;
    } else if (activity === "waiting") {
      summary.waiting += 1;
      summary.inFlight += 1;
    } else {
      summary.done += 1;
    }
  }

  return summary;
}

/**
 * Wall-clock duration of a run. Finished runs report createdAt -> finishedAt;
 * still-running runs report how long they have been running so far.
 */
export function formatRunDuration(run: CanvasesCanvasRun, now: number = Date.now()): string | null {
  if (!run.createdAt) return null;
  const start = new Date(run.createdAt).getTime();
  const end = run.finishedAt ? new Date(run.finishedAt).getTime() : now;
  const elapsed = Math.max(0, end - start);
  return formatCompactDuration(elapsed);
}

/**
 * How long a single step took (createdAt -> updatedAt). Only reported for
 * finished executions; in-flight steps (running/waiting) have no final duration.
 */
export function formatStepDuration(execution: CanvasesCanvasNodeExecution): string | null {
  if (execution.state !== "STATE_FINISHED") return null;
  if (!execution.createdAt || !execution.updatedAt) return null;
  const elapsed = Math.max(0, new Date(execution.updatedAt).getTime() - new Date(execution.createdAt).getTime());
  return formatCompactDuration(elapsed);
}

/**
 * Timestamp shown on the trigger step: when the triggering event was received
 * (i.e. when the run started). Formatted as `HH:mm - dd:MM` (24h time, then
 * day:month), since triggers have no duration to report.
 */
export function formatEventTimestamp(timestamp?: string): string | null {
  if (!timestamp) return null;
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return null;
  const pad = (value: number) => String(value).padStart(2, "0");
  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  const dayMonth = `${date.getDate()}.${months[date.getMonth()]}`;
  return `${time} - ${dayMonth}`;
}

/** Elapsed offset from a reference start (e.g. `+0s`, `+3s`, `+7m 39s`), used for the step status sub-timeline rows. */
export function formatRelativeOffset(fromMs: number, toMs: number): string {
  const totalSeconds = Math.max(0, Math.round((toMs - fromMs) / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) return `+${seconds}s`;
  if (seconds === 0) return `+${minutes}m`;
  return `+${minutes}m ${seconds}s`;
}

/**
 * A step's elapsed time. Like {@link formatStepDuration} but also reports the
 * in-flight elapsed (createdAt -> now) for steps that are still running/waiting.
 */
export function formatStepElapsed(execution: CanvasesCanvasNodeExecution, now: number = Date.now()): string | null {
  if (!execution.createdAt) return null;
  const start = new Date(execution.createdAt).getTime();
  const end =
    execution.state === "STATE_FINISHED" && execution.updatedAt ? new Date(execution.updatedAt).getTime() : now;
  return formatCompactDuration(Math.max(0, end - start));
}

export function isRunFinished(run: CanvasesCanvasRun): boolean {
  return Boolean(run.finishedAt) || run.state === "STATE_FINISHED";
}

/** Every step whose execution errored (powers the error banner / marker). */
export function findErrorExecutions(executions: CanvasesCanvasNodeExecution[]): CanvasesCanvasNodeExecution[] {
  return executions.filter(isErrorExecution);
}

export function findFirstErrorExecution(executions: CanvasesCanvasNodeExecution[]): CanvasesCanvasNodeExecution | null {
  return executions.find(isErrorExecution) ?? null;
}
