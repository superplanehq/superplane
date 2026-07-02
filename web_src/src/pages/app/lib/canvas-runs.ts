import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { DEFAULT_EVENT_STATE_MAP, type EventState } from "@/ui/componentBase";
import { getNodeComponentName } from "../utils";

export function resolveNodeIconSlug(
  node: ComponentsNode | undefined,
  componentIconMap: Record<string, string>,
): string | undefined {
  const name = getNodeComponentName(node);
  if (!name) return undefined;
  return componentIconMap[name];
}

const STATUS_TO_EVENT_STATE: Record<string, EventState> = {
  completed: "success",
  success: "success",
  triggered: "triggered",
  finished: "success",
  passed: "success",
  approved: "success",
  opened: "success",
  true: "success",
  waiting: "queued",
  queued: "queued",
  pending: "queued",
  running: "running",
  failed: "failed",
  error: "error",
  rejected: "error",
  false: "error",
  timeout: "failed",
  cancelled: "cancelled",
  "pushed through": "running",
};

const STATUS_LABELS: Record<string, string> = {
  completed: "Completed",
  success: "Success",
  triggered: "Triggered",
  finished: "Finished",
  waiting: "Waiting",
  approved: "Approved",
  rejected: "Rejected",
  "pushed through": "Pushed Through",
  opened: "Opened",
  true: "True",
  false: "False",
  passed: "Passed",
  timeout: "Timeout",
  queued: "Queued",
  failed: "Failed",
  running: "Running",
  pending: "Pending",
  cancelled: "Cancelled",
  error: "Error",
};

export function getStatusBadgeProps(status: string) {
  const eventState = STATUS_TO_EVENT_STATE[status] || "neutral";
  const style = DEFAULT_EVENT_STATE_MAP[eventState];
  const label = STATUS_LABELS[status] || status || "Unknown";
  return { badgeColor: style.badgeColor, label };
}

export function countUnacknowledgedErrors(runs: CanvasesCanvasRun[]): number {
  let count = 0;
  for (const run of runs) {
    for (const exec of run.executions || []) {
      if (exec.result === "RESULT_FAILED" && exec.resultReason !== "RESULT_REASON_ERROR_RESOLVED") {
        count++;
      }
    }
  }
  return count;
}

export function findNode(nodes: ComponentsNode[], nodeId: string | undefined): ComponentsNode | undefined {
  if (!nodeId) return undefined;
  const baseNodeId = nodeId.includes(":") ? nodeId.split(":")[0] : nodeId;
  return nodes.find((n) => n.id === nodeId) || nodes.find((n) => n.id === baseNodeId);
}
