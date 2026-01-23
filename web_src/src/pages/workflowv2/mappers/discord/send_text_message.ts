import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, OutputPayload } from "../types";
import { ComponentBaseProps, ComponentBaseSpec, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";

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
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _items?: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name!;

    return {
      title: node.name!,
      iconSlug: "discord",
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      eventSections: lastExecution ? sendTextMessageEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendTextMessageMetadataList(node),
      specs: sendTextMessageSpecs(node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, _node: ComponentsNode): Record<string, string> {
    const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
    const message = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;

    return {
      "Message ID": stringOrDash(message?.id),
      "Channel ID": stringOrDash(message?.channel_id),
      Content: stringOrDash(message?.content),
      Timestamp: stringOrDash(message?.timestamp),
    };
  },

  subtitle(_node: ComponentsNode, execution: WorkflowsWorkflowNodeExecution): string {
    if (!execution.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
};

function sendTextMessageMetadataList(node: ComponentsNode): MetadataItem[] {
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

function sendTextMessageSpecs(node: ComponentsNode): ComponentBaseSpec[] {
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
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id,
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
