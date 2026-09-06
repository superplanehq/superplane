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
import openCodeGoIcon from "@/assets/icons/integrations/opencodego.svg";
import { renderTimeAgo } from "@/components/TimeAgo";

type GetUsagePayload = {
  rollingStatus?: string;
  rollingPercent?: number;
  rollingResetsAt?: string;
  weeklyStatus?: string;
  weeklyPercent?: number;
  weeklyResetsAt?: string;
  monthlyStatus?: string;
  monthlyPercent?: number;
  monthlyResetsAt?: string;
};

function formatTimestamp(timestamp?: string): string | undefined {
  return timestamp ? new Date(timestamp).toLocaleString() : undefined;
}

function windowDetails(
  details: Record<string, string>,
  label: string,
  status?: string,
  percent?: number,
  resetsAt?: string,
): void {
  if (percent === undefined) return;
  const resets = resetsAt ? formatTimestamp(resetsAt) : undefined;
  details[label] = `${Math.round(percent)}% used${resets ? ` · resets ${resets}` : ""}`;
  if (status && status !== "ok") {
    details[`${label} Status`] = status;
  }
}

export const getUsageMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "opencodego";

    return {
      iconSrc: openCodeGoIcon,
      iconSlug: context.componentDefinition?.icon ?? "activity",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title:
        context.node.name || context.componentDefinition?.label || context.componentDefinition?.name || "Get Usage",
      eventSections: lastExecution ? getUsageEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const payload = outputs?.default?.[0];
    const data = payload?.data as GetUsagePayload | undefined;

    const details: Record<string, string> = {};
    if (data) {
      windowDetails(details, "Rolling", data.rollingStatus, data.rollingPercent, data.rollingResetsAt);
      windowDetails(details, "Weekly", data.weeklyStatus, data.weeklyPercent, data.weeklyResetsAt);
      windowDetails(details, "Monthly", data.monthlyStatus, data.monthlyPercent, data.monthlyResetsAt);
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
