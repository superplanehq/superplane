import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  EventStateRegistry,
  StateFunction,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { GetHttpSyntheticCheckConfiguration, GetHttpSyntheticCheckNodeMetadata } from "./types";
import { formatTimeAgo } from "@/utils/date";

// Output channel names matching the backend constants
const CHANNEL_HEALTHY = "healthy";
const CHANNEL_DEGRADED = "degraded";
const CHANNEL_CRITICAL = "critical";

// Type for outputs with channel structure
type GetHttpSyntheticCheckOutputs = {
  default?: OutputPayload[];
  healthy?: OutputPayload[];
  degraded?: OutputPayload[];
  critical?: OutputPayload[];
};

export const getHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = getFirstPayload(context.execution);

    if (!payload) {
      return { Response: "No data returned" };
    }

    const responseData = payload.data as Record<string, unknown> | undefined;

    if (!responseData) {
      return { Response: "No data returned" };
    }

    const details: Record<string, string> = {};

    const metrics = responseData.metrics as Record<string, unknown> | undefined;
    const config = responseData.configuration as Record<string, unknown> | undefined;
    const metadata = config?.metadata as Record<string, unknown> | undefined;
    const configSpec = config?.spec as Record<string, unknown> | undefined;
    const plugin = configSpec?.plugin as Record<string, unknown> | undefined;
    const pluginSpec = plugin?.spec as Record<string, unknown> | undefined;
    const request = pluginSpec?.request as Record<string, unknown> | undefined;
    const assertions = pluginSpec?.assertions as Record<string, unknown> | undefined;
    const schedule = configSpec?.schedule as Record<string, unknown> | undefined;

    if (payload?.timestamp) {
      details["Executed At"] = new Date(payload.timestamp).toLocaleString();
    }

    const display = configSpec?.display as Record<string, unknown> | undefined;
    const name = metadata?.name || display?.name;
    if (name) {
      details["Name"] = String(name);
    }

    if (request?.url) {
      const method = request.method ? String(request.method).toUpperCase() : "GET";
      details["Target"] = `${method} ${request.url}`;
    }

    const criticalAssertions = assertions?.criticalAssertions as Array<Record<string, unknown>> | undefined;
    if (criticalAssertions && criticalAssertions.length > 0) {
      const parts = criticalAssertions.map((a) => {
        const kind = String(a.kind || "").replace(/_/g, " ");
        const spec = a.spec as Record<string, unknown> | undefined;
        const operator = spec?.operator ? String(spec.operator) : "";
        const value = spec?.value ? String(spec.value) : "";
        return `${kind} ${operator} ${value}`.trim();
      });
      details["Expected"] = parts.join(", ");
    }

    if (schedule) {
      const parts: string[] = [];
      if (schedule.interval) parts.push(`every ${schedule.interval}`);
      if (Array.isArray(schedule.locations) && schedule.locations.length > 0) {
        const validLocations = schedule.locations.filter((loc): loc is string => typeof loc === "string");
        if (validLocations.length > 0) {
          parts.push(`from ${validLocations.join(", ")}`);
        }
      }
      if (schedule.strategy) parts.push(`(${String(schedule.strategy).replace(/_/g, " ")})`);
      details["Scheduling"] = parts.join(" ");
    }

    // Show metrics or mark as Not Available
    if (metrics) {
      details["Last Outcome"] = metrics.lastOutcome != null ? String(metrics.lastOutcome) : "Not Available";
      details["Total Runs (24h)"] = metrics.totalRuns24h != null ? String(metrics.totalRuns24h) : "Not Available";
      details["Healthy Runs (24h)"] = metrics.healthyRuns24h != null ? String(metrics.healthyRuns24h) : "Not Available";
      details["Critical Runs (24h)"] =
        metrics.criticalRuns24h != null ? String(metrics.criticalRuns24h) : "Not Available";
    }

    if (configSpec?.enabled != null) {
      details["Enabled"] = configSpec.enabled ? "Yes" : "No";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as GetHttpSyntheticCheckNodeMetadata | undefined;
  const configuration = node.configuration as GetHttpSyntheticCheckConfiguration;

  if (nodeMetadata?.checkName) {
    metadata.push({ icon: "activity", label: nodeMetadata.checkName });
  } else if (configuration?.checkId) {
    // Fallback to check ID if name is not yet available (e.g., before first setup)
    const idPreview =
      configuration.checkId.length > 24 ? configuration.checkId.substring(0, 24) + "…" : configuration.checkId;
    metadata.push({ icon: "fingerprint", label: idPreview });
  }

  if (configuration?.dataset) {
    metadata.push({ icon: "database", label: configuration.dataset });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.createdAt || !execution.rootEvent?.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const timeAgo = formatTimeAgo(new Date(execution.createdAt));
  const activeChannel = getActiveChannel(execution);
  const statusLabel = activeChannel ? channelLabel(activeChannel) : null;
  const eventSubtitle = statusLabel ? `${statusLabel} · ${timeAgo}` : timeAgo;

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}

/**
 * Extracts the first payload from execution outputs, checking all possible channels.
 */
function getFirstPayload(execution: ExecutionInfo): OutputPayload | null {
  const outputs = execution.outputs as GetHttpSyntheticCheckOutputs | undefined;
  if (!outputs) return null;

  for (const channel of [CHANNEL_CRITICAL, CHANNEL_DEGRADED, CHANNEL_HEALTHY]) {
    const channelOutputs = outputs[channel as keyof GetHttpSyntheticCheckOutputs];
    if (channelOutputs && channelOutputs.length > 0) {
      return channelOutputs[0];
    }
  }

  // Check empty channel (no status case)
  const emptyChannel = outputs["" as keyof GetHttpSyntheticCheckOutputs];
  if (emptyChannel && emptyChannel.length > 0) {
    return emptyChannel[0];
  }

  // Fallback to default channel for backwards compatibility
  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

/**
 * Determines which output channel has data, indicating the check state.
 * Returns empty string for no status case (empty channel from backend).
 */
function getActiveChannel(execution: ExecutionInfo): string | null {
  const outputs = execution.outputs as GetHttpSyntheticCheckOutputs | undefined;
  if (!outputs) return null;

  if (outputs.critical && outputs.critical.length > 0) return CHANNEL_CRITICAL;
  if (outputs.degraded && outputs.degraded.length > 0) return CHANNEL_DEGRADED;
  if (outputs.healthy && outputs.healthy.length > 0) return CHANNEL_HEALTHY;
  if (outputs.default && outputs.default.length > 0) return "default";

  // Check for empty channel (no status case)
  // When backend emits to empty channel, it appears as a channel with empty string key
  const emptyChannel = outputs["" as keyof GetHttpSyntheticCheckOutputs];
  if (emptyChannel && emptyChannel.length > 0) return "";

  return null;
}

function channelLabel(channel: string): string {
  switch (channel) {
    case CHANNEL_CRITICAL:
      return "failing";
    case CHANNEL_DEGRADED:
      return "degraded";
    case CHANNEL_HEALTHY:
      return "passing";
    case "":
      return "no status";
    default:
      return "";
  }
}

// --- State registry ---

export const GET_HTTP_SYNTHETIC_CHECK_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  healthy: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-green-500",
  },
  degraded: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-amber-100",
    badgeColor: "bg-amber-500",
  },
  critical: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  noStatus: {
    icon: "alert-circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-400",
  },
};

export const getHttpSyntheticCheckStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const activeChannel = getActiveChannel(execution);

    if (activeChannel === CHANNEL_CRITICAL) return "critical";
    if (activeChannel === CHANNEL_DEGRADED) return "degraded";
    if (activeChannel === CHANNEL_HEALTHY) return "healthy";
    if (activeChannel === "") return "noStatus";

    return "healthy";
  }

  return "failed";
};

export const GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GET_HTTP_SYNTHETIC_CHECK_STATE_MAP,
  getState: getHttpSyntheticCheckStateFunction,
};
