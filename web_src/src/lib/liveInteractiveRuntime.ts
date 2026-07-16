import type { CanvasesCanvasNodeExecution, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

export const ACTIVE_LIVE_RUNTIME_EXECUTION_STATES = new Set(["STATE_STARTED", "STATE_PENDING"]);

export const LIVE_INTERACTIVE_SIDEBAR_COMPONENTS = new Set(["approval"]);

function executionTimestamp(execution: CanvasesCanvasNodeExecution): number {
  return Date.parse(execution.updatedAt || execution.createdAt || "");
}

export function newestExecution(executions: CanvasesCanvasNodeExecution[]): CanvasesCanvasNodeExecution | null {
  let newest: CanvasesCanvasNodeExecution | null = null;
  let newestTimestamp = Number.NEGATIVE_INFINITY;

  for (const execution of executions) {
    const candidateTimestamp = executionTimestamp(execution);
    const safeTimestamp = Number.isFinite(candidateTimestamp) ? candidateTimestamp : Number.NEGATIVE_INFINITY;
    if (safeTimestamp > newestTimestamp) {
      newest = execution;
      newestTimestamp = safeTimestamp;
    }
  }

  return newest;
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
