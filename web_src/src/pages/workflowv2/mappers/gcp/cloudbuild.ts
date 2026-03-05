import { DEFAULT_EVENT_STATE_MAP, EventState } from "@/ui/componentBase";
import { EventStateRegistry, ExecutionInfo, OutputPayload, StateFunction } from "../types";

type CloudBuildGitSource = {
  revision?: string;
  url?: string;
};

type CloudBuildTimingWindow = {
  endTime?: string;
  startTime?: string;
};

type CloudBuildStep = {
  args?: string[];
  entrypoint?: string;
  name?: string;
  status?: string;
};

type CloudBuildResults = {
  buildStepImages?: string[];
  buildStepOutputs?: string[];
};

type CloudBuildOptions = {
  dynamicSubstitutions?: boolean;
  logging?: string;
  substitutionOption?: string;
};

export type CloudBuildData = {
  buildTriggerId?: string;
  createTime?: string;
  finishTime?: string;
  id?: string;
  logUrl?: string;
  name?: string;
  options?: CloudBuildOptions;
  projectId?: string;
  queueTtl?: string;
  results?: CloudBuildResults;
  serviceAccount?: string;
  source?: {
    gitSource?: CloudBuildGitSource;
  };
  sourceProvenance?: {
    resolvedGitSource?: CloudBuildGitSource;
  };
  startTime?: string;
  status?: string;
  steps?: CloudBuildStep[];
  substitutions?: Record<string, string>;
  tags?: string[];
  timeout?: string;
  timing?: Record<string, CloudBuildTimingWindow>;
};

type CloudBuildOutputPayload = OutputPayload & {
  data?: CloudBuildData;
};

export function getCloudBuildOutputPayload(execution: ExecutionInfo): CloudBuildOutputPayload | undefined {
  const outputs = execution.outputs as
    | { passed?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] }
    | undefined;
  const payload = outputs?.passed?.[0] ?? outputs?.failed?.[0] ?? outputs?.default?.[0];
  if (!payload || typeof payload !== "object") {
    return undefined;
  }

  return payload as CloudBuildOutputPayload;
}

export function getCloudBuildData(execution: ExecutionInfo): CloudBuildData | undefined {
  const payload = getCloudBuildOutputPayload(execution);
  if (payload?.data) {
    return payload.data;
  }

  const metadata = execution.metadata as { build?: CloudBuildData } | undefined;
  return metadata?.build;
}

export const cloudBuildExecutionStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) {
    return "neutral";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  const buildState = cloudBuildStatusToExecutionState(getCloudBuildData(execution)?.status);
  if (buildState) {
    return buildState;
  }

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "success";
  }

  return "failed";
};

export const CLOUD_BUILD_EXECUTION_STATE_REGISTRY: EventStateRegistry = {
  stateMap: DEFAULT_EVENT_STATE_MAP,
  getState: cloudBuildExecutionStateFunction,
};

export const CLOUD_BUILD_CREATE_STATE_REGISTRY = CLOUD_BUILD_EXECUTION_STATE_REGISTRY;

export function buildCloudBuildSummaryDetails({
  build,
  timestamp,
  receivedAt,
}: {
  build?: CloudBuildData;
  timestamp?: string;
  receivedAt?: string;
}): Record<string, string> {
  const details: Record<string, string> = {};

  const ts = receivedAt ?? timestamp;
  if (ts) {
    const label = receivedAt ? "Received At" : "Built At";
    const formatted = formatDateTime(ts);
    if (formatted) details[label] = formatted;
  }

  if (build?.status) {
    details["Status"] = build.status;
  }

  if (build?.id) {
    details["Build ID"] = build.id;
  }

  const branch = build?.substitutions?.["BRANCH_NAME"];
  if (branch) {
    details["Branch"] = branch;
  }

  const repo = build?.substitutions?.["REPO_FULL_NAME"];
  if (repo) {
    details["Repository"] = repo;
  }

  if (build?.logUrl) {
    details["Log URL"] = build.logUrl;
  }

  return details;
}

