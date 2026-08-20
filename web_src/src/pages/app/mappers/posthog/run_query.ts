import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type {
  ComponentBaseMapper,
  ComponentBaseContext,
  SubtitleContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  OutputPayload,
  NodeInfo,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "..";
import posthogIcon from "@/assets/icons/integrations/posthog.svg";
import { buildSubtitle } from "../utils";
import { renderTimeAgo } from "@/components/TimeAgo";

interface RunQueryConfiguration {
  projectId?: string;
  mode?: string;
  events?: string[];
  aggregation?: string;
  timeRange?: string;
  query?: string;
}

const AGGREGATION_LABELS: Record<string, string> = {
  events: "Matching events",
  countByEvent: "Count per event",
  totalCount: "Total count",
  uniqueUsers: "Unique users",
};

const TIME_RANGE_LABELS: Record<string, string> = {
  "1h": "Last hour",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 90 days",
};

interface RunQueryOutput {
  projectId?: string;
  columns?: string[];
  rowCount?: number;
  rows?: Record<string, unknown>[];
}

/**
 * Queries are shown on a single metadata line, so collapse the whitespace a
 * multi-line HogQL statement carries and cut it to something readable.
 */
function summarizeQuery(query: string): string {
  const collapsed = query.replace(/\s+/g, " ").trim();
  return collapsed.length > 60 ? `${collapsed.slice(0, 57)}...` : collapsed;
}

function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

function getRunQueryMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as RunQueryConfiguration | undefined;

  if (configuration?.projectId) {
    metadata.push({ icon: "folder", label: configuration.projectId });
  }

  // A built query has no query text to show, so summarize the fields it was built from.
  if (configuration?.mode === "hogql" || (!configuration?.mode && configuration?.query)) {
    if (configuration?.query) {
      metadata.push({ icon: "code", label: summarizeQuery(configuration.query) });
    }

    return metadata;
  }

  const events = configuration?.events ?? [];
  metadata.push({
    icon: "zap",
    label: events.length > 0 ? events.join(", ") : "All events",
  });

  metadata.push({
    icon: "activity",
    label: AGGREGATION_LABELS[configuration?.aggregation ?? "events"] ?? "Matching events",
  });

  metadata.push({
    icon: "clock",
    label: TIME_RANGE_LABELS[configuration?.timeRange ?? "7d"] ?? "Last 7 days",
  });

  return metadata;
}

export const runQueryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const { node, componentDefinition, lastExecutions } = context;
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const componentName = componentDefinition.name || node.componentName || "unknown";

    return {
      iconSrc: posthogIcon,
      iconColor: getColorClass(componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(componentDefinition.color),
      collapsed: node.isCollapsed,
      title: node.name || componentDefinition.label || "Run Query",
      metadata: getRunQueryMetadata(node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    return buildSubtitle("", context.execution.updatedAt || context.execution.createdAt);
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const details: Record<string, string> = {};

    if (!outputs?.default?.length) {
      return details;
    }

    const result = outputs.default[0].data as RunQueryOutput;
    if (!result) return details;

    if (result.projectId) details["Project"] = result.projectId;
    if (result.rowCount !== undefined) details["Rows"] = String(result.rowCount);
    if (result.columns?.length) details["Columns"] = result.columns.join(", ");

    return details;
  },
};
