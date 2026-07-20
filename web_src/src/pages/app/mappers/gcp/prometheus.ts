import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
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
import gcpMonitoringIcon from "@/assets/icons/integrations/gcp.monitoring.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";

interface QueryConfiguration {
  query?: string;
  time?: string;
}

interface QueryRangeConfiguration {
  query?: string;
  start?: string;
  end?: string;
  step?: string;
}

interface QueryNodeMetadata {
  query?: string;
}

interface QueryOutputData {
  resultType?: string;
  seriesCount?: number;
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

function baseProps(
  context: ComponentBaseContext,
  iconSlug: string,
  fallbackTitle: string,
  metadata: MetadataItem[],
): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name ?? "gcp";

  return {
    iconSrc: gcpMonitoringIcon,
    iconSlug: context.componentDefinition?.icon ?? iconSlug,
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition?.label || fallbackTitle,
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function getQueryOutput(context: ExecutionDetailsContext): QueryOutputData | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as QueryOutputData | undefined;
}

function queryDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }
  const result = getQueryOutput(context);
  if (!result) return details;
  if (result.resultType) details["Result Type"] = result.resultType;
  if (result.seriesCount !== undefined) details["Series"] = String(result.seriesCount);
  return details;
}

// The configured query, preferring the value resolved onto the node at Setup.
function queryLabel(node: NodeInfo): string | undefined {
  const meta = node.metadata as QueryNodeMetadata | undefined;
  const config = node.configuration as QueryConfiguration | undefined;
  const query = meta?.query || config?.query;
  return query && !query.includes("{{") ? query : undefined;
}

function queryMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const label = queryLabel(node);
  if (label) metadata.push({ icon: "chart-line", label });
  return metadata;
}

function queryRangeMetadata(node: NodeInfo): MetadataItem[] {
  const metadata = queryMetadata(node);
  const config = node.configuration as QueryRangeConfiguration | undefined;
  // Show the resolution step (the most compact part of the window); skip
  // unresolved expressions like other GCP mappers.
  const step = config?.step;
  if (step && !step.includes("{{")) {
    metadata.push({ icon: "clock", label: `step ${step}` });
  }
  return metadata;
}

export const queryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "chart-line", "Query Managed Prometheus", queryMetadata(context.node));
  },
  getExecutionDetails: queryDetails,
  subtitle,
};

export const queryRangeMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "chart-line", "Query Managed Prometheus Range", queryRangeMetadata(context.node));
  },
  getExecutionDetails: queryDetails,
  subtitle,
};
