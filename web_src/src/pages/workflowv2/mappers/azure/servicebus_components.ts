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
import azureIcon from "@/assets/icons/integrations/azure.svg";
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import { stringOrDash } from "../utils";

// ── Shared configuration interfaces ────────────────────────────────────────

interface QueueComponentConfiguration {
  resourceGroup?: string;
  namespaceName?: string;
  queueName?: string;
}

interface TopicComponentConfiguration {
  resourceGroup?: string;
  namespaceName?: string;
  topicName?: string;
}

interface SendMessageConfiguration {
  namespaceName?: string;
  queueName?: string;
  body?: string;
  contentType?: string;
}

interface PublishMessageConfiguration {
  namespaceName?: string;
  topicName?: string;
  body?: string;
  contentType?: string;
}

// ── Helper: build event sections ────────────────────────────────────────────

function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
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

function subtitleFromExecution(context: SubtitleContext): string {
  if (!context.execution.createdAt) return "";
  return formatTimeAgo(new Date(context.execution.createdAt));
}

// ── Queue actions ────────────────────────────────────────────────────────────

function queueMetadata(node: NodeInfo): MetadataItem[] {
  const cfg = node.configuration as QueueComponentConfiguration | undefined;
  if (cfg?.queueName) return [{ icon: "inbox", label: cfg.queueName }];
  return [];
}

function makeQueueMapper(componentName: string): ComponentBaseMapper {
  return {
    props(context: ComponentBaseContext): ComponentBaseProps {
      const lastExecution = context.lastExecutions[0] ?? null;
      return {
        title: context.node.name || context.componentDefinition.label || "Unnamed component",
        iconSrc: azureIcon,
        iconColor: getColorClass(context.componentDefinition.color),
        collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
        collapsed: context.node.isCollapsed,
        eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
        includeEmptyState: !lastExecution,
        metadata: queueMetadata(context.node),
        eventStateMap: getStateMap(componentName),
      };
    },
    subtitle: subtitleFromExecution,
    getExecutionDetails(_context: ExecutionDetailsContext) {
      return {};
    },
  };
}

// ── Topic actions ────────────────────────────────────────────────────────────

function topicMetadata(node: NodeInfo): MetadataItem[] {
  const cfg = node.configuration as TopicComponentConfiguration | undefined;
  if (cfg?.topicName) return [{ icon: "radio", label: cfg.topicName }];
  return [];
}

function makeTopicMapper(componentName: string): ComponentBaseMapper {
  return {
    props(context: ComponentBaseContext): ComponentBaseProps {
      const lastExecution = context.lastExecutions[0] ?? null;
      return {
        title: context.node.name || context.componentDefinition.label || "Unnamed component",
        iconSrc: azureIcon,
        iconColor: getColorClass(context.componentDefinition.color),
        collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
        collapsed: context.node.isCollapsed,
        eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
        includeEmptyState: !lastExecution,
        metadata: topicMetadata(context.node),
        eventStateMap: getStateMap(componentName),
      };
    },
    subtitle: subtitleFromExecution,
    getExecutionDetails(_context: ExecutionDetailsContext) {
      return {};
    },
  };
}

// ── Send Service Bus Message ─────────────────────────────────────────────────

interface SendMessageOutput {
  queue?: string;
  namespaceName?: string;
  sent?: boolean;
}

function sendMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const cfg = node.configuration as SendMessageConfiguration | undefined;
  if (!cfg?.body) return [];
  return [
    {
      title: "Message body",
      tooltipTitle: "Message body",
      iconSlug: "message-square",
      value: cfg.body,
      contentType: cfg.contentType === "application/json" ? "json" : "text",
    },
  ];
}

export const sendServiceBusMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions[0] ?? null;
    const componentName = context.componentDefinition.name;
    const cfg = context.node.configuration as SendMessageConfiguration | undefined;

    const metadata: MetadataItem[] = [];
    if (cfg?.queueName) metadata.push({ icon: "inbox", label: cfg.queueName });

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: azureIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata,
      specs: sendMessageSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle: subtitleFromExecution,
  getExecutionDetails(context: ExecutionDetailsContext) {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as SendMessageOutput | undefined;
    if (!result) return {};
    return {
      Queue: stringOrDash(result.queue),
      Namespace: stringOrDash(result.namespaceName),
      Sent: result.sent ? "Yes" : "No",
    };
  },
};

// ── Publish Service Bus Message ──────────────────────────────────────────────

interface PublishMessageOutput {
  topic?: string;
  namespaceName?: string;
  published?: boolean;
}

function publishMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const cfg = node.configuration as PublishMessageConfiguration | undefined;
  if (!cfg?.body) return [];
  return [
    {
      title: "Message body",
      tooltipTitle: "Message body",
      iconSlug: "message-square",
      value: cfg.body,
      contentType: cfg.contentType === "application/json" ? "json" : "text",
    },
  ];
}

export const publishServiceBusMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions[0] ?? null;
    const componentName = context.componentDefinition.name;
    const cfg = context.node.configuration as PublishMessageConfiguration | undefined;

    const metadata: MetadataItem[] = [];
    if (cfg?.topicName) metadata.push({ icon: "radio", label: cfg.topicName });

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: azureIcon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata,
      specs: publishMessageSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle: subtitleFromExecution,
  getExecutionDetails(context: ExecutionDetailsContext) {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as PublishMessageOutput | undefined;
    if (!result) return {};
    return {
      Topic: stringOrDash(result.topic),
      Namespace: stringOrDash(result.namespaceName),
      Published: result.published ? "Yes" : "No",
    };
  },
};

// ── Exported mappers ─────────────────────────────────────────────────────────

export const createServiceBusQueueMapper = makeQueueMapper("azure.createServiceBusQueue");
export const deleteServiceBusQueueMapper = makeQueueMapper("azure.deleteServiceBusQueue");
export const getServiceBusQueueMapper = makeQueueMapper("azure.getServiceBusQueue");

export const createServiceBusTopicMapper = makeTopicMapper("azure.createServiceBusTopic");
export const deleteServiceBusTopicMapper = makeTopicMapper("azure.deleteServiceBusTopic");
export const getServiceBusTopicMapper = makeTopicMapper("azure.getServiceBusTopic");
