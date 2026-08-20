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
import openRouterIcon from "@/assets/icons/integrations/openrouter.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

type ChatCompletionNodeMetadata = {
  model?: string;
  providerRouting?: boolean;
  structuredOutput?: boolean;
  webSearch?: boolean;
};

type ChatCompletionConfiguration = {
  model?: string;
  provider?: Record<string, unknown>;
  outputSchema?: string;
  webSearch?: boolean;
};

type ChatCompletionPayload = {
  model?: string;
  provider?: string;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
    cost?: number;
  };
};

// metadataList surfaces the configured model and whether provider routing is on.
// The model prefers backend node metadata (set in Setup) and falls back to the
// configuration so it shows before the first execution. The routing badge comes
// from the live configuration, since metadata can go stale (autosave updates
// configuration only, not metadata).
function metadataList(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const meta = node.metadata as ChatCompletionNodeMetadata | undefined;
  const config = node.configuration as ChatCompletionConfiguration | undefined;

  const model = meta?.model || config?.model;
  if (model) {
    items.push({ icon: "sparkles", label: model });
  }

  const routing = config ? Boolean(config.provider) : Boolean(meta?.providerRouting);
  if (routing) {
    items.push({ icon: "route", label: "Provider routing" });
  }

  const structured = config ? hasSchema(config.outputSchema) : Boolean(meta?.structuredOutput);
  if (structured) {
    items.push({ icon: "braces", label: "Structured output" });
  }

  const webSearch = config ? Boolean(config.webSearch) : Boolean(meta?.webSearch);
  if (webSearch) {
    items.push({ icon: "globe", label: "Web search" });
  }

  return items;
}

function hasSchema(schema: unknown): boolean {
  return typeof schema === "string" && schema.trim().length > 0;
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

function formatCost(usage?: ChatCompletionPayload["usage"]): string | undefined {
  return typeof usage?.cost === "number" ? `$${usage.cost.toFixed(6)}` : undefined;
}

function modelUrl(model?: string): string | undefined {
  return model ? `https://openrouter.ai/${model}` : undefined;
}

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "openrouter";

    return {
      iconSrc: openRouterIcon,
      iconSlug: context.componentDefinition?.icon ?? "sparkles",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "OpenRouter",
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
    // The provider that actually served the request can differ between runs on
    // the same model, so it is worth showing next to the model.
    add("Provider", data?.provider);
    add("Tokens", formatTokens(data?.usage));
    add("Cost", formatCost(data?.usage));
    add("View Model", modelUrl(data?.model));

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
