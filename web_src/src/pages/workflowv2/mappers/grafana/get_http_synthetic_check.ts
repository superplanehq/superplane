import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ComponentBaseContext,
  ExecutionInfo,
  EventStateRegistry,
  StateFunction,
  NodeInfo,
  OutputPayload,
} from "../types";
import grafanaIcon from "@/assets/icons/integrations/grafana.svg";
import type { GetHttpSyntheticCheckConfiguration, SyntheticCheckNodeMetadata } from "./types";
import { buildGetSyntheticCheckDetails, buildSyntheticCheckSelectionMetadata } from "./synthetic_check_shared";
import { grafanaCreatedAtSubtitle } from "./base";
import { renderTimeAgo, renderWithTimeAgo } from "@/components/TimeAgo";

const CHANNEL_UP = "up";
const CHANNEL_PARTIAL = "partial";
const CHANNEL_DOWN = "down";

type GetHttpSyntheticCheckOutputs = {
  default?: OutputPayload[];
  up?: OutputPayload[];
  partial?: OutputPayload[];
  down?: OutputPayload[];
};

export const getHttpSyntheticCheckMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";
    const configuration = context.node.configuration as GetHttpSyntheticCheckConfiguration | undefined;
    const nodeMetadata = context.node.metadata as SyntheticCheckNodeMetadata | undefined;

    return {
      iconSrc: grafanaIcon,
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: buildSyntheticCheckSelectionMetadata(nodeMetadata, configuration?.syntheticCheck),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const payload = getFirstPayload(context.execution);
    return buildGetSyntheticCheckDetails(payload ?? undefined);
  },

  subtitle: grafanaCreatedAtSubtitle,
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.createdAt || !execution.rootEvent?.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const date = new Date(execution.createdAt);
  const activeChannel = getActiveChannel(execution);
  const statusLabel = activeChannel ? channelLabel(activeChannel) : null;
  const eventSubtitle = statusLabel ? renderWithTimeAgo(statusLabel, date) : renderTimeAgo(date);

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

function getFirstPayload(execution: ExecutionInfo): OutputPayload | null {
  const outputs = execution.outputs as GetHttpSyntheticCheckOutputs | undefined;
  if (!outputs) return null;

  for (const channel of [CHANNEL_DOWN, CHANNEL_PARTIAL, CHANNEL_UP]) {
    const channelOutputs = outputs[channel as keyof GetHttpSyntheticCheckOutputs];
    if (channelOutputs && channelOutputs.length > 0) {
      return channelOutputs[0];
    }
  }

  const emptyChannel = outputs["" as keyof GetHttpSyntheticCheckOutputs];
  if (emptyChannel && emptyChannel.length > 0) {
    return emptyChannel[0];
  }

  if (outputs.default && outputs.default.length > 0) {
    return outputs.default[0];
  }

  return null;
}

function getActiveChannel(execution: ExecutionInfo): string | null {
  const outputs = execution.outputs as GetHttpSyntheticCheckOutputs | undefined;
  if (!outputs) return null;

  if (outputs.down && outputs.down.length > 0) return CHANNEL_DOWN;
  if (outputs.partial && outputs.partial.length > 0) return CHANNEL_PARTIAL;
  if (outputs.up && outputs.up.length > 0) return CHANNEL_UP;
  if (outputs.default && outputs.default.length > 0) return "default";

  const emptyChannel = outputs["" as keyof GetHttpSyntheticCheckOutputs];
  if (emptyChannel && emptyChannel.length > 0) return "";

  return null;
}

function channelLabel(channel: string): string {
  switch (channel) {
    case CHANNEL_DOWN:
      return "down";
    case CHANNEL_PARTIAL:
      return "partial";
    case CHANNEL_UP:
      return "up";
    case "":
      return "no status";
    default:
      return "";
  }
}

export const GET_HTTP_SYNTHETIC_CHECK_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  up: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-green-500",
  },
  partial: {
    icon: "alert-triangle",
    textColor: "text-gray-800",
    backgroundColor: "bg-amber-100",
    badgeColor: "bg-amber-500",
  },
  down: {
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

function getFinishedSyntheticCheckState(activeChannel: string | null): EventState {
  const channelStateMap: Record<string, EventState> = {
    [CHANNEL_DOWN]: "down",
    [CHANNEL_PARTIAL]: "partial",
    [CHANNEL_UP]: "up",
    "": "noStatus",
  };

  return channelStateMap[activeChannel ?? ""] ?? "noStatus";
}

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
    return getFinishedSyntheticCheckState(getActiveChannel(execution));
  }

  return "failed";
};

export const GET_HTTP_SYNTHETIC_CHECK_STATE_REGISTRY: EventStateRegistry = {
  stateMap: GET_HTTP_SYNTHETIC_CHECK_STATE_MAP,
  getState: getHttpSyntheticCheckStateFunction,
};
