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
import logfireIcon from "@/assets/icons/integrations/logfire.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "logfire";

    return {
      iconSrc: logfireIcon,
      iconSlug: context.componentDefinition?.icon ?? "loader",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "logfire",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
      metadata: metadataList(context.node),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const configuration = context.execution.configuration as QueryLogfireNodeConfiguration | undefined;
    const nodeMetadata = context.node.metadata as QueryLogfireNodeMetadata | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    const projectName = nodeMetadata?.project?.name?.trim() || configuration?.projectId;
    if (projectName) {
      details["Project"] = projectName;
    }

    if (configuration?.sql) {
      const normalized = configuration.sql.trim().replace(/\s+/g, " ");
      details["SQL"] = normalized.length > 120 ? normalized.slice(0, 120) + "..." : normalized;
    }

    details["Rows Returned"] = String(countRows(payload?.data));

    const executedAt = context.execution.updatedAt || context.execution.createdAt;
    if (executedAt) {
      details["Executed At"] = new Date(executedAt).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

type QueryLogfireNodeConfiguration = {
  sql?: string;
  projectId?: string;
  timeWindow?: string;
  minTimestamp?: string;
  maxTimestamp?: string;
  limit?: number;
  rowOriented?: boolean;
};

type QueryLogfireNodeMetadata = {
  project?: { id?: string; name?: string };
};

function countRows(data: unknown): number {
  if (!data || typeof data !== "object") return 0;

  const record = data as Record<string, unknown>;

  if (Array.isArray(record.rows)) return record.rows.length;

  const columns = Object.values(record);
  for (const col of columns) {
    if (Array.isArray(col)) return col.length;
    if (col && typeof col === "object" && Array.isArray((col as Record<string, unknown>).values)) {
      return ((col as Record<string, unknown>).values as unknown[]).length;
    }
  }

  return 0;
}

function truncateText(value: string, maxLength: number): string {
  const trimmed = value.trim();
  if (trimmed.length <= maxLength) return trimmed;
  return trimmed.slice(0, maxLength) + "...";
}

const timeWindowLabels: Record<string, string> = {
  "5m": "Last 5 min",
  "15m": "Last 15 min",
  "1h": "Last 1 hour",
  "6h": "Last 6 hours",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
};

function buildTimeWindowMetadata(configuration?: QueryLogfireNodeConfiguration): MetadataItem[] {
  const tw = configuration?.timeWindow?.trim();
  if (!tw || tw === "none") return [];

  if (tw === "custom") {
    const parts: string[] = [];
    if (configuration?.minTimestamp?.trim()) parts.push(`from ${configuration.minTimestamp.trim()}`);
    if (configuration?.maxTimestamp?.trim()) parts.push(`to ${configuration.maxTimestamp.trim()}`);
    if (parts.length === 0) return [];
    return [{ icon: "clock", label: `Window: ${parts.join(" ")}` }];
  }

  const label = timeWindowLabels[tw];
  if (!label) return [];
  return [{ icon: "clock", label }];
}

function buildProjectMetadata(projectId?: string, nodeMetadata?: QueryLogfireNodeMetadata): MetadataItem[] {
  const projectName = nodeMetadata?.project?.name?.trim();
  const label = projectName || projectId?.trim();
  if (!label) return [];

  return [
    {
      icon: "folder",
      label: `Project: ${truncateText(label, 60)}`,
    },
  ];
}

function buildQueryMetadata(configuration: QueryLogfireNodeConfiguration | undefined): MetadataItem[] {
  const sql = configuration?.sql?.trim();
  if (sql) {
    return [
      {
        icon: "code",
        label: `SQL: ${truncateText(sql.replace(/\s+/g, " "), 60)}`,
      },
    ];
  }

  const limit = configuration?.limit;
  if (typeof limit === "number" && limit > 0) {
    return [
      {
        icon: "funnel",
        label: `Limit: ${limit}`,
      },
    ];
  }

  if (configuration?.rowOriented) {
    return [
      {
        icon: "list",
        label: "Row JSON",
      },
    ];
  }

  return [];
}

function metadataList(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as QueryLogfireNodeConfiguration | undefined;
  const nodeMetadata = node.metadata as QueryLogfireNodeMetadata | undefined;

  const metadata: MetadataItem[] = [
    ...buildProjectMetadata(configuration?.projectId, nodeMetadata),
    ...buildQueryMetadata(configuration),
    ...buildTimeWindowMetadata(configuration),
  ];

  return metadata.slice(0, 3);
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
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
