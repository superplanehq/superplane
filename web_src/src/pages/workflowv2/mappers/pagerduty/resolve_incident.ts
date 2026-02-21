import {
  ComponentBaseProps,
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
import pdIcon from "@/assets/icons/integrations/pagerduty.svg";
import { buildIncidentExecutionDetails } from "./base";
import { formatTimeAgo } from "@/utils/date";

const CHANNEL_SUCCESS = "success";
const CHANNEL_FAILED = "failed";

type ResolveIncidentOutputs = {
  default?: OutputPayload[];
  success?: OutputPayload[];
  failed?: OutputPayload[];
};

function getActiveChannel(execution: ExecutionInfo): string | null {
  const outputs = execution.outputs as unknown as ResolveIncidentOutputs | undefined;
  if (!outputs) return null;

  if (outputs.success && outputs.success.length > 0) return CHANNEL_SUCCESS;
  if (outputs.failed && outputs.failed.length > 0) return CHANNEL_FAILED;
  if (outputs.default && outputs.default.length > 0) return "default";

  return null;
}

export const RESOLVE_INCIDENT_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  resolved: DEFAULT_EVENT_STATE_MAP.success,
};

export const resolveIncidentStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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
    if (activeChannel === CHANNEL_FAILED) {
      return "failed";
    }

    return "resolved";
  }

  return "failed";
};

export const RESOLVE_INCIDENT_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RESOLVE_INCIDENT_STATE_MAP,
  getState: resolveIncidentStateFunction,
};

export const resolveIncidentMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "pagerduty";

    return {
      iconSrc: pdIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    return buildIncidentExecutionDetails(context.execution);
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;

  if (configuration.incidentId) {
    metadata.push({ icon: "alert-triangle", label: `Incident: ${configuration.incidentId}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
