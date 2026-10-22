import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import type { AutomationRun, RunStatus } from "@/ui/AutomationRunHistory/types";
import type { IntakeRunOutcome } from "@/ui/AutomationRunHistory/intakeRunOutcome";
import type { LineIntakeSourceId } from "../lineIntakeModel";
import { intakeRunSource, record, text } from "./intakeRunSource";

export type RunHistoryIntakeSource = LineIntakeSourceId | "scheduled-intake";

const summaries: Record<IntakeRunOutcome, string> = {
  created: "SuperPlane created a task from this source event.",
  filtered: "This source event did not match the intake filters. No task was created.",
  processing: "SuperPlane is processing this source event. The final outcome is not available yet.",
  failed: "This intake run failed. Open the failed activity for details.",
  cancelled: "This run was cancelled before intake finished. No final outcome was recorded.",
  unknown: "This run completed, but the recorded data does not explain its intake outcome.",
  "no-proposal": "The scheduled review finished without proposing a task.",
};

export function intakeRunPresentation(
  run: CanvasesCanvasRun,
  sourceId: RunHistoryIntakeSource,
  status: RunStatus,
  orders: FactoriesWorkOrder[],
): Pick<AutomationRun, "source" | "intake"> {
  const outcome = recordedIntakeOutcome(run, sourceId, status, orders);
  const payload = record(run.rootEvent?.data?.data ?? run.rootEvent?.data);
  return {
    source: intakeRunSource(sourceId, payload),
    intake: { outcome, summary: summaries[outcome], event: eventDescription(payload) },
  };
}

function recordedIntakeOutcome(
  run: CanvasesCanvasRun,
  sourceId: RunHistoryIntakeSource,
  status: RunStatus,
  orders: FactoriesWorkOrder[],
): IntakeRunOutcome {
  if (status === "Failed") return "failed";
  if (status === "Cancelled") return "cancelled";
  if (["Pending", "Running", "Cancelling"].includes(status)) return "processing";
  if (status !== "Passed") return "unknown";
  const executions = run.executions ?? [];
  // createWorkOrder emits only when it creates a task. Duplicate/update paths
  // pass without an output. A title-based task link alone proves neither case.
  if (
    executions.some((e) => e.componentName === "createWorkOrder" && e.outputChannels?.includes("default")) ||
    orders.some((order) => order.sourceRunId === run.id && Boolean(run.id))
  )
    return "created";
  const rejected = lastFilterAttempts(run).filter(
    (execution) => execution.outputChannels?.length === 1 && execution.outputChannels[0] === "false",
  );
  if (sourceId === "scheduled-intake")
    return rejected.some((execution) => execution.nodeId === "has-proposal") ? "no-proposal" : "unknown";
  return rejected.length > 0 ? "filtered" : "unknown";
}

function lastFilterAttempts(run: CanvasesCanvasRun) {
  const lastByNode = new Map<string, NonNullable<CanvasesCanvasRun["executions"]>[number]>();
  for (const execution of [...(run.executions ?? [])].sort((a, b) =>
    (a.createdAt ?? "").localeCompare(b.createdAt ?? ""),
  )) {
    if (execution.nodeId) lastByNode.set(execution.nodeId, execution);
  }
  // Use recorded branches and historical component identity, never current
  // filters or the absence of a linked task. Incomplete old records stay unknown.
  return [...lastByNode.values()].filter(
    (execution) => execution.componentName === "if" && execution.result === "RESULT_PASSED",
  );
}

function eventDescription(payload: Record<string, unknown>): string {
  const action = text(payload.action, payload.event, payload.alert_transition);
  return action ? `Source event · ${action}` : "Source event";
}
