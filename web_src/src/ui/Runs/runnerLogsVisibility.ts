import type { CanvasesCanvasNodeExecution } from "@/api-client";
import type { RunInspectorNodeSection } from "./types";

const RUNNER_COMPONENTS = new Set(["runner", "runnerBash", "runnerJS", "runnerPython"]);
const BROKER_TASK_ID_METADATA_KEY = "runner_broker_task_id";

export function shouldShowRunnerLogs(section: RunInspectorNodeSection): boolean {
  if (section.isTrigger || !section.execution?.id) {
    return false;
  }

  const component = section.workflowNode?.component ?? "";
  return RUNNER_COMPONENTS.has(component) && hasBrokerTaskId(section.execution);
}

function hasBrokerTaskId(execution: CanvasesCanvasNodeExecution): boolean {
  const metadata = execution.metadata;
  if (!metadata || typeof metadata !== "object") {
    return false;
  }

  const taskId = (metadata as Record<string, unknown>)[BROKER_TASK_ID_METADATA_KEY];
  if (typeof taskId === "string") {
    return taskId.trim() !== "";
  }

  return taskId !== undefined && taskId !== null && String(taskId).trim() !== "";
}
