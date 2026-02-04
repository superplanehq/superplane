import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import discordIcon from "@/assets/icons/integrations/discord.svg";

interface SendTextMessageConfiguration {
  channel?: string;
  content?: string;
  embedTitle?: string;
  embedDescription?: string;
  embedColor?: string;
  embedUrl?: string;
}

interface ChannelMetadata {
  id?: string;
  name?: string;
}

interface SendTextMessageMetadata {
  hasEmbed?: boolean;
  channel?: ChannelMetadata;
}

export const sendTextMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
      iconSrc: discordIcon,
      iconSlug: "discord",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? sendTextMessageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendTextMessageMetadataList(context.node),
      specs: sendTextMessageSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const message = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;

    return {
      "Message ID": stringOrDash(message?.id),
      "Channel ID": stringOrDash(message?.channel_id),
      Content: stringOrDash(message?.content),
      Timestamp: stringOrDash(message?.timestamp),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function sendTextMessageMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendTextMessageMetadata | undefined;

  if (nodeMetadata?.channel?.name) {
    metadata.push({ icon: "hash", label: nodeMetadata.channel.name });
  }

  if (nodeMetadata?.hasEmbed) {
    metadata.push({ icon: "square", label: "With Embed" });
  }

  return metadata;
}

function sendTextMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendTextMessageConfiguration | undefined;

  if (configuration?.content) {
    specs.push({
      title: "content",
      tooltipTitle: "content",
      iconSlug: "message-square",
      value: configuration.content,
      contentType: "text",
    });
  }

  if (configuration?.embedTitle) {
    specs.push({
      title: "embed title",
      tooltipTitle: "embed title",
      iconSlug: "square",
      value: configuration.embedTitle,
      contentType: "text",
    });
  }

  if (configuration?.embedDescription) {
    specs.push({
      title: "embed description",
      tooltipTitle: "embed description",
      iconSlug: "align-left",
      value: configuration.embedDescription,
      contentType: "text",
    });
  }

  return specs;
}

function sendTextMessageEventSections(
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
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
