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
import ociIcon from "@/assets/icons/integrations/oci.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type DefaultChannelPayload<T> = OutputPayload & {
  data?: T;
};

export const baseMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "oci";

    return {
      iconSrc: ociIcon,
      iconSlug: context.componentDefinition?.icon ?? "oci",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Oracle Cloud Infrastructure",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];

    if (payload?.type) {
      details["Event Type"] = payload.type;
    }

    if (payload?.timestamp) {
      details["Emitted At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

/** Omits only null/undefined keys — keeps empty strings and other falsy string values. */
export function compactDetails(entries: Array<[string, string | null | undefined]>): Record<string, string> {
  const details: Record<string, string> = {};

  for (const [key, value] of entries) {
    if (value != null) {
      details[key] = value;
    }
  }

  return details;
}

/** First item on the default output channel: uses `data` when present, otherwise the payload object. */
export function getDefaultChannelOutputData<T>(context: ExecutionDetailsContext): T | undefined {
  const outputs = context.execution.outputs as { default?: DefaultChannelPayload<T>[] } | undefined;
  const payload = outputs?.default?.[0];
  if (!payload) return undefined;
  return (payload.data ?? payload) as T;
}

export function executedAt(context: ExecutionDetailsContext): string {
  return context.execution.createdAt ? new Date(context.execution.createdAt).toLocaleString() : "-";
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.nodeId || !rootEvent?.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";
  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : undefined;

  return [
    {
      receivedAt,
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}
