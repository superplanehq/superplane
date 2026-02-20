import {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { formatTimeAgo } from "@/utils/date";
import { CanvasesCanvasNodeExecution } from "@/api-client";

interface WaitForButtonClickConfiguration {
  channel?: string;
  message?: string;
  timeout?: number;
  buttons?: Array<{ name: string; value: string }>;
}

interface WaitForButtonClickMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
  messageTS?: string;
  selectedButton?: string;
}

// State map for wait for button click - includes "waiting" state
const WAIT_FOR_BUTTON_CLICK_STATE_MAP: EventStateMap = {
  finished: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "ban",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-400",
  },
};

// State function to determine the state of an execution
const waitForButtonClickStateFunction: StateFunction = (execution: CanvasesCanvasNodeExecution): EventState => {
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED") {
    return "failed";
  }

  // Use "waiting" state when pending or started (not "running")
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "waiting";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "finished";
  }

  return "failed";
};

export const WAIT_FOR_BUTTON_CLICK_STATE_REGISTRY: EventStateRegistry = {
  stateMap: WAIT_FOR_BUTTON_CLICK_STATE_MAP,
  getState: waitForButtonClickStateFunction,
};

export const waitForButtonClickMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? waitForButtonClickEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: waitForButtonClickMetadataList(context.node),
      specs: waitForButtonClickSpecs(context.node),
      eventStateMap: WAIT_FOR_BUTTON_CLICK_STATE_MAP,
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { received?: OutputPayload[]; timeout?: OutputPayload[] } | undefined;
    const metadata = context.execution.metadata as WaitForButtonClickMetadata | undefined;

    // Get data from the received output channel
    const receivedData = outputs?.received?.[0]?.data as Record<string, unknown> | undefined;

    const details: Record<string, string> = {};

    // Add "Sent at" timestamp from execution creation
    if (context.execution.createdAt) {
      details["Sent at"] = new Date(context.execution.createdAt).toLocaleString();
    }

    // Add "Clicked at" if button was clicked
    if (receivedData?.clicked_at) {
      details["Clicked at"] = formatTimestamp(receivedData.clicked_at);
    }

    if ((receivedData?.clicked_by as Record<string, unknown>)?.username) {
      details["Clicked by"] = (receivedData?.clicked_by as Record<string, unknown>)?.username as string;
    }

    // Add selected button value if available
    if (metadata?.selectedButton) {
      details["Selected Button"] = metadata.selectedButton;
    }

    // Add timeout information if execution timed out
    const timeoutData = outputs?.timeout?.[0]?.data as Record<string, unknown> | undefined;
    if (timeoutData?.timeout_at) {
      details["Timed out at"] = formatTimestamp(timeoutData.timeout_at);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function waitForButtonClickMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as WaitForButtonClickMetadata | undefined;
  const configuration = node.configuration as WaitForButtonClickConfiguration | undefined;

  // Show channel in metadata like slackSendMessage does
  const channelLabel = nodeMetadata?.channel?.name || configuration?.channel;
  if (channelLabel) {
    metadata.push({ icon: "hash", label: channelLabel });
  }

  return metadata;
}

function waitForButtonClickSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as WaitForButtonClickConfiguration | undefined;

  if (configuration?.message) {
    specs.push({
      title: "message",
      tooltipTitle: "message",
      iconSlug: "message-square",
      value: configuration.message,
      contentType: "text",
    });
  }

  return specs;
}

function waitForButtonClickEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: waitForButtonClickStateFunction(execution as any),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function formatTimestamp(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  const asDate = new Date(String(value));
  if (!Number.isNaN(asDate.getTime())) {
    return asDate.toLocaleString();
  }

  return String(value);
}
