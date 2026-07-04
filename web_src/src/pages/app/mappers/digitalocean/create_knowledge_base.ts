import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
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
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { CreateKnowledgeBaseConfiguration } from "./types";

function getKBResult(context: ExecutionDetailsContext): Record<string, unknown> | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
}

function getKnowledgeBaseDetails(result: Record<string, unknown>): Record<string, string> {
  const details: Record<string, string> = {};

  details["Knowledge Base"] = String(result.name || "-");

  if (result.uuid) {
    details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${result.uuid}`;
  }

  if (result.databaseId) {
    details["View OpenSearch Database"] = `https://cloud.digitalocean.com/databases/${result.databaseId}`;
  }

  details["Region"] = String(result.region || "-");
  details["Embedding Model"] = String(result.embeddingModelName || result.embeddingModelUUID || "-");
  details["Project"] = String(result.projectName || result.projectId || "-");

  const tags = result.tags as string[] | undefined;
  if (tags && tags.length > 0) {
    details["Tags"] = tags.join(", ");
  }

  return details;
}

export const createKnowledgeBaseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const result = getKBResult(context);
    if (!result) return details;

    return { ...details, ...getKnowledgeBaseDetails(result) };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as CreateKnowledgeBaseConfiguration;

  if (configuration?.name) {
    metadata.push({ icon: "brain", label: configuration.name });
  }

  if (configuration?.region) {
    metadata.push({ icon: "map-pin", label: configuration.region });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) return [];

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}
