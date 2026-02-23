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
import { formatTimeAgo } from "@/utils/date";
import { MetadataItem } from "@/ui/metadataList";
import telegramIcon from "@/assets/icons/integrations/telegram.svg";

interface SendMessageConfiguration {
  chatId?: string;
  text?: string;
  parseMode?: string;
}

interface ChatMetadata {
  id?: string;
  name?: string;
}

interface SendMessageMetadata {
  chat?: ChatMetadata;
}

export const sendMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      iconSrc: telegramIcon,
      iconSlug: "telegram",
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution ? sendMessageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      metadata: sendMessageMetadataList(context.node),
      specs: sendMessageSpecs(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const message = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;

    return {
      "Message ID": stringOrDash(message?.message_id),
      "Chat ID": stringOrDash(message?.chat_id),
      Text: stringOrDash(message?.text),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },
};

function sendMessageMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as SendMessageMetadata | undefined;
  const configuration = node.configuration as SendMessageConfiguration | undefined;

  const chatLabel = nodeMetadata?.chat?.name || configuration?.chatId;
  if (chatLabel) {
    metadata.push({ icon: "message-circle", label: chatLabel });
  }

  if (configuration?.parseMode) {
    metadata.push({ icon: "code", label: configuration.parseMode });
  }

  return metadata;
}

function sendMessageSpecs(node: NodeInfo): ComponentBaseSpec[] {
  const specs: ComponentBaseSpec[] = [];
  const configuration = node.configuration as SendMessageConfiguration | undefined;

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

function sendMessageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.rootEvent) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}

function stringOrDash(value?: unknown): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  return String(value);
}
