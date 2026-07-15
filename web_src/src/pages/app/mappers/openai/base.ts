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
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "openai";

    return {
      iconSrc: openAiIcon,
      iconSlug: context.componentDefinition?.icon ?? "sparkles",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "OpenAI",
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

    if (payload?.type) {
      details["Event Type"] = payload.type;
    }

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
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
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
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

type ResponseNodeMetadata = {
  model?: string;
  structuredOutput?: boolean;
  codeInterpreter?: boolean;
};

type ResponseConfiguration = {
  model?: string;
  outputSchema?: string;
  codeInterpreter?: boolean;
};

// metadataList surfaces the configured model and structured-output state on the
// canvas node tile. The model prefers backend node metadata (set in Setup) and
// falls back to the configuration so it shows before the first execution. The
// structured-output badge is derived from the live configuration, since metadata
// can go stale (autosave updates configuration only, not metadata).
function metadataList(node: NodeInfo): MetadataItem[] {
  const items: MetadataItem[] = [];
  const meta = node.metadata as ResponseNodeMetadata | undefined;
  const config = node.configuration as ResponseConfiguration | undefined;

  const model = meta?.model || config?.model;
  if (model) {
    items.push({ icon: "sparkles", label: model });
  }

  const structured = config ? hasSchema(config.outputSchema) : Boolean(meta?.structuredOutput);
  if (structured) {
    items.push({ icon: "braces", label: "Structured output" });
  }

  const codeInterpreter = config ? Boolean(config.codeInterpreter) : Boolean(meta?.codeInterpreter);
  if (codeInterpreter) {
    items.push({ icon: "terminal", label: "Code interpreter" });
  }

  return items;
}

function hasSchema(schema: unknown): boolean {
  return typeof schema === "string" && schema.trim().length > 0;
}
