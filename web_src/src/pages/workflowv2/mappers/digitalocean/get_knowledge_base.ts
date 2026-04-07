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
import type { GetKBNodeMetadata, GetKnowledgeBaseConfiguration } from "./types";

export const getKnowledgeBaseMapper: ComponentBaseMapper = {
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

    details["Knowledge Base"] = String(result.name || result.uuid || "-");

    const uuid = result.uuid as string | undefined;
    if (uuid) {
      details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${uuid}`;
    }

    if (result.databaseStatus) {
      details["Database Status"] = String(result.databaseStatus);
    }

    const db = result.database as Record<string, unknown> | undefined;
    if (db) {
      const dbName = db.name ? String(db.name) : String(db.id || "-");
      details["Database"] = dbName;

      const dbId = db.id as string | undefined;
      if (dbId) {
        details["View Database"] = `https://cloud.digitalocean.com/databases/${dbId}`;
      }
    }

    details["Region"] = String(result.region || "-");

    if (result.embeddingModelName) {
      details["Embedding Model"] = String(result.embeddingModelName);
    }

    if (result.projectName) {
      details["Project"] = String(result.projectName);
    }

    const dataSources = result.dataSources as unknown[] | undefined;
    if (dataSources && dataSources.length > 0) {
      details["Data Sources"] = String(dataSources.length);
    }

    const job = result.lastIndexingJob as Record<string, unknown> | undefined;
    if (job) {
      const completed = job.completedDataSources ?? 0;
      const total = job.totalDataSources ?? 0;
      details["Last Indexing"] = `${job.status} — ${completed}/${total} sources`;

      const finishedAt = job.finishedAt as string | undefined;
      if (finishedAt) {
        details["Last Indexed At"] = new Date(finishedAt).toLocaleString();
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
  const nodeMetadata = node.metadata as GetKBNodeMetadata | undefined;
  const configuration = node.configuration as GetKnowledgeBaseConfiguration | undefined;

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
