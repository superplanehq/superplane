import { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { MetadataItem } from "@/ui/metadataList";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { formatTimeAgo } from "@/utils/date";

interface SendTextMessageConfiguration {
  channel?: string;
  text?: string;
}

interface SendTextMessageMetadata {
  channel?: {
    id?: string;
    name?: string;
  };
}

export const sendTextMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
      iconSrc: slackIcon,
      iconSlug: "slack",
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
      "Sent At": formatSlackTimestamp(message?.ts || message?.event_ts) || "-",
      Channel: stringOrDash(message?.channel),
      User: stringOrDash(message?.user),
      Text: stringOrDash(message?.text),
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
  const configuration = node.configuration as SendTextMessageConfiguration | undefined;

  const channelLabel = nodeMetadata?.channel?.name || configuration?.channel;
  if (channelLabel) {
    metadata.push({ icon: "hash", label: channelLabel });
  }

  return metadata;
}

function sendTextMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendTextMessageConfiguration | undefined;

  if (configuration?.text) {
    specs.push({
      title: "text",
      tooltipTitle: "text",
      iconSlug: "message-square",
      value: configuration.text,
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
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}

function formatSlackTimestamp(value?: unknown): string | undefined {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }

  const raw = String(value);
  const seconds = Number.parseFloat(raw);
  if (!Number.isNaN(seconds)) {
    return new Date(seconds * 1000).toLocaleString();
  }

  const asDate = new Date(raw);
  if (!Number.isNaN(asDate.getTime())) {
    return asDate.toLocaleString();
  }

  return raw;
}
