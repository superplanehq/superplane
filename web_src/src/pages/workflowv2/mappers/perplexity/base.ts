import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { MetadataItem } from "@/ui/metadataList";
import perplexityIcon from "@/assets/icons/integrations/perplexity.svg";
import { formatTimeAgo } from "@/utils/date";

interface RunAgentConfiguration {
  modelSource?: string;
  preset?: string;
  model?: string;
}

interface AgentPayload {
  model?: string;
  status?: string;
  text?: string;
  citations?: { type: string; url: string }[];
  usage?: {
    input_tokens: number;
    output_tokens: number;
    total_tokens: number;
  };
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as RunAgentConfiguration | undefined;

  if (configuration?.preset) {
    metadata.push({ icon: "sparkles", label: configuration.preset });
  } else if (configuration?.model) {
    metadata.push({ icon: "cpu", label: configuration.model });
  }

  return metadata;
}

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "perplexity";

    return {
      iconSrc: perplexityIcon,
      iconSlug: context.componentDefinition?.icon ?? "bot",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "Perplexity",
      metadata: metadataList(context.node),
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as AgentPayload | undefined;

    if (context.execution.createdAt) {
      details["Started At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    if (data?.model) {
      details["Model"] = data.model;
    }

    if (data?.citations && data.citations.length > 0) {
      details["Citations"] = `${data.citations.length} source${data.citations.length === 1 ? "" : "s"}`;
    }

    if (data?.usage) {
      const { input_tokens, output_tokens, total_tokens } = data.usage;
      if (total_tokens) {
        details["Tokens"] =
          `${total_tokens.toLocaleString()} (${input_tokens.toLocaleString()} in / ${output_tokens.toLocaleString()} out)`;
      }
    }

    if (context.execution.resultMessage) {
      details["Error"] = context.execution.resultMessage;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

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
