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
import openCodeGoIcon from "@/assets/icons/integrations/opencodego.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

type ChatCompletionNodeMetadata = {
  model?: string;
  structuredOutput?: boolean;
};

type ChatCompletionConfiguration = {
  model?: string;
  outputSchema?: string;
};

type ChatCompletionPayload = {
  model?: string;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
  };
};

// metadataList surfaces the configured model and enabled structured output.
// The model prefers backend node metadata (set in Setup) and falls back to the
// configuration so it shows before the first execution.
function metadataList(node: NodeInfo): MetadataItem[] {
  const meta = node.metadata as ChatCompletionNodeMetadata | undefined;
  const config = node.configuration as ChatCompletionConfiguration | undefined;

  const model = meta?.model || config?.model;
  const items: MetadataItem[] = model ? [{ icon: "sparkles", label: model }] : [];
  const structuredOutput = config ? Boolean(config.outputSchema?.trim()) : Boolean(meta?.structuredOutput);

  if (structuredOutput) {
    items.push({ icon: "braces", label: "Structured output" });
  }

  return items;
}

function formatTimestamp(timestamp?: string): string | undefined {
  return timestamp ? new Date(timestamp).toLocaleString() : undefined;
}

function formatTokens(usage?: ChatCompletionPayload["usage"]): string | undefined {
  if (!usage?.total_tokens) return undefined;
  const input = usage.prompt_tokens ?? 0;
  const output = usage.completion_tokens ?? 0;
  return `${usage.total_tokens.toLocaleString()} (${input.toLocaleString()} in / ${output.toLocaleString()} out)`;
}

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "opencodego";

    return {
      iconSrc: openCodeGoIcon,
      iconSlug: context.componentDefinition?.icon ?? "sparkles",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "OpenCode Go",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as ChatCompletionPayload | undefined;

    const details: Record<string, string> = {};
    const add = (label: string, value?: string) => {
      if (value) details[label] = value;
    };

    add("Completed At", formatTimestamp(payload?.timestamp));
    add("Model", data?.model);
    add("Tokens", formatTokens(data?.usage));

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
