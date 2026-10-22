import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  FactoriesWorkOrder,
  SuperplaneComponentsNode,
} from "@/api-client";
import { intakeRunPresentation, recordedExistingTaskId, type RunHistoryIntakeSource } from "./intakeRunData";
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

export function toAutomationRun(
  run: CanvasesCanvasRun,
  orders: FactoriesWorkOrder[],
  now: number,
  intakeSource?: RunHistoryIntakeSource,
): AutomationRun {
  const order = intakeSource ? intakeWorkOrder(run, orders) : findWorkOrderForAutomationRun(orders, run.id ?? "", run);
  return {
    ...(intakeSource ? intakeRunPresentation(run, intakeSource, recordedStatus(run), orders) : {}),
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

function intakeWorkOrder(run: CanvasesCanvasRun, orders: FactoriesWorkOrder[]): FactoriesWorkOrder | undefined {
  const created = orders.find((order) => Boolean(run.id) && order.sourceRunId === run.id);
  if (created) return created;
  const existingTaskId = recordedExistingTaskId(run);
  return existingTaskId ? orders.find((order) => order.id === existingTaskId) : undefined;
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
      name: "Source event",
      status: "Passed",
      outputs: run.rootEvent.data,
      defaultTab: "outputs",
      outputsLabel: "Event data",
      recordedAt: run.createdAt,
    });
  }
  for (const item of run.queueItems ?? []) {
    if (!item.id) continue;
    const node = nodes.find((node) => node.id === item.nodeId);
    activities.push({
      id: `queue:${item.id}`,
      name: node?.name || item.nodeId || "Queued execution",
      status: "Pending",
      recordedAt: item.createdAt,
      logs: node?.component?.startsWith("runner") ? { state: "waiting" } : undefined,
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
  const hasLogs = hasRunnerLogs(execution, node);
  return {
    id: execution.id!,
    name: node?.name || execution.nodeId || "Execution",
    status: recordedStatus(execution),
    recordedAt: execution.createdAt,
    error: execution.result === "RESULT_FAILED" ? execution.resultMessage || undefined : undefined,
    logs: hasLogs ? { state: "empty" } : undefined,
    defaultTab: hasLogs ? "logs" : "outputs",
    outputs: execution.outputs && Object.keys(execution.outputs).length ? execution.outputs : undefined,
    additionalInstructions: typeof instructions === "string" && instructions.trim() ? instructions : undefined,
  };
}

function hasRunnerLogs(
  execution: CanvasesCanvasNodeExecutionRef & CanvasesCanvasNodeExecution,
  node?: SuperplaneComponentsNode,
): boolean {
  const component = execution.componentName || node?.component;
  return Boolean(execution.metadata?.runner_broker_task_id || component?.startsWith("runner"));
}
