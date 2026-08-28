import { isQueuedStepRow, type WorkOrderStepRow } from "./workOrderExecutions";

export function resolvePhaseRunStatus(execution: WorkOrderStepRow): {
  kind: "running" | "waiting" | "queued" | "failed" | "idle";
  label: string;
} {
  if (isQueuedStepRow(execution)) {
    const position = execution.queuePosition ?? 0;
    return { kind: "queued", label: position > 0 ? `Queued #${position}` : "Queued" };
  }
  if (execution.state === "STATE_STARTED") {
    return { kind: "running", label: "Executing" };
  }
  if (execution.state === "STATE_CANCELLING") {
    // In-flight like Automations (running tick), but keep Cancelling label.
    return { kind: "running", label: "Cancelling" };
  }
  if (execution.state === "STATE_PENDING") {
    return { kind: "queued", label: "Queued" };
  }
  if (execution.state === "STATE_FINISHED") {
    if (execution.result === "RESULT_PASSED") {
      return { kind: "idle", label: "Passed" };
    }
    if (execution.result === "RESULT_FAILED") {
      return { kind: "failed", label: "Failed" };
    }
    if (execution.result === "RESULT_CANCELLED") {
      return { kind: "idle", label: "Cancelled" };
    }
  }
  return { kind: "idle", label: "Unknown" };
}
