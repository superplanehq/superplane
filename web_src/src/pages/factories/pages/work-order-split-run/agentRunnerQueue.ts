import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";
import { isAgentHarnessComponent } from "@/lib/agentRunnerSteps";

const ACTIVE_AGENT_STATES = new Set(["STATE_STARTED", "STATE_CANCELLING"]);
const WORKING_AGENT_STATES = new Set(["STATE_STARTED", "STATE_CANCELLING"]);
const WORK_START_STATES = new Set(["STATE_STARTED", "STATE_CANCELLING", "STATE_FINISHED"]);

export type AgentRunnerQueue = {
  queued: boolean;
  workStartedAt?: string;
  workEndedAt?: string;
  completedWorkMs?: number;
  openStartedAt?: string[];
};

export function agentRunnerQueue(
  canvas: CanvasesCanvas | undefined,
  run: CanvasesCanvasRun | undefined,
): AgentRunnerQueue {
  if (!run) {
    return { queued: false };
  }
  const agentIds = agentNodeIds(canvas);
  const timing = workTiming(run, agentIds);
  if (run.state !== "STATE_STARTED") {
    return { queued: false, ...timing };
  }
  if (
    agentIds.size === 0 ||
    !(hasAgentQueueItem(run, agentIds) || hasPendingAgentExecution(run, agentIds)) ||
    hasActiveAgentExecution(run, agentIds)
  ) {
    return { queued: false, ...timing };
  }
  return { queued: true, ...timing };
}

function agentNodeIds(canvas: CanvasesCanvas | undefined): Set<string> {
  const ids = new Set<string>();
  for (const node of canvas?.spec?.nodes ?? []) {
    if (!node.id || !isAgentHarnessComponent(node.component)) {
      continue;
    }
    if (node.type && node.type !== "TYPE_ACTION") {
      continue;
    }
    ids.add(node.id);
  }
  return ids;
}

function hasAgentQueueItem(run: CanvasesCanvasRun, agentIds: Set<string>): boolean {
  return (run.queueItems ?? []).some((item) => Boolean(item.nodeId && agentIds.has(item.nodeId)));
}

function hasPendingAgentExecution(run: CanvasesCanvasRun, agentIds: Set<string>): boolean {
  return (run.executions ?? []).some(
    (execution) => isAgentExecution(execution, agentIds) && execution.state === "STATE_PENDING",
  );
}

function hasActiveAgentExecution(run: CanvasesCanvasRun, agentIds: Set<string>): boolean {
  return (run.executions ?? []).some(
    (execution) =>
      Boolean(execution.nodeId && agentIds.has(execution.nodeId)) && ACTIVE_AGENT_STATES.has(execution.state ?? ""),
  );
}

function workTiming(
  run: CanvasesCanvasRun,
  agentIds: Set<string>,
): Pick<AgentRunnerQueue, "workStartedAt" | "workEndedAt" | "completedWorkMs" | "openStartedAt"> {
  const closed: ClosedAgentWork[] = [];
  const open: OpenAgentWork[] = [];
  for (const execution of run.executions ?? []) {
    const interval = agentWorkInterval(execution, agentIds);
    if (!interval) {
      continue;
    }
    if ("endedMs" in interval) {
      closed.push(interval);
      continue;
    }
    open.push(interval);
  }

  const completedWorkMs = closed.reduce((sum, interval) => sum + interval.durationMs, 0);
  if (open.length > 0) {
    const earliest = open.reduce((best, interval) => (interval.startedMs < best.startedMs ? interval : best));
    return {
      workStartedAt: earliest.startedAt,
      ...(completedWorkMs > 0 ? { completedWorkMs } : {}),
      ...(open.length > 1 ? { openStartedAt: open.map((interval) => interval.startedAt) } : {}),
    };
  }
  if (closed.length === 1) {
    return { workStartedAt: closed[0].startedAt, workEndedAt: closed[0].endedAt };
  }
  if (closed.length > 1) {
    const latest = closed.reduce((best, interval) => (interval.endedMs > best.endedMs ? interval : best));
    return { completedWorkMs, workEndedAt: latest.endedAt };
  }
  return unfinishedWorkTiming(run, agentIds);
}

