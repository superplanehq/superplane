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
import { MetadataItem } from "@/ui/metadataList";
import slackIcon from "@/assets/icons/integrations/slack.svg";

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
      iconSrc: slackIcon,
      iconSlug: "slack",
      iconBackground: "bg-white",
      iconColor: getColorClass(componentDefinition.color),
      headerColor: "bg-transparent",
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
      Channel: stringOrDash(message?.channel),
      User: stringOrDash(message?.user),
      Timestamp: stringOrDash(message?.ts || message?.event_ts),
      Text: stringOrDash(message?.text),
    };
  },
};

function sendTextMessageMetadataList(node: ComponentsNode): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendTextMessageMetadata | undefined;
  const configuration = node.configuration as SendTextMessageConfiguration | undefined;

  const channelLabel = nodeMetadata?.channel?.name || configuration?.channel;
  if (channelLabel) {
    metadata.push({ icon: "hash", label: channelLabel });
  }

  return metadata;
}

function sendTextMessageSpecs(node: ComponentsNode): ComponentBaseSpec[] {
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
