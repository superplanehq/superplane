import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

export const ACTIVE_LIVE_RUNTIME_EXECUTION_STATES = new Set(["STATE_STARTED", "STATE_PENDING"]);

export const LIVE_INTERACTIVE_SIDEBAR_COMPONENTS = new Set(["approval"]);

function executionTimestamp(execution: CanvasesCanvasNodeExecution): number {
  return Date.parse(execution.createdAt || execution.updatedAt || "");
}

export function newestItemByTimestamp<T>(items: T[], timestamp: (item: T) => number): T | null {
  if (items.length === 0) {
    return null;
  }

  let newest: T | null = null;
  let newestTimestamp = Number.NEGATIVE_INFINITY;
  let newestIndex = Infinity;

  for (let index = 0; index < items.length; index++) {
    const item = items[index];
    const candidateTimestamp = timestamp(item);
    const safeTimestamp = Number.isFinite(candidateTimestamp) ? candidateTimestamp : Number.NEGATIVE_INFINITY;
    if (
      newest === null ||
      safeTimestamp > newestTimestamp ||
      (safeTimestamp === newestTimestamp && index < newestIndex)
    ) {
      newest = item;
      newestTimestamp = safeTimestamp;
      newestIndex = index;
    }
  }

  return newest;
}

export function newestExecution(executions: CanvasesCanvasNodeExecution[]): CanvasesCanvasNodeExecution | null {
  return newestItemByTimestamp(executions, executionTimestamp);
}

export function orderExecutionsNewestFirst(executions: CanvasesCanvasNodeExecution[]): CanvasesCanvasNodeExecution[] {
  const newest = newestExecution(executions);
  if (!newest || executions.length <= 1) {
    return executions;
  }

  return [newest, ...executions.filter((execution) => execution.id !== newest.id)];
}

export function hasActiveLiveRuntimeExecutionOnLatest(executions: CanvasesCanvasNodeExecution[]): boolean {
  const latestExecution = newestExecution(executions);
  const state = latestExecution?.state;
  return Boolean(state && ACTIVE_LIVE_RUNTIME_EXECUTION_STATES.has(state));
}

export function isLiveNodeSetupState(workflowNode: ComponentsNode | undefined): boolean {
  if (!workflowNode) {
    return false;
  }

  if (workflowNode.errorMessage) {
    return true;
  }

  return !workflowNode.component && workflowNode.name === "New Component";
}