type OpenAgentWork = {
  startedAt: string;
  startedMs: number;
};

type ClosedAgentWork = OpenAgentWork & {
  endedAt: string;
  endedMs: number;
  durationMs: number;
};

function agentWorkInterval(
  execution: NonNullable<CanvasesCanvasRun["executions"]>[number],
  agentIds: Set<string>,
): OpenAgentWork | ClosedAgentWork | undefined {
  if (!isAgentExecution(execution, agentIds) || !WORK_START_STATES.has(execution.state ?? "")) {
    return undefined;
  }
  const startedAt = execution.createdAt;
  const startedMs = Date.parse(startedAt ?? "");
  if (!startedAt || !Number.isFinite(startedMs)) {
    return undefined;
  }
  if (WORKING_AGENT_STATES.has(execution.state ?? "")) {
    return { startedAt, startedMs };
  }
  const endedAt = execution.updatedAt;
  const endedMs = Date.parse(endedAt ?? "");
  if (!endedAt || !Number.isFinite(endedMs) || endedMs < startedMs) {
    return undefined;
  }
  return { startedAt, startedMs, endedAt, endedMs, durationMs: endedMs - startedMs };
}

function finishedRunEnd(run: CanvasesCanvasRun): string | undefined {
  return run.state === "STATE_FINISHED" ? run.finishedAt || run.updatedAt : undefined;
}

function unfinishedWorkTiming(
  run: CanvasesCanvasRun,
  agentIds: Set<string>,
): Pick<AgentRunnerQueue, "workStartedAt" | "workEndedAt"> {
  const startedAt = agentExecutionTimestamp(run, agentIds, WORK_START_STATES, "createdAt");
  if (!startedAt) {
    return {};
  }
  if (hasWorkingAgentExecution(run, agentIds)) {
    return { workStartedAt: startedAt };
  }
  const endedAt =
    agentExecutionTimestamp(run, agentIds, new Set(["STATE_FINISHED"]), "updatedAt", "latest") ?? finishedRunEnd(run);
  return endedAt ? { workStartedAt: startedAt, workEndedAt: endedAt } : { workStartedAt: startedAt };
}

function hasWorkingAgentExecution(run: CanvasesCanvasRun, agentIds: Set<string>): boolean {
  return (run.executions ?? []).some(
    (execution) => isAgentExecution(execution, agentIds) && WORKING_AGENT_STATES.has(execution.state ?? ""),
  );
}

function agentExecutionTimestamp(
  run: CanvasesCanvasRun,
  agentIds: Set<string>,
  states: ReadonlySet<string>,
  field: "createdAt" | "updatedAt",
  which: "earliest" | "latest" = "earliest",
): string | undefined {
  let chosen: string | undefined;
  let chosenMs = which === "latest" ? Number.NEGATIVE_INFINITY : Number.POSITIVE_INFINITY;
  for (const execution of run.executions ?? []) {
    if (!isAgentExecution(execution, agentIds) || !states.has(execution.state ?? "")) {
      continue;
    }
    const value = execution[field];
    const parsed = Date.parse(value ?? "");
    const keep = which === "latest" ? parsed > chosenMs : parsed < chosenMs;
    if (!Number.isFinite(parsed) || !keep) {
      continue;
    }
    chosenMs = parsed;
    chosen = value;
  }
  return chosen;
}

function isAgentExecution(
  execution: NonNullable<CanvasesCanvasRun["executions"]>[number],
  agentIds: Set<string>,
): boolean {
  return Boolean(execution.nodeId && agentIds.has(execution.nodeId));
}
