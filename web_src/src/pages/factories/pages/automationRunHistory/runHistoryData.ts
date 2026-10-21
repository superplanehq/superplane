import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  FactoriesWorkOrder,
  SuperplaneComponentsNode,
} from "@/api-client";
import { formatCompactDuration } from "@/lib/duration";
import type { AutomationRun, RunActivity, RunStatus } from "@/ui/AutomationRunHistory/types";
import { findWorkOrderForAutomationRun } from "../../lib/factoryAutomationStatus";

export function recordedStatus(record: Pick<CanvasesCanvasRun, "state" | "result">): RunStatus {
  if (record.state === "STATE_CANCELLING") return "Cancelling";
  if (record.state === "STATE_PENDING") return "Pending";
  if (record.state === "STATE_STARTED") return "Running";
  if (record.result === "RESULT_FAILED") return "Failed";
  if (record.result === "RESULT_CANCELLED") return "Cancelled";
  if (record.result === "RESULT_PASSED") return "Passed";
  return "Unknown";
}

export function runDuration(run: CanvasesCanvasRun, now: number): string {
  const start = Date.parse(run.createdAt ?? "");
  const end = run.finishedAt ? Date.parse(run.finishedAt) : run.state === "STATE_FINISHED" ? NaN : now;
  return Number.isFinite(start) && Number.isFinite(end) ? formatCompactDuration(Math.max(0, end - start)) : "—";
}

export function toAutomationRun(run: CanvasesCanvasRun, orders: FactoriesWorkOrder[], now: number): AutomationRun {
  const order = findWorkOrderForAutomationRun(orders, run.id ?? "", run);
  return {
    id: run.id ?? "",
    title: run.rootEvent?.customName,
    task: order ? { id: order.key || order.id || "—", title: order.title || "Untitled task" } : undefined,
    status: recordedStatus(run),
    startedAt: run.createdAt ?? "",
    finishedAt: run.finishedAt,
    duration: runDuration(run, now),
    error: run.errors?.filter(Boolean).join("\n") || undefined,
    activities: [],
  };
}

export function toRunActivities(
  run: CanvasesCanvasRun,
  executions: CanvasesCanvasNodeExecution[],
  nodes: SuperplaneComponentsNode[],
): RunActivity[] {
  const references = run.executions ?? [];
  const ids = new Set(references.map((execution) => execution.id));
  const scoped = executions.filter((execution) =>
    execution.runId ? execution.runId === run.id : ids.has(execution.id),
  );
  const fullById = new Map(scoped.map((execution) => [execution.id, execution]));
  const merged: CanvasesCanvasNodeExecution[] = references.map((ref) => ({ ...ref, ...fullById.get(ref.id) }));
  const missing = scoped.filter((execution) => !ids.has(execution.id));
  const activities = [...merged, ...missing]
    .filter((execution) => execution.id)
    .map((execution) => executionActivity(execution, nodes));
  if (run.rootEvent?.id) {
    activities.push({
      id: `event:${run.rootEvent.id}`,
      name: "Run started",
      status: "Passed",
      logs: { state: "empty" },
      outputs: run.rootEvent.data,
      recordedAt: run.createdAt,
    });
  }
  for (const item of run.queueItems ?? []) {
    if (!item.id) continue;
    activities.push({
      id: `queue:${item.id}`,
      name: nodes.find((node) => node.id === item.nodeId)?.name || item.nodeId || "Queued execution",
      status: "Pending",
      recordedAt: item.createdAt,
      logs: { state: "empty" },
    });
  }
  return activities.sort((a, b) => (a.recordedAt ?? "").localeCompare(b.recordedAt ?? ""));
}

function executionActivity(
  execution: CanvasesCanvasNodeExecutionRef & CanvasesCanvasNodeExecution,
  nodes: SuperplaneComponentsNode[],
): RunActivity {
  const node = nodes.find((node) => node.id === execution.nodeId);
  const instructions = execution.configuration?.additionalInstructions;
  return {
    id: execution.id!,
    name: node?.name || execution.nodeId || "Execution",
    status: recordedStatus(execution),
    recordedAt: execution.createdAt,
    error: execution.result === "RESULT_FAILED" ? execution.resultMessage || undefined : undefined,
    logs: { state: "empty" },
    outputs: execution.outputs && Object.keys(execution.outputs).length ? execution.outputs : undefined,
    additionalInstructions: typeof instructions === "string" && instructions.trim() ? instructions : undefined,
  };
}
