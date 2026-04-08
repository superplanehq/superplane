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
import type { AddDSNodeMetadata, AddDataSourceConfiguration } from "./types";

export const addDataSourceMapper: ComponentBaseMapper = {
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

    details["Knowledge Base"] = String(result.knowledgeBaseName || result.knowledgeBaseUUID || "-");

    const kbUUID = result.knowledgeBaseUUID as string | undefined;
    if (kbUUID) {
      details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${kbUUID}`;
    }

    if (result.dataSourceUUID) {
      details["Data Source"] = String(result.dataSourceUUID);
    }

    const job = result.indexingJob as Record<string, unknown> | undefined;
    if (job) {
      details["Indexing Status"] = String(job.status || "-");

      const completed = job.completedDataSources ?? 0;
      const total = job.totalDataSources ?? 0;
      details["Data Sources Indexed"] = `${completed}/${total} completed`;

      if (job.totalTokens) {
        details["Total Tokens"] = String(job.totalTokens);
      }

      const finishedAt = job.finishedAt as string | undefined;
      if (finishedAt) {
        details["Finished At"] = new Date(finishedAt).toLocaleString();
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

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as AddDSNodeMetadata | undefined;
  const configuration = node.configuration as AddDataSourceConfiguration | undefined;

  if (nodeMetadata?.knowledgeBaseName) {
    metadata.push({ icon: "brain", label: nodeMetadata.knowledgeBaseName });
  } else if (configuration?.knowledgeBase) {
    metadata.push({ icon: "brain", label: `KB: ${configuration.knowledgeBase}` });
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
