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
};

type ChatCompletionConfiguration = {
  model?: string;
  provider?: Record<string, unknown>;
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

  return items;
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
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as ChatCompletionPayload | undefined;

    if (payload?.timestamp) {
      details["Completed At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (data?.model) {
      details["Model"] = data.model;
    }

    // The provider that actually served the request can differ between runs on
    // the same model, so it is worth showing next to the model.
    if (data?.provider) {
      details["Provider"] = data.provider;
    }

    const usage = data?.usage;
    if (usage?.total_tokens) {
      const input = usage.prompt_tokens ?? 0;
      const output = usage.completion_tokens ?? 0;
      details["Tokens"] =
        `${usage.total_tokens.toLocaleString()} (${input.toLocaleString()} in / ${output.toLocaleString()} out)`;
    }

    if (typeof usage?.cost === "number") {
      details["Cost"] = `$${usage.cost.toFixed(6)}`;
    }

    if (data?.model) {
      details["View Model"] = `https://openrouter.ai/${data.model}`;
    }

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
