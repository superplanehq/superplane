import {
  ComponentsNode,
  ComponentsComponent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import {
  ComponentBaseProps,
  EventSection,
  ComponentBaseSpec,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload, EventStateRegistry, StateFunction } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import dash0Icon from "@/assets/icons/integrations/dash0.svg";
import { ListIssuesConfiguration, PrometheusResponse } from "./types";
import { formatTimeAgo } from "@/utils/date";

// Output channel names matching the backend constants
const CHANNEL_CLEAR = "clear";
const CHANNEL_DEGRADED = "degraded";
const CHANNEL_CRITICAL = "critical";

// Type for outputs with new channel structure
type ListIssuesOutputs = {
  default?: OutputPayload[];
  clear?: OutputPayload[];
  degraded?: OutputPayload[];
  critical?: OutputPayload[];
};

/**
 * Extracts the first payload from execution outputs, checking all possible channels.
 * Supports both the new channel-based outputs (clear/degraded/critical) and
 * the legacy default channel for backwards compatibility.
 */
function getFirstPayload(execution: WorkflowsWorkflowNodeExecution): OutputPayload | null {
  const outputs = execution.outputs as ListIssuesOutputs | undefined;
  if (!outputs) return null;

  // Check new channel-based outputs first (in severity order)
  for (const channel of [CHANNEL_CRITICAL, CHANNEL_DEGRADED, CHANNEL_CLEAR]) {
    const channelOutputs = outputs[channel as keyof ListIssuesOutputs];
    if (channelOutputs && channelOutputs.length > 0) {
      return channelOutputs[0];
    }
  }

  // Fallback to default channel for backwards compatibility
  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

/**
 * Determines which output channel has data, indicating the issue state.
 * Returns the channel name or null if no output found.
 */
function getActiveChannel(execution: WorkflowsWorkflowNodeExecution): string | null {
  const outputs = execution.outputs as ListIssuesOutputs | undefined;
  if (!outputs) return null;

  // Check new channel-based outputs
  if (outputs.critical && outputs.critical.length > 0) return CHANNEL_CRITICAL;
  if (outputs.degraded && outputs.degraded.length > 0) return CHANNEL_DEGRADED;
  if (outputs.clear && outputs.clear.length > 0) return CHANNEL_CLEAR;

  // Fallback to default channel
  if (outputs.default && outputs.default.length > 0) return "default";

  return null;
}

export const listIssuesMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name!;

    const configuration = node.configuration as unknown as ListIssuesConfiguration;
    const specs = getSpecs(configuration);

    return {
      iconSrc: dash0Icon,
      collapsedBackground: "bg-white",
      collapsed: node.isCollapsed,
      title: node.name!,
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      specs,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution, additionalData?: unknown): string {
    // Check if this is being called from ChainItem (which passes additionalData as undefined or a different structure)
    // For ChainItem, just return the time without counts
    const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

    // If additionalData is explicitly a marker object indicating ChainItem context, skip counts
    // Otherwise, include counts for SidebarEventItem
    if (additionalData && typeof additionalData === "object" && "skipIssueCounts" in additionalData) {
      return timeAgo;
    }

    const { critical, degraded } = getIssueCounts(execution);

    // Build subtitle with counts and time
    const countParts: string[] = [];
    if (critical > 0) {
      countParts.push(`${critical} critical`);
    }
    if (degraded > 0) {
      countParts.push(`${degraded} degraded`);
    }

    if (countParts.length > 0) {
      return `${countParts.join(", ")} · ${timeAgo}`;
    }

    // No issues found - show "no issues" with time
    return `no issues · ${timeAgo}`;
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};

    // Add "Checked at" timestamp
    if (execution.createdAt) {
      details["Checked at"] = new Date(execution.createdAt).toLocaleString();
    }

    const payload = getFirstPayload(execution);
    if (!payload || !payload.data) {
      details["Issues"] = [];
      return details;
    }

    const responseData = payload.data as PrometheusResponse | undefined;
    if (!responseData || !responseData.data || !responseData.data.result) {
      details["Issues"] = [];
      return details;
    }

    const results = responseData.data.result;

    // Parse issues from Prometheus response
    const issues = results.map((result) => {
      const metric = result.metric || {};
      const value = result.value;

      // Extract status from value: [timestamp, "status"]
      let status: "degraded" | "critical" = "degraded";
      if (value && Array.isArray(value) && value.length >= 2) {
        const statusValue = String(value[1]);
        status = statusValue === "2" ? "critical" : "degraded";
      }

      // Extract check information from metric labels
      const checkName = metric["dash0_check_name"] || "Unknown Check";
      const checkSummary = metric["dash0_check_summary_template"] || "";
      const checkDescription = metric["dash0_check_description_template"] || "";

      return {
        status,
        checkName,
        checkSummary,
        checkDescription,
      };
    });

    // Add issues list with special type for custom rendering
    details["Issues"] = issues;

    return details;
  },
};

