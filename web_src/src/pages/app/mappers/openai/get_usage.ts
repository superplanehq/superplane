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
import openAiIcon from "@/assets/icons/integrations/openai.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type GetUsagePayload = {
  data?: unknown[];
  period?: { startDate?: string; endDate?: string };
  usageType?: string;
};

function formatPeriod(period: GetUsagePayload["period"]): string | undefined {
  if (!period) return undefined;
  if (!period.startDate && !period.endDate) return undefined;
  return `${period.startDate ?? "?"} – ${period.endDate ?? "?"}`;
}

export const getUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "openai";

    return {
      iconSrc: openAiIcon,
      iconSlug: context.componentDefinition?.icon ?? "bar-chart",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Get Usage Data",
      eventSections: lastExecution ? getUsageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as GetUsagePayload | undefined;

    if (data?.usageType) {
      details["Usage Type"] = data.usageType;
    }

    const periodStr = formatPeriod(data?.period);
    if (periodStr) {
      details["Period"] = periodStr;
    }

    if (Array.isArray(data?.data)) {
      details["Buckets"] = String(data.data.length);
    }

    if (payload?.timestamp) {
      details["Fetched At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function getUsageEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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
