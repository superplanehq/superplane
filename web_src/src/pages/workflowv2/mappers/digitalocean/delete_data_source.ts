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
import type { DeleteDSNodeMetadata, DeleteDataSourceConfiguration } from "./types";

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

    details["Knowledge Base"] = nodeMetadata?.knowledgeBaseName || String(result.knowledgeBaseName || result.knowledgeBaseUUID || "-");

    const kbUUID = result.knowledgeBaseUUID as string | undefined;
    if (kbUUID) {
      details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${kbUUID}`;
    }

    if (result.dataSourceUUID) {
      details["Data Source"] = nodeMetadata?.dataSourceName || String(result.dataSourceUUID);
    }

    const job = result.indexingJob as Record<string, unknown> | undefined;
    if (job) {
      details["Indexing Status"] = `${formatIndexingStatus(String(job.status))}`;

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

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function formatIndexingStatus(status: string): string {
  const lower = status.toLowerCase().replace(/^index_job_status_/, "");
  const map: Record<string, string> = {
    completed: "Completed",
    successful: "Successful",
    no_changes: "No changes",
    partial: "Partially completed",
    running: "Running",
    pending: "Pending",
    failed: "Failed",
    cancelled: "Cancelled",
    in_progress: "In progress"
  };
  return map[lower] ?? status;
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
