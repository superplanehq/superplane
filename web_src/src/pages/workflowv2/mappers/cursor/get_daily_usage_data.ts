import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "..";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import cursorIcon from "@/assets/icons/integrations/cursor.svg";
import { formatTimeAgo } from "@/utils/date";

type GetDailyUsageDataPayload = {
  data?: unknown[];
  period?: { startDate?: number; endDate?: number };
};

function formatPeriod(period: GetDailyUsageDataPayload["period"]): string | undefined {
  if (!period) return undefined;
  const start = period.startDate != null ? new Date(period.startDate) : null;
  const end = period.endDate != null ? new Date(period.endDate) : null;
  if (!start && !end) return undefined;
  const startStr = start ? start.toLocaleDateString() : "?";
  const endStr = end ? end.toLocaleDateString() : "?";
  return `${startStr} – ${endStr}`;
}

export const getDailyUsageDataMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "cursor";

    return {
      iconSrc: cursorIcon,
      iconSlug: context.componentDefinition?.icon ?? "cpu",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name ||
        context.componentDefinition?.label ||
        context.componentDefinition?.name ||
        "Get Daily Usage Data",
      eventSections: lastExecution
        ? getDailyUsageDataEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as GetDailyUsageDataPayload | undefined;

    if (data?.period) {
      const periodStr = formatPeriod(data.period);
      if (periodStr) {
        details["Period"] = periodStr;
      }
    }

    if (Array.isArray(data?.data)) {
      details["Users"] = String(data.data.length);
    }

    if (payload?.timestamp) {
      details["Fetched At"] = new Date(payload.timestamp).toLocaleString();
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? formatTimeAgo(new Date(timestamp)) : "";
  },
};

function getDailyUsageDataEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

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
