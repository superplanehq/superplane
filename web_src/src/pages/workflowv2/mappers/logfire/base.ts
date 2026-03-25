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
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
    }

    if (payload?.type) {
      details["Event Type"] = payload.type;
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
  minTimestamp?: string;
  maxTimestamp?: string;
  limit?: number;
  rowOriented?: boolean;
};

function truncateText(value: string, maxLength: number): string {
  const trimmed = value.trim();
  if (trimmed.length <= maxLength) return trimmed;
  return trimmed.slice(0, maxLength) + "...";
}

function formatTimestampForMetadata(value?: string): string | undefined {
  const trimmed = value?.trim();
  if (!trimmed) return undefined;
  const d = new Date(trimmed);
  if (Number.isNaN(d.getTime())) return trimmed;

  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function buildTimeWindowMetadata(minTs?: string, maxTs?: string): MetadataItem[] {
  if (minTs && maxTs) {
    return [
      {
        icon: "clock",
        label: `Window: ${minTs} -> ${maxTs}`,
      },
    ];
  }

  if (minTs) {
    return [
      {
        icon: "clock",
        label: `From: ${minTs}`,
      },
    ];
  }

  if (maxTs) {
    return [
      {
        icon: "clock",
        label: `Until: ${maxTs}`,
      },
    ];
  }

  return [];
}

function buildProjectMetadata(projectId?: string): MetadataItem[] {
  const trimmed = projectId?.trim();
  if (!trimmed) return [];

  return [
    {
      icon: "folder",
      // Avoid cutting off the project id in the node metadata list.
      label: `Project: ${truncateText(trimmed, 60)}`,
    },
  ];
}

function buildQueryMetadata(configuration: QueryLogfireNodeConfiguration | undefined): MetadataItem[] {
  const sql = configuration?.sql?.trim();
  if (sql) {
    return [
      {
        icon: "code",
        label: `SQL: ${truncateText(sql.replace(/\\s+/g, " "), 60)}`,
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

  const minTs = formatTimestampForMetadata(configuration?.minTimestamp);
  const maxTs = formatTimestampForMetadata(configuration?.maxTimestamp);

  // Keep timestamps first to match the UX expectation.
  const metadata: MetadataItem[] = [
    ...buildTimeWindowMetadata(minTs, maxTs),
    ...buildProjectMetadata(configuration?.projectId),
    ...buildQueryMetadata(configuration),
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
