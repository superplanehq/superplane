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
import type { DeleteDSNodeMetadata, DeleteDataSourceConfiguration } from "./types";
import { baseEventSections, formatIndexingStatus } from "./event_helpers";

export const deleteDataSourceMapper: ComponentBaseMapper = {
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

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    if (!result) return details;

    const nodeMetadata = context.node.metadata as DeleteDSNodeMetadata | undefined;

    details["Knowledge Base"] =
      nodeMetadata?.knowledgeBaseName || String(result.knowledgeBaseName || result.knowledgeBaseUUID || "-");

    const kbUUID = result.knowledgeBaseUUID as string | undefined;
    if (kbUUID) {
      details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${kbUUID}`;
    }

    if (result.dataSourceUUID) {
      details["Data Source"] = nodeMetadata?.dataSourceName || String(result.dataSourceUUID);
    }

    const job = result.indexingJob as Record<string, unknown> | undefined;
    if (job) appendIndexingJobDetails(details, job, kbUUID);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function appendIndexingJobDetails(
  details: Record<string, string>,
  job: Record<string, unknown>,
  kbUUID?: string,
): void {
  details["Indexing Status"] = formatIndexingStatus(String(job.status));
  const completed = job.completedDataSources ?? 0;
  const total = job.totalDataSources ?? 0;
  details["Data Sources Indexed"] = `${completed}/${total} completed`;
  if (job.totalTokens) {
    details["Total Tokens"] = String(job.totalTokens);
  }
  const finishedAt = job.finishedAt as string | undefined;
  if (finishedAt) {
    details["Indexing finished at"] = new Date(finishedAt).toLocaleString();
  }
  if (kbUUID) {
    details["View Activity"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${kbUUID}/activity`;
  }
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DeleteDSNodeMetadata | undefined;
  const configuration = node.configuration as DeleteDataSourceConfiguration | undefined;

  if (nodeMetadata?.knowledgeBaseName) {
    metadata.push({ icon: "book-marked", label: nodeMetadata.knowledgeBaseName });
  } else if (configuration?.knowledgeBase) {
    metadata.push({ icon: "book-marked", label: `KB: ${configuration.knowledgeBase}` });
  }

  if (nodeMetadata?.dataSourceName) {
    metadata.push({ icon: "database", label: nodeMetadata.dataSourceName });
  }

  return metadata;
}
