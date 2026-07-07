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
import claudeIcon from "@/assets/icons/integrations/claude.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

type BatchRequestCounts = {
  processing?: number;
  succeeded?: number;
  errored?: number;
  canceled?: number;
  expired?: number;
};

type CreateBatchMessagePayload = {
  status?: string;
  batchId?: string;
  requestCounts?: BatchRequestCounts;
  results?: { customId?: string }[];
};

type CreateBatchMessageConfiguration = {
  model?: string;
  outputSchema?: string;
  requests?: { customId?: string }[];
};

function executedAt(execution: ExecutionInfo): string {
  const timestamp = execution.updatedAt || execution.createdAt;
  return timestamp ? new Date(timestamp).toLocaleString() : "-";
}

function detailEntry(label: string, value: string | undefined): [string, string] | undefined {
  return value != null ? [label, value] : undefined;
}

function formatCount(value: number | undefined): string | undefined {
  return value != null ? String(value) : undefined;
}

// Batch ID, status, and the succeeded/errored split are the fields most useful
// for a quick review of a completed (or failed-to-complete) batch run.
function batchDetailFields(data: CreateBatchMessagePayload | undefined): Record<string, string> {
  const entries = [
    detailEntry("Status", data?.status),
    detailEntry("Batch ID", data?.batchId),
    detailEntry("Succeeded", formatCount(data?.requestCounts?.succeeded)),
    detailEntry("Errored", formatCount(data?.requestCounts?.errored)),
  ];

  return Object.fromEntries(entries.filter((entry): entry is [string, string] => entry != null));
}

function requestsMetadata(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as CreateBatchMessageConfiguration | undefined;
  const items: MetadataItem[] = [];

  const model = config?.model;
  if (model) {
    items.push({ icon: "sparkles", label: model });
  }

  const count = config?.requests?.length;
  if (count) {
    items.push({ icon: "layers", label: `${count} request${count === 1 ? "" : "s"}` });
  }

  return items;
}

export const createBatchMessageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "claude";

    return {
      iconSrc: claudeIcon,
      iconSlug: context.componentDefinition?.icon ?? "layers",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Create Batch Message",
      eventSections: lastExecution
        ? createBatchMessageEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      metadata: requestsMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as CreateBatchMessagePayload | undefined;

    return {
      "Executed At": executedAt(context.execution),
      ...batchDetailFields(data),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function createBatchMessageEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
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
