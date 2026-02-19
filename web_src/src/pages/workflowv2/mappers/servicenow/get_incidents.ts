import {
  ComponentBaseProps,
  ComponentBaseSpec,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseMapper,
  OutputPayload,
  EventStateRegistry,
  StateFunction,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  NodeInfo,
  ExecutionInfo,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import snIcon from "@/assets/icons/integrations/servicenow.svg";
import { formatTimeAgo } from "@/utils/date";
import {
  BaseNodeMetadata,
  GetIncidentsConfiguration,
  IncidentRecord,
  STATE_LABELS,
  URGENCY_LABELS,
  IMPACT_LABELS,
} from "./types";

// Output channel names matching the backend constants
const CHANNEL_CLEAR = "clear";
const CHANNEL_LOW = "low";
const CHANNEL_HIGH = "high";

type GetIncidentsOutputs = {
  default?: OutputPayload[];
  clear?: OutputPayload[];
  low?: OutputPayload[];
  high?: OutputPayload[];
};

type GetIncidentsResponse = {
  incidents?: IncidentRecord[];
  total?: number;
};

function getFirstPayload(execution: ExecutionInfo): OutputPayload | null {
  const outputs = execution.outputs as unknown as GetIncidentsOutputs | undefined;
  if (!outputs) return null;

  for (const channel of [CHANNEL_HIGH, CHANNEL_LOW, CHANNEL_CLEAR]) {
    const channelOutputs = outputs[channel as keyof GetIncidentsOutputs];
    if (channelOutputs && channelOutputs.length > 0) {
      return channelOutputs[0];
    }
  }

  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

function getActiveChannel(execution: ExecutionInfo): string | null {
  const outputs = execution.outputs as unknown as GetIncidentsOutputs | undefined;
  if (!outputs) return null;

  if (outputs.high && outputs.high.length > 0) return CHANNEL_HIGH;
  if (outputs.low && outputs.low.length > 0) return CHANNEL_LOW;
  if (outputs.clear && outputs.clear.length > 0) return CHANNEL_CLEAR;
  if (outputs.default && outputs.default.length > 0) return "default";

  return null;
}

function getIncidents(execution: ExecutionInfo): IncidentRecord[] {
  const payload = getFirstPayload(execution);
  if (!payload || !payload.data) return [];

  const responseData = payload.data as GetIncidentsResponse | undefined;
  if (!responseData || !responseData.incidents) return [];

  return responseData.incidents;
}

export const getIncidentsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "servicenow";
    const configuration = context.node.configuration as unknown as GetIncidentsConfiguration;
    const specs = getSpecs(configuration, context.node);

    return {
      iconSrc: snIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      specs,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    const timeAgo = formatTimeAgo(new Date(context.execution.createdAt!));
    const incidents = getIncidents(context.execution);

    if (incidents.length > 0) {
      return `${incidents.length} incidents · ${timeAgo}`;
    }

    return `no incidents · ${timeAgo}`;
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};

    if (context.execution.createdAt) {
      details["Checked at"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const incidents = getIncidents(context.execution);

    if (incidents.length > 0) {
      const incidentDetails = incidents.map((incident) => ({
        number: incident.number || "",
        short_description: incident.short_description || "No description",
        state: incident.state ? STATE_LABELS[incident.state] || incident.state : "",
        urgency: incident.urgency ? URGENCY_LABELS[incident.urgency] || incident.urgency : "",
        impact: incident.impact ? IMPACT_LABELS[incident.impact] || incident.impact : "",
        sys_id: incident.sys_id,
        sys_created_on: incident.sys_created_on,
      }));

      details["Incidents"] = incidentDetails;
    }

    if (
      context.execution.resultMessage &&
      (context.execution.resultReason === "RESULT_REASON_ERROR" ||
        (context.execution.result === "RESULT_FAILED" &&
          context.execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
    ) {
      details["Error"] = {
        __type: "error",
        message: context.execution.resultMessage,
      };
    }

    return details;
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as BaseNodeMetadata | undefined;

  if (nodeMetadata?.instanceUrl) {
    const instanceName = nodeMetadata.instanceUrl.replace(/^https?:\/\//, "").replace(/\.service-now\.com$/, "");
    metadata.push({ icon: "globe", label: instanceName });
  }

  return metadata;
}

function getSpecs(configuration: GetIncidentsConfiguration, node: NodeInfo): ComponentBaseSpec[] | undefined {
  const specs: ComponentBaseSpec[] = [];
  const nodeMetadata = node.metadata as BaseNodeMetadata | undefined;

  if (configuration?.assignmentGroup) {
    const groupLabel = nodeMetadata?.assignmentGroup?.name ?? configuration.assignmentGroup;
    specs.push({
      title: "Group",
      tooltipTitle: "Assignment Group",
      values: [{ badges: [{ label: groupLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" }] }],
    });
  }

  if (configuration?.state) {
    const stateLabel = STATE_LABELS[configuration.state] ?? configuration.state;
    specs.push({
      title: "State",
      tooltipTitle: "State Filter",
      values: [{ badges: [{ label: stateLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" }] }],
    });
  }

  if (configuration?.urgency) {
    const urgencyLabel = URGENCY_LABELS[configuration.urgency] ?? configuration.urgency;
    specs.push({
      title: "Urgency",
      tooltipTitle: "Urgency Filter",
      values: [{ badges: [{ label: urgencyLabel, bgColor: "bg-gray-100", textColor: "text-gray-700" }] }],
    });
  }

  return specs.length > 0 ? specs : undefined;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  const incidents = getIncidents(execution);
  const timeAgo = formatTimeAgo(new Date(execution.createdAt!));
  const eventSubtitle =
    incidents.length > 0 ? `${incidents.length} incidents · ${timeAgo}` : `no incidents · ${timeAgo}`;

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

export const GET_INCIDENTS_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  clear: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  low: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-yellow-100",
    badgeColor: "bg-yellow-500",
  },
  high: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
};

export const getIncidentsStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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

    if (activeChannel === CHANNEL_HIGH) return "high";
    if (activeChannel === CHANNEL_LOW) return "low";
    if (activeChannel === CHANNEL_CLEAR) return "clear";

    // Fallback: analyze incidents from data
    const incidents = getIncidents(execution);
    if (incidents.length > 0) {
      const hasHigh = incidents.some((i) => i.urgency === "1");
      return hasHigh ? "high" : "low";
    }

    return "clear";
  }

  return "failed";
};

export const GET_INCIDENTS_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GET_INCIDENTS_STATE_MAP,
  getState: getIncidentsStateFunction,
};
