import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";
import { isAgentHarnessComponent } from "@/lib/agentRunnerSteps";

const ACTIVE_AGENT_STATES = new Set(["STATE_PENDING", "STATE_STARTED", "STATE_CANCELLING"]);
const WORKING_AGENT_STATES = new Set(["STATE_STARTED", "STATE_CANCELLING"]);

export type AgentRunnerQueue = {
  queued: boolean;
  workStartedAt?: string;
};

export function agentRunnerQueue(
  canvas: CanvasesCanvas | undefined,
  run: CanvasesCanvasRun | undefined,
): AgentRunnerQueue {
  if (!run || run.state !== "STATE_STARTED") {
    return { queued: false };
  }
  const agentIds = agentNodeIds(canvas);
  const startedAt = workStartedAt(run, agentIds);
  if (agentIds.size === 0 || !hasAgentQueueItem(run, agentIds) || hasActiveAgentExecution(run, agentIds)) {
    return startedAt ? { queued: false, workStartedAt: startedAt } : { queued: false };
  }
  return { queued: true };
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

function hasActiveAgentExecution(run: CanvasesCanvasRun, agentIds: Set<string>): boolean {
  return (run.executions ?? []).some(
    (execution) =>
      Boolean(execution.nodeId && agentIds.has(execution.nodeId)) && ACTIVE_AGENT_STATES.has(execution.state ?? ""),
  );
}

function workStartedAt(run: CanvasesCanvasRun, agentIds: Set<string>): string | undefined {
  let startedAt: string | undefined;
  let startedMs = Number.POSITIVE_INFINITY;
  for (const execution of run.executions ?? []) {
    if (!execution.nodeId || !agentIds.has(execution.nodeId)) {
      continue;
    }
    if (!WORKING_AGENT_STATES.has(execution.state ?? "")) {
      continue;
    }
    const createdMs = Date.parse(execution.createdAt ?? "");
    if (!Number.isFinite(createdMs) || createdMs >= startedMs) {
      continue;
    }
    startedMs = createdMs;
    startedAt = execution.createdAt;
  }
  return startedAt;
}
