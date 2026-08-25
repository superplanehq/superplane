import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getState, getStateMap, getTriggerRenderer } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import groqIcon from "@/assets/icons/integrations/groq.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type ChatCompletionConfiguration = {
  model?: string;
};

type ChatCompletionPayload = {
  model?: string;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
  };
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as ChatCompletionConfiguration | undefined;
  if (!configuration?.model) return [];

  return [{ icon: "sparkles", label: configuration.model }];
}

function formatTokens(usage?: ChatCompletionPayload["usage"]): string | undefined {
  if (!usage?.total_tokens) return undefined;
  const input = usage.prompt_tokens ?? 0;
  const output = usage.completion_tokens ?? 0;
  return `${usage.total_tokens.toLocaleString()} (${input.toLocaleString()} in / ${output.toLocaleString()} out)`;
}

function formatTimestamp(timestamp?: string): string | undefined {
  if (!timestamp) return undefined;

  const date = new Date(timestamp);
  return Number.isNaN(date.getTime()) ? undefined : date.toLocaleString();
}

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "groq";

    return {
      iconSrc: groqIcon,
      iconSlug: context.componentDefinition?.icon ?? "message-square",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "Groq",
      metadata: metadataList(context.node),
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as ChatCompletionPayload | undefined;
    const details: Record<string, string> = {};

    const startedAt = formatTimestamp(context.execution.createdAt);
    if (startedAt) {
      details["Started At"] = startedAt;
    }

    const completedAt = formatTimestamp(payload?.timestamp);
    if (completedAt) {
      details["Completed At"] = completedAt;
    }
    if (data?.model) {
      details["Model"] = data.model;
    }
    const tokens = formatTokens(data?.usage);
    if (tokens) {
      details["Tokens"] = tokens;
    }
    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  const createdAt = formatDate(execution.createdAt);
  const receivedAt = createdAt ?? new Date();
  const eventSubtitle = createdAt ? renderTimeAgo(createdAt) : "";
  const eventState = getState(componentName)(execution);

  if (!rootEvent?.id) {
    return [
      {
        receivedAt,
        eventTitle: "Execution",
        eventSubtitle,
        eventState,
        eventId: execution.id ?? "",
      },
    ];
  }

  const rootTriggerNode = nodes.find((node) => node.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt,
      eventTitle: title,
      eventSubtitle,
      eventState,
      eventId: rootEvent.id,
    },
  ];
}

function formatDate(timestamp?: string): Date | undefined {
  if (!timestamp) return undefined;

  const date = new Date(timestamp);
  return Number.isNaN(date.getTime()) ? undefined : date;
}
