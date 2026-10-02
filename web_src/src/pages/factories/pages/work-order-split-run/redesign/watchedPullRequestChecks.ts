export const WAIT_FOR_PULL_REQUEST_CHECKS_COMPONENT = "github.waitForPullRequestChecks";

const FAILING_CONCLUSIONS = new Set(["failure", "error", "timed_out", "action_required"]);
const PASSING_CONCLUSIONS = new Set(["success", "neutral", "skipped", "cancelled"]);
const OUTPUT_CHANNELS = ["passed", "failed", "timedOut", "default"] as const;

export type WatchedCheckStatus = "Pending" | "Failed" | "Passed";

export type WatchedPullRequestCheck = {
  name: string;
  status: WatchedCheckStatus;
  detailsUrl?: string;
};

type CanvasNodeRef = {
  id?: string;
  component?: string;
};

export function waitForPullRequestChecksNodeId(nodes: CanvasNodeRef[] | undefined): string | undefined {
  return nodes?.find((node) => node.component === WAIT_FOR_PULL_REQUEST_CHECKS_COMPONENT && node.id)?.id;
}

export function watchedPullRequestChecksFromExecutions(executions: unknown, nodeId: string): WatchedPullRequestCheck[] {
  const execution = latestExecutionForNode(executions, nodeId);
  if (!execution) {
    return [];
  }
  return selectedChecksForExecution(execution);
}

export function watchedCheckStatus(status: string, conclusion: string): WatchedCheckStatus {
  if (status.trim().toLowerCase() !== "completed") {
    return "Pending";
  }
  const normalized = conclusion.trim().toLowerCase();
  if (FAILING_CONCLUSIONS.has(normalized)) {
    return "Failed";
  }
  if (normalized !== "" && !PASSING_CONCLUSIONS.has(normalized)) {
    return "Pending";
  }
  return "Passed";
}

function selectedChecksForExecution(execution: Record<string, unknown>): WatchedPullRequestCheck[] {
  const fromMetadata = selectedChecksList(execution.metadata);
  if (fromMetadata) {
    return fromMetadata;
  }
  if (execution.state !== "STATE_FINISHED") {
    return [];
  }
  return selectedChecksFromOutputs(execution.outputs) ?? [];
}

function selectedChecksFromOutputs(outputs: unknown): WatchedPullRequestCheck[] | undefined {
  if (!isRecord(outputs)) {
    return undefined;
  }
  for (const channel of OUTPUT_CHANNELS) {
    const list = selectedChecksFromChannel(outputs[channel]);
    if (list) {
      return list;
    }
  }
  return undefined;
}

function selectedChecksFromChannel(channel: unknown): WatchedPullRequestCheck[] | undefined {
  if (!Array.isArray(channel)) {
    return undefined;
  }
  for (const item of channel) {
    if (!isRecord(item)) {
      continue;
    }
    const payload = isRecord(item.data) ? item.data : item;
    const list = selectedChecksList(payload);
    if (list) {
      return list;
    }
  }
  return undefined;
}

function selectedChecksList(source: unknown): WatchedPullRequestCheck[] | undefined {
  if (!isRecord(source) || !Object.prototype.hasOwnProperty.call(source, "selectedChecks")) {
    return undefined;
  }
  if (!Array.isArray(source.selectedChecks)) {
    return undefined;
  }
  return source.selectedChecks.flatMap(watchedCheckFromValue);
}

function watchedCheckFromValue(value: unknown): WatchedPullRequestCheck[] {
  if (!isRecord(value) || typeof value.name !== "string") {
    return [];
  }
  const name = value.name.trim();
  if (!name) {
    return [];
  }
  const detailsUrl = typeof value.detailsUrl === "string" ? value.detailsUrl.trim() : "";
  return [
    {
      name,
      status: watchedCheckStatus(stringField(value.status), stringField(value.conclusion)),
      ...(detailsUrl ? { detailsUrl } : {}),
    },
  ];
}

function latestExecutionForNode(executions: unknown, nodeId: string): Record<string, unknown> | undefined {
  if (!Array.isArray(executions)) {
    return undefined;
  }
  let latest: Record<string, unknown> | undefined;
  for (const execution of executions) {
    if (!isRecord(execution) || execution.nodeId !== nodeId) {
      continue;
    }
    if (!latest || createdAtMs(execution.createdAt) >= createdAtMs(latest.createdAt)) {
      latest = execution;
    }
  }
  return latest;
}

function createdAtMs(value: unknown): number {
  if (typeof value !== "string") {
    return 0;
  }
  const time = Date.parse(value);
  return Number.isFinite(time) ? time : 0;
}

function stringField(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