function metadataList(_node: ComponentsNode): MetadataItem[] {
  return [];
}

function getSpecs(configuration: ListIssuesConfiguration): ComponentBaseSpec[] | undefined {
  if (!configuration?.checkRules || configuration.checkRules.length === 0) {
    return undefined;
  }

  return [
    {
      title: "Check Rule",
      tooltipTitle: "Check Rules",
      values: configuration.checkRules.map((rule) => ({
        badges: [
          {
            label: rule,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    },
  ];
}

export const LIST_ISSUES_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  clear: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  degraded: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-yellow-100",
    badgeColor: "bg-yellow-500",
  },
  critical: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export const listIssuesStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  // Handle error states
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
  ) {
    return "error";
  }

  // Handle cancelled state
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  // Handle running state
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  // Only analyze issue status for finished, successful executions
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    // First, try to determine state from the active output channel
    // This is the preferred method as the backend now routes to the appropriate channel
    const activeChannel = getActiveChannel(execution);

    if (activeChannel === CHANNEL_CRITICAL) {
      return "critical";
    }
    if (activeChannel === CHANNEL_DEGRADED) {
      return "degraded";
    }
    if (activeChannel === CHANNEL_CLEAR) {
      return "clear";
    }

    // Fallback for legacy executions using 'default' channel:
    // Analyze the data to determine state
    if (activeChannel === "default") {
      const payload = getFirstPayload(execution);
      if (!payload || !payload.data) {
        return "clear";
      }

      const responseData = payload.data as PrometheusResponse | undefined;
      if (!responseData || !responseData.data || !responseData.data.result) {
        return "clear";
      }

      const results = responseData.data.result;

      // No issues found
      if (results.length === 0) {
        return "clear";
      }

      // Analyze issue statuses
      let hasCritical = false;
      let hasDegraded = false;

      for (const result of results) {
        // For instant queries, check the value field: [timestamp, "status"]
        if (result.value && Array.isArray(result.value) && result.value.length >= 2) {
          const status = String(result.value[1]);
          if (status === "2") {
            hasCritical = true;
          } else if (status === "1") {
            hasDegraded = true;
          }
        }
      }

      // Return critical if there's at least one critical issue
      if (hasCritical) {
        return "critical";
      }

      // Return degraded if there are only degraded issues
      if (hasDegraded) {
        return "degraded";
      }

      // Default to clear if we can't determine status
      return "clear";
    }

    // No output found - default to clear
    return "clear";
  }

  return "failed";
};

export const LIST_ISSUES_STATE_REGISTRY: EventStateRegistry = {
  stateMap: LIST_ISSUES_STATE_MAP,
  getState: listIssuesStateFunction,
};

function getIssueCounts(execution: WorkflowsWorkflowNodeExecution): { critical: number; degraded: number } {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) {
    return { critical: 0, degraded: 0 };
  }

  const responseData = payload.data as PrometheusResponse | undefined;
  if (!responseData || !responseData.data || !responseData.data.result) {
    return { critical: 0, degraded: 0 };
  }

  const results = responseData.data.result;

  let critical = 0;
  let degraded = 0;

  for (const result of results) {
    if (result.value && Array.isArray(result.value) && result.value.length >= 2) {
      const status = String(result.value[1]);
      if (status === "2") {
        critical++;
      } else if (status === "1") {
        degraded++;
      }
    }
  }

  return { critical, degraded };
}

function baseEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const { critical, degraded } = getIssueCounts(execution);
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

  // Build subtitle with counts and time
  const countParts: string[] = [];
  if (critical > 0) {
    countParts.push(`${critical} critical`);
  }
  if (degraded > 0) {
    countParts.push(`${degraded} degraded`);
  }

  let eventSubtitle: string;
  if (countParts.length > 0) {
    eventSubtitle = `${countParts.join(", ")} · ${timeAgo}`;
  } else {
    // No issues found - show "no issues" with time
    eventSubtitle = `no issues · ${timeAgo}`;
  }

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id,
    },
  ];
}
