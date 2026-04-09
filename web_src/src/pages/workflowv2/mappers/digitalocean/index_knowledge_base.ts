import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/lib/colors";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { IndexKBNodeMetadata, IndexKnowledgeBaseConfiguration } from "./types";
import { baseEventSections, formatIndexingStatus } from "./event_helpers";

export const indexKnowledgeBaseMapper: ComponentBaseMapper = {
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

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    if (!result) return details;

    const startedAt = result.startedAt as string | undefined;
    if (startedAt) {
      details["Started At"] = new Date(startedAt).toLocaleString();
    }

    const finishedAt = result.finishedAt as string | undefined;
    if (finishedAt) {
      details["Finished At"] = new Date(finishedAt).toLocaleString();
    }

    details["Knowledge Base"] = String(result.knowledgeBaseName || result.knowledgeBaseUUID || "-");

    const kbUUID = result.knowledgeBaseUUID as string | undefined;
    if (kbUUID) {
      details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${kbUUID}`;
      details["View Activity"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${kbUUID}/activity`;
    }

    details["Indexing Status"] = formatIndexingStatus(String(result.status || "-"));

    const completed = result.completedDataSources ?? 0;
    const total = result.totalDataSources ?? 0;
    details["Data Sources Indexed"] = `${completed}/${total} completed`;

    if (result.totalTokens) {
      details["Total Tokens"] = String(result.totalTokens);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as IndexKBNodeMetadata | undefined;
  const configuration = node.configuration as IndexKnowledgeBaseConfiguration | undefined;

  if (nodeMetadata?.knowledgeBaseName) {
    metadata.push({ icon: "book-marked", label: nodeMetadata.knowledgeBaseName });
  } else if (configuration?.knowledgeBase) {
    metadata.push({ icon: "book-marked", label: `KB: ${configuration.knowledgeBase}` });
  }

  return metadata;
}
