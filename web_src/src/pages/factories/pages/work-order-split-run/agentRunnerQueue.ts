import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";
import { isAgentHarnessComponent } from "@/lib/agentRunnerSteps";

const ACTIVE_AGENT_STATES = new Set(["STATE_STARTED", "STATE_CANCELLING"]);

export type AgentRunnerQueue = {
  queued: boolean;
};

export function agentRunnerQueue(
  canvas: CanvasesCanvas | undefined,
  run: CanvasesCanvasRun | undefined,
): AgentRunnerQueue {
  if (!run) {
    return { queued: false };
  }
  const agentIds = agentNodeIds(canvas);
  if (run.state !== "STATE_STARTED") {
    return { queued: false };
  }
  if (
    agentIds.size === 0 ||
    !(hasAgentQueueItem(run, agentIds) || hasPendingAgentExecution(run, agentIds)) ||
    hasActiveAgentExecution(run, agentIds)
  ) {
    return { queued: false };
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

function isAgentExecution(
  execution: NonNullable<CanvasesCanvasRun["executions"]>[number],
  agentIds: Set<string>,
): boolean {
  return Boolean(execution.nodeId && agentIds.has(execution.nodeId));
}
