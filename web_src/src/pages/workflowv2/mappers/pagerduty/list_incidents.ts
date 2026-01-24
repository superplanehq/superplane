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
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentBaseMapper, OutputPayload, EventStateRegistry, StateFunction } from "../types";
import { MetadataItem } from "@/ui/metadataList";
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { Incident, ListIncidentsConfiguration, ListIncidentsResponse } from "./types";
import { formatTimeAgo } from "@/utils/date";

// Output channel names matching the backend constants
const CHANNEL_CLEAR = "clear";
const CHANNEL_LOW = "low";
const CHANNEL_HIGH = "high";

// Type for outputs with channel structure
type ListIncidentsOutputs = {
  default?: OutputPayload[];
  clear?: OutputPayload[];
  low?: OutputPayload[];
  high?: OutputPayload[];
};

/**
 * Extracts the first payload from execution outputs, checking all possible channels.
 */
function getFirstPayload(execution: WorkflowsWorkflowNodeExecution): OutputPayload | null {
  const outputs = execution.outputs as ListIncidentsOutputs | undefined;
  if (!outputs) return null;

  // Check channel-based outputs first (in severity order)
  for (const channel of [CHANNEL_HIGH, CHANNEL_LOW, CHANNEL_CLEAR]) {
    const channelOutputs = outputs[channel as keyof ListIncidentsOutputs];
    if (channelOutputs && channelOutputs.length > 0) {
      return channelOutputs[0];
    }
  }

  // Fallback to default channel
  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

/**
 * Determines which output channel has data, indicating the incident urgency state.
 */
function getActiveChannel(execution: WorkflowsWorkflowNodeExecution): string | null {
  const outputs = execution.outputs as ListIncidentsOutputs | undefined;
  if (!outputs) return null;

  if (outputs.high && outputs.high.length > 0) return CHANNEL_HIGH;
  if (outputs.low && outputs.low.length > 0) return CHANNEL_LOW;
  if (outputs.clear && outputs.clear.length > 0) return CHANNEL_CLEAR;
  if (outputs.default && outputs.default.length > 0) return "default";

  return null;
}

/**
 * Extracts incidents from the execution payload.
 */
function getIncidents(execution: WorkflowsWorkflowNodeExecution): Incident[] {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) return [];

  const responseData = payload.data as ListIncidentsResponse | undefined;
  if (!responseData || !responseData.incidents) return [];

  return responseData.incidents;
}

export const listIncidentsMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name!;

    const configuration = node.configuration as unknown as ListIncidentsConfiguration;
    const specs = getSpecs(configuration);

    return {
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name!,
      eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(node),
      specs,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    const timeAgo = formatTimeAgo(new Date(execution.createdAt!));
    const incidents = getIncidents(execution);

    if (incidents.length > 0) {
      const highCount = incidents.filter((i) => i.urgency === "high").length;
      const lowCount = incidents.filter((i) => i.urgency === "low").length;

      const countParts: string[] = [];
      if (highCount > 0) {
        countParts.push(`${highCount} high urgency`);
      }
      if (lowCount > 0) {
        countParts.push(`${lowCount} low urgency`);
      }

      if (countParts.length > 0) {
        return `${countParts.join(", ")} · ${timeAgo}`;
      }

      return `${incidents.length} incidents · ${timeAgo}`;
    }

    return `no incidents · ${timeAgo}`;
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};

    // Add "Checked at" timestamp
    if (execution.createdAt) {
      details["Checked at"] = new Date(execution.createdAt).toLocaleString();
    }

    const incidents = getIncidents(execution);

    if (incidents.length === 0) {
      details["Incidents"] = [];
      return details;
    }

    // Parse incidents for display - format matches PagerDutyIncidentEntry type in ChainItem
    const incidentDetails = incidents.map((incident) => ({
      id: incident.id || "",
      title: incident.title || "Untitled Incident",
      status: incident.status || "triggered",
      urgency: incident.urgency || "low",
      service: incident.service?.summary,
      priority: incident.priority?.summary,
      html_url: incident.html_url,
      created_at: incident.created_at,
    }));

    details["Incidents"] = incidentDetails;

    return details;
  },
};

function metadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as { services?: { id: string; name: string }[] } | undefined;

  if (nodeMetadata?.services && nodeMetadata.services.length > 0) {
    const serviceNames = nodeMetadata.services.map((s) => s.name).join(", ");
    metadata.push({ icon: "bell", label: serviceNames });
  }

  return metadata;
}

function getSpecs(configuration: ListIncidentsConfiguration): ComponentBaseSpec[] | undefined {
  if (!configuration?.services || configuration.services.length === 0) {
    return undefined;
  }

  return [
    {
      title: "Service",
      tooltipTitle: "Services",
      values: configuration.services.map((service) => ({
        badges: [
          {
            label: service,
            bgColor: "bg-gray-100",
            textColor: "text-gray-700",
          },
        ],
      })),
    },
  ];
}

export const LIST_INCIDENTS_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  clear: {
    icon: "circle-check",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-gray-100 dark:bg-gray-700",
    badgeColor: "bg-gray-500",
  },
  low: {
    icon: "alert-triangle",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-yellow-100 dark:bg-yellow-900/50",
    badgeColor: "bg-yellow-500",
  },
  high: {
    icon: "circle-x",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-500",
  },
};

export const listIncidentsStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
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

  // Only analyze incident urgency for finished, successful executions
  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const activeChannel = getActiveChannel(execution);

    if (activeChannel === CHANNEL_HIGH) {
      return "high";
    }
    if (activeChannel === CHANNEL_LOW) {
      return "low";
    }
    if (activeChannel === CHANNEL_CLEAR) {
      return "clear";
    }

    // Fallback: analyze incidents from data
    const incidents = getIncidents(execution);
    if (incidents.length > 0) {
      const hasHigh = incidents.some((i) => i.urgency === "high");
      return hasHigh ? "high" : "low";
    }

    return "clear";
  }

  return "failed";
};

export const LIST_INCIDENTS_STATE_REGISTRY: EventStateRegistry = {
  stateMap: LIST_INCIDENTS_STATE_MAP,
  getState: listIncidentsStateFunction,
};

function baseEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const incidents = getIncidents(execution);
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));

  let eventSubtitle: string;
  if (incidents.length > 0) {
    const highCount = incidents.filter((i) => i.urgency === "high").length;
    const lowCount = incidents.filter((i) => i.urgency === "low").length;

    const countParts: string[] = [];
    if (highCount > 0) {
      countParts.push(`${highCount} high urgency`);
    }
    if (lowCount > 0) {
      countParts.push(`${lowCount} low urgency`);
    }

    if (countParts.length > 0) {
      eventSubtitle = `${countParts.join(", ")} · ${timeAgo}`;
    } else {
      eventSubtitle = `${incidents.length} incidents · ${timeAgo}`;
    }
  } else {
    eventSubtitle = `no incidents · ${timeAgo}`;
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
