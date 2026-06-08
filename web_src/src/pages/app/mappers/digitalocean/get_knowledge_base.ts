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
import type { GetKBNodeMetadata, GetKnowledgeBaseConfiguration } from "./types";
import { baseEventSections, formatIndexingStatus } from "./event_helpers";

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

    const uuid = result.uuid as string | undefined;
    if (uuid) {
      details["View Knowledge Base"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${uuid}`;
      details["Knowledge Base Endpoint"] = `https://kbaas.do-ai.run/v1/${uuid}/retrieve`;
    }

    const dataSources = result.dataSources as unknown[] | undefined;
    if (dataSources && dataSources.length > 0) {
      details["Data Sources"] = String(dataSources.length);
    }

    const db = result.database as Record<string, unknown> | undefined;
    if (db) appendDatabaseDetails(details, db);

    const job = result.lastIndexingJob as Record<string, unknown> | undefined;
    if (job) appendLastIndexingJobDetails(details, job, uuid);

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function appendDatabaseDetails(details: Record<string, string>, db: Record<string, unknown>): void {
  details["Database"] = db.name ? String(db.name) : String(db.id || "-");
  const dbId = db.id as string | undefined;
  if (dbId) {
    details["View Database"] = `https://cloud.digitalocean.com/databases/${dbId}`;
  }
}

function appendLastIndexingJobDetails(
  details: Record<string, string>,
  job: Record<string, unknown>,
  uuid?: string,
): void {
  const completed = job.completedDataSources ?? 0;
  const total = job.totalDataSources ?? 0;
  details["Last Indexing"] = `${formatIndexingStatus(String(job.status))} — ${completed}/${total} sources`;
  const finishedAt = job.finishedAt as string | undefined;
  if (finishedAt) {
    details["Last Indexed At"] = new Date(finishedAt).toLocaleString();
  }
  if (uuid) {
    details["View Activity"] = `https://cloud.digitalocean.com/gen-ai/knowledge-bases/${uuid}/activity`;
  }
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as GetKBNodeMetadata | undefined;
  const configuration = node.configuration as GetKnowledgeBaseConfiguration | undefined;

  if (nodeMetadata?.knowledgeBaseName) {
    metadata.push({ icon: "book-marked", label: nodeMetadata.knowledgeBaseName });
  } else if (configuration?.knowledgeBase) {
    metadata.push({ icon: "book-marked", label: `KB: ${configuration.knowledgeBase}` });
  }

  return metadata;
}
