import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";
import type { ComponentBaseProps, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import type React from "react";

type EndpointConfiguration = {
  url?: string;
  method?: string;
  expectedStatus?: string;
  intervalSeconds?: number;
  timeoutSeconds?: number;
};

type EndpointMetadata = {
  startedAt?: string;
  deadlineAt?: string;
  attempts?: number;
  lastStatus?: number | null;
  lastError?: string;
  nextAttemptAt?: string;
  resolvedUrl?: string;
};

type EndpointOutput = {
  status?: number | null;
  attempts?: number;
  elapsedMs?: number;
  reason?: string;
  lastStatus?: number | null;
  lastError?: string;
  resolvedUrl?: string;
};

type EndpointOutputs = {
  ready?: OutputPayload[];
  timeout?: OutputPayload[];
};

type ExecutionDetail = [string, string | undefined];

export const WAIT_FOR_ENDPOINT_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  running: {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-sky-100",
    badgeColor: "bg-blue-500",
  },
  ready: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  timeout: {
    icon: "clock-alert",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  error: {
    icon: "triangle-alert",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export function waitForEndpointStateFunction(execution: ExecutionInfo): EventState {
  if (isExecutionError(execution)) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_CANCELLING") {
    return "cancelling";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const outputs = execution.outputs as EndpointOutputs | undefined;
    if (outputs?.ready?.length) {
      return "ready";
    }
    if (outputs?.timeout?.length) {
      return "timeout";
    }
  }

  return "failed";
}

export const WAIT_FOR_ENDPOINT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: WAIT_FOR_ENDPOINT_STATE_MAP,
  getState: waitForEndpointStateFunction,
};

export const waitForEndpointMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return {
      iconSlug: context.componentDefinition.icon || "activity",
      iconColor: getColorClass("black"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title: context.node.name || context.componentDefinition.label || "Wait for Endpoint",
      includeEmptyState: context.lastExecutions.length === 0,
      metadata: endpointMetadata(context.node),
      eventStateMap: WAIT_FOR_ENDPOINT_STATE_MAP,
    };
  },

  subtitle(context: SubtitleContext): React.ReactNode {
    return endpointSubtitle(context.execution);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.execution.configuration as EndpointConfiguration | undefined;
    const metadata = context.execution.metadata as EndpointMetadata | undefined;
    const outputs = context.execution.outputs as EndpointOutputs | undefined;
    const output = firstOutput(outputs?.ready) ?? firstOutput(outputs?.timeout);

    return compactDetails([
      endpointDetail(configuration),
      expectedStatusDetail(configuration),
      attemptsDetail(metadata),
      lastStatusDetail(output, metadata),
      elapsedDetail(output),
      resolvedUrlDetail(output, metadata),
      lastErrorDetail(output, metadata),
    ]);
  },
};

function isExecutionError(execution: ExecutionInfo): boolean {
  if (!execution.resultMessage) {
    return false;
  }
  if (execution.resultReason === "RESULT_REASON_ERROR") {
    return true;
  }
  return execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED";
}

function endpointSubtitle(execution: ExecutionInfo): React.ReactNode {
  const state = waitForEndpointStateFunction(execution);
  const metadata = execution.metadata as EndpointMetadata | undefined;
  const outputs = execution.outputs as EndpointOutputs | undefined;

  switch (state) {
    case "running":
      return runningSubtitle(metadata);
    case "ready":
      return readySubtitle(firstOutput(outputs?.ready));
    case "timeout":
      return timeoutSubtitle(firstOutput(outputs?.timeout));
    default:
      return execution.updatedAt ? renderTimeAgo(new Date(execution.updatedAt)) : "";
  }
}

function runningSubtitle(metadata?: EndpointMetadata): string {
  const attempt = metadata?.attempts ?? 0;
  if (metadata?.lastStatus != null) {
    return `Attempt ${attempt} - last status ${metadata.lastStatus}`;
  }
  if (metadata?.lastError) {
    return `Attempt ${attempt} - ${metadata.lastError}`;
  }
  return attempt > 0 ? `Waiting after attempt ${attempt}` : "Checking endpoint...";
}

function readySubtitle(output?: EndpointOutput): string {
  return output?.status != null ? `Ready - status ${output.status}` : "Endpoint ready";
}

function timeoutSubtitle(output?: EndpointOutput): string {
  return output?.attempts ? `Timed out after ${output.attempts} attempts` : "Readiness timed out";
}

function endpointLabel(configuration?: EndpointConfiguration): string | undefined {
  if (!configuration?.url) {
    return undefined;
  }
  return `${configuration.method || "GET"} ${configuration.url}`;
}

function endpointDetail(configuration?: EndpointConfiguration): ExecutionDetail {
  return ["Endpoint", endpointLabel(configuration)];
}

function expectedStatusDetail(configuration?: EndpointConfiguration): ExecutionDetail {
  return ["Expected Status", configuration?.expectedStatus];
}

function attemptsDetail(metadata?: EndpointMetadata): ExecutionDetail {
  return ["Attempts", numberString(metadata?.attempts)];
}

function lastStatusDetail(output?: EndpointOutput, metadata?: EndpointMetadata): ExecutionDetail {
  return ["Last Status", numberString(output?.status ?? output?.lastStatus ?? metadata?.lastStatus)];
}

function elapsedDetail(output?: EndpointOutput): ExecutionDetail {
  return ["Elapsed", output?.elapsedMs == null ? undefined : formatDuration(output.elapsedMs)];
}

function resolvedUrlDetail(output?: EndpointOutput, metadata?: EndpointMetadata): ExecutionDetail {
  return ["Resolved URL", output?.resolvedUrl ?? metadata?.resolvedUrl];
}

function lastErrorDetail(output?: EndpointOutput, metadata?: EndpointMetadata): ExecutionDetail {
  return ["Last Error", output?.lastError ?? metadata?.lastError];
}

function numberString(value?: number | null): string | undefined {
  return value == null ? undefined : String(value);
}

function compactDetails(entries: ExecutionDetail[]): Record<string, string> {
  return Object.fromEntries(entries.filter((entry): entry is [string, string] => entry[1] !== undefined));
}

function endpointMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as EndpointConfiguration | undefined;
  if (!configuration) {
    return [];
  }

  const metadata: MetadataItem[] = [];
  if (configuration.url) {
    metadata.push({
      icon: "link",
      label: `${configuration.method || "GET"} ${configuration.url}`,
    });
  }

  if (configuration.intervalSeconds || configuration.timeoutSeconds) {
    const interval = configuration.intervalSeconds ?? 10;
    const timeout = configuration.timeoutSeconds ?? 300;
    metadata.push({
      icon: "clock",
      label: `Every ${interval}s, timeout ${timeout}s`,
    });
  }

  return metadata;
}

function firstOutput(payloads?: OutputPayload[]): EndpointOutput | undefined {
  return payloads?.[0]?.data as EndpointOutput | undefined;
}

function formatDuration(milliseconds: number): string {
  if (milliseconds < 1000) {
    return `${milliseconds}ms`;
  }
  if (milliseconds < 60000) {
    return `${Math.round(milliseconds / 1000)}s`;
  }
  return `${Math.round(milliseconds / 60000)}m`;
}
