import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { formatTimeAgo } from "@/utils/date";

interface SendAndWaitConfiguration {
  channel?: string;
  message?: string;
  timeout?: number;
  buttons?: Array<{ name: string; value: string }>;
}

interface SendAndWaitMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
  state?: string;
  messageTs?: string;
}

export const sendAndWaitMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

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
      eventSections: lastExecution
        ? sendAndWaitEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendAndWaitMetadataList(context.node),
      specs: sendAndWaitSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const metadata = context.execution.metadata as SendAndWaitMetadata | undefined;
    const outputs = context.execution.outputs as { received?: OutputPayload[] } | undefined;
    const response = outputs?.received?.[0]?.data as Record<string, unknown> | undefined;

    return {
      Status: metadata?.state || "-",
      "Button Value": response?.value ? String(response.value) : "-",
      User: response?.user
        ? String((response.user as Record<string, unknown>)?.username || "-")
        : "-",
      Channel: metadata?.channel?.name || "-",
    };
  },

  subtitle(context: SubtitleContext): string {
    const metadata = context.execution.metadata as SendAndWaitMetadata | undefined;
    const state = metadata?.state;

    if (state === "waiting") {
      return "Waiting for response...";
    }

    if (state === "received") {
      return "Response received";
    }

    if (state === "timed_out") {
      return "Timed out";
    }

    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function sendAndWaitMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendAndWaitMetadata | undefined;
  const configuration = node.configuration as SendAndWaitConfiguration | undefined;

  const channelLabel = nodeMetadata?.channel?.name || configuration?.channel;
  if (channelLabel) {
    metadata.push({ icon: "hash", label: channelLabel });
  }

  const buttonCount = configuration?.buttons?.length;
  if (buttonCount) {
    metadata.push({ icon: "mouse-pointer-click", label: `${buttonCount} button${buttonCount > 1 ? "s" : ""}` });
  }

  return metadata;
}

function sendAndWaitSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendAndWaitConfiguration | undefined;

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

function sendAndWaitEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

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
