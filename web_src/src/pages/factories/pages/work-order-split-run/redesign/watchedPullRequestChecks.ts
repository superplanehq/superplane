export const WAIT_FOR_PULL_REQUEST_CHECKS_COMPONENT = "github.waitForPullRequestChecks";
const WAIT_FOR_BUILDS_COMPONENT = "bitbucket.waitForBuilds";

const WAIT_FOR_CHECKS_COMPONENTS = new Set([WAIT_FOR_PULL_REQUEST_CHECKS_COMPONENT, WAIT_FOR_BUILDS_COMPONENT]);

const FAILING_CONCLUSIONS = new Set(["failure", "error", "timed_out", "action_required", "failed", "stopped"]);
const PASSING_CONCLUSIONS = new Set(["success", "neutral", "skipped", "cancelled"]);
const OUTPUT_CHANNELS = ["passed", "failed", "timedOut", "default"] as const;

export type WatchedCheckStatus = "Pending" | "Failed" | "Passed";

export type WatchedPullRequestCheck = {
  name: string;
  status: WatchedCheckStatus;
  description?: string;
  detailsUrl?: string;
  summary?: string;
};

type CanvasNodeRef = {
  id?: string;
  component?: string;
  configuration?: unknown;
};

export function waitForPullRequestChecksNode(
  nodes: CanvasNodeRef[] | undefined,
): { id: string; checkNames: string[] } | undefined {
  const node = nodes?.find((item) => item.component && WAIT_FOR_CHECKS_COMPONENTS.has(item.component) && item.id);
  if (!node?.id) {
    return undefined;
  }
  return { id: node.id, checkNames: configuredCheckNames(node.configuration) };
}

function configuredCheckNames(configuration: unknown): string[] {
  if (!isRecord(configuration)) {
    return [];
  }
  if (Object.prototype.hasOwnProperty.call(configuration, "checkNames")) {
    return uniqueNames(configuration.checkNames);
  }
  if (Object.prototype.hasOwnProperty.call(configuration, "buildKeys")) {
    return uniqueNames(configuration.buildKeys);
  }
  return [];
}

function uniqueNames(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  const names: string[] = [];
  const seen = new Set<string>();
  for (const item of value) {
    if (typeof item !== "string") {
      continue;
    }
    const name = item.trim();
    if (!name) {
      continue;
    }
    const key = name.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    names.push(name);
  }
  return names;
}

export function watchedPullRequestChecksFromExecutions(
  executions: unknown,
  nodeId: string,
  configuredNames: string[] = [],
): WatchedPullRequestCheck[] {
  const execution = latestExecutionForNode(executions, nodeId);
  if (!execution) {
    return [];
  }
  return selectedChecksForExecution(execution, configuredNames);
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

function selectedChecksForExecution(
  execution: Record<string, unknown>,
  configuredNames: string[],
): WatchedPullRequestCheck[] {
  const fromMetadata = selectedChecksList(execution.metadata);
  if (fromMetadata && fromMetadata.length > 0) {
    return fromMetadata;
  }
  if (execution.state !== "STATE_FINISHED") {
    return pendingConfiguredChecks(configuredNames);
  }
  return selectedChecksFromOutputs(execution.outputs) ?? [];
}

function pendingConfiguredChecks(names: string[]): WatchedPullRequestCheck[] {
  return names.map((name) => ({ name, status: "Pending" as const }));
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
  if (!isRecord(source)) {
    return undefined;
  }
  if (Object.prototype.hasOwnProperty.call(source, "selectedChecks")) {
    return checkList(source.selectedChecks);
  }
  if (Object.prototype.hasOwnProperty.call(source, "selectedBuilds")) {
    return checkList(source.selectedBuilds);
  }
  return undefined;
}

function checkList(value: unknown): WatchedPullRequestCheck[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }
  return value.flatMap(watchedCheckFromValue);
}

function watchedCheckFromValue(value: unknown): WatchedPullRequestCheck[] {
  if (!isRecord(value)) {
    return [];
  }
  const name = checkName(value);
  if (!name) {
    return [];
  }
  const description = typeof value.description === "string" ? value.description.trim() : "";
  const detailsUrl = typeof value.detailsUrl === "string" ? value.detailsUrl.trim() : "";
  const summary = typeof value.summary === "string" ? value.summary.trim() : "";
  return [
    {
      name,
      status: watchedCheckStatus(stringField(value.status), stringField(value.conclusion)),
      ...(description ? { description } : {}),
      ...(detailsUrl ? { detailsUrl } : {}),
      ...(summary ? { summary } : {}),
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

function checkName(value: Record<string, unknown>): string {
  const name = stringField(value.name).trim();
  if (name) {
    return name;
  }
  return stringField(value.key).trim();
}

function stringField(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
