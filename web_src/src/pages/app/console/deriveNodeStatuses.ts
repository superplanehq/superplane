import type { CanvasesCanvasNodeExecution } from "@/api-client";

import type { ConsoleNodeStatus } from "./ConsoleContext";

/**
 * Build the latest-status map consumed by Dashboard status chips.
 *
 * For each node we look at the most recent execution and translate its
 * `state` / `result` pair into a small dashboard-friendly enum. Nodes with no
 * executions are intentionally omitted (the chip falls back to "unknown").
 */
export function deriveConsoleNodeStatuses(
  executionsByNodeId: Record<string, CanvasesCanvasNodeExecution[] | undefined>,
): Record<string, ConsoleNodeStatus> {
  const out: Record<string, ConsoleNodeStatus> = {};
  for (const [nodeId, executions] of Object.entries(executionsByNodeId)) {
    if (!executions || executions.length === 0) continue;
    // Take the chronologically latest execution. We pick by `createdAt` when
    // available, otherwise fall back to the array order returned by the API.
    const latest = [...executions].sort(byCreatedAtDesc)[0];
    out[nodeId] = statusFromExecution(latest);
  }
  return out;
}

function byCreatedAtDesc(a: CanvasesCanvasNodeExecution, b: CanvasesCanvasNodeExecution): number {
  const aTime = a.createdAt ? Date.parse(a.createdAt) : 0;
  const bTime = b.createdAt ? Date.parse(b.createdAt) : 0;
  return bTime - aTime;
}

export function statusFromExecution(execution: CanvasesCanvasNodeExecution): ConsoleNodeStatus {
  if (execution.state === "STATE_PENDING") return "pending";
  if (execution.state === "STATE_STARTED") return "running";
  if (execution.result === "RESULT_FAILED") return "failed";
  if (execution.result === "RESULT_CANCELLED") return "cancelled";
  if (execution.result === "RESULT_PASSED" || execution.state === "STATE_FINISHED") return "passed";
  return "unknown";
}