export function buildCloudBuildDetails({
  build,
  receivedAt,
  timestamp,
  type,
}: {
  build?: CloudBuildData;
  receivedAt?: string;
  timestamp?: string;
  type?: string;
}): Record<string, string> {
  const details: Record<string, string> = {};

  addDateDetail(details, "Received At", receivedAt ?? timestamp);
  addDetail(details, "Event Type", type);

  if (!build) {
    return details;
  }

  addDetail(details, "Build ID", build.id);
  addDetail(details, "Build Name", build.name);
  addDetail(details, "Project", build.projectId);
  addDetail(details, "Status", build.status);
  addDetail(details, "Trigger ID", build.buildTriggerId);
  addDateDetail(details, "Created At", build.createTime);
  addDateDetail(details, "Started At", build.startTime);
  addDateDetail(details, "Finished At", build.finishTime);
  addDetail(details, "Log URL", build.logUrl);
  addDetail(details, "Source", formatSource(build.source?.gitSource));
  addDetail(details, "Resolved Source", formatSource(build.sourceProvenance?.resolvedGitSource));
  addDetail(details, "Service Account", build.serviceAccount);
  addDetail(details, "Logging", build.options?.logging);
  addDetail(details, "Dynamic Substitutions", formatBoolean(build.options?.dynamicSubstitutions));
  addDetail(details, "Substitution Option", build.options?.substitutionOption);
  addDetail(details, "Queue TTL", build.queueTtl);
  addDetail(details, "Timeout", build.timeout);
  addDetail(details, "Tags", build.tags?.join(", "));
  addDetail(details, "Steps", formatSteps(build.steps));
  addDetail(details, "Results", formatResults(build.results));
  addDetail(details, "Substitutions", formatSubstitutions(build.substitutions));
  addDetail(details, "Timing Phases", formatTiming(build.timing));

  return details;
}

export function buildCloudBuildEventSubtitle(build?: CloudBuildData): string {
  return [build?.status, build?.id].filter((value): value is string => Boolean(value)).join(" · ");
}

export function cloudBuildStatusToTriggerState(status?: string): EventState {
  return cloudBuildStatusToExecutionState(status) ?? "triggered";
}

function cloudBuildStatusToExecutionState(status?: string): EventState | undefined {
  switch (status?.toUpperCase()) {
    case "PENDING":
    case "QUEUED":
      return "running";
    case "WORKING":
      return "running";
    case "SUCCESS":
      return "success";
    case "CANCELLED":
      return "cancelled";
    case "FAILURE":
    case "INTERNAL_ERROR":
    case "TIMEOUT":
    case "EXPIRED":
      return "failed";
    default:
      return undefined;
  }
}

function addDetail(details: Record<string, string>, key: string, value?: string) {
  if (!value) {
    return;
  }

  details[key] = value;
}

function addDateDetail(details: Record<string, string>, key: string, value?: string) {
  const formatted = formatDateTime(value);
  if (!formatted) {
    return;
  }

  details[key] = formatted;
}

function formatDateTime(value?: string): string | undefined {
  if (!value) {
    return undefined;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toLocaleString();
}

function formatSource(source?: CloudBuildGitSource): string | undefined {
  if (!source) {
    return undefined;
  }

  const parts = [source.url, source.revision].filter((value): value is string => Boolean(value));
  if (parts.length === 0) {
    return undefined;
  }

  if (parts.length === 2) {
    return `${parts[0]} @ ${parts[1]}`;
  }

  return parts[0];
}

function formatBoolean(value?: boolean): string | undefined {
  if (value === undefined) {
    return undefined;
  }

  return value ? "Yes" : "No";
}

function formatSteps(steps?: CloudBuildStep[]): string | undefined {
  if (!steps || steps.length === 0) {
    return undefined;
  }

  return steps
    .map((step) => {
      const identity = [step.name, step.entrypoint].filter((value): value is string => Boolean(value)).join(" / ");
      const prefix = identity || "Unnamed step";
      return step.status ? `${prefix} (${step.status})` : prefix;
    })
    .join("; ");
}

function formatResults(results?: CloudBuildResults): string | undefined {
  if (!results) {
    return undefined;
  }

  const parts: string[] = [];
  if (results.buildStepImages?.length) {
    parts.push(`Step images: ${results.buildStepImages.length}`);
  }
  if (results.buildStepOutputs?.length) {
    parts.push(`Step outputs: ${results.buildStepOutputs.length}`);
  }

  return parts.length > 0 ? parts.join(", ") : undefined;
}

function formatSubstitutions(substitutions?: Record<string, string>): string | undefined {
  if (!substitutions) {
    return undefined;
  }

  const entries = Object.entries(substitutions);
  if (entries.length === 0) {
    return undefined;
  }

  return entries.map(([key, value]) => `${key}=${value}`).join(", ");
}

function formatTiming(timing?: Record<string, CloudBuildTimingWindow>): string | undefined {
  if (!timing) {
    return undefined;
  }

  const phases = Object.keys(timing);
  if (phases.length === 0) {
    return undefined;
  }

  return phases.join(", ");
}
