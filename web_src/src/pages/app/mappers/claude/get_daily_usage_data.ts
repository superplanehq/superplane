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

type GetDailyUsageDataPayload = {
  period?: { startDate?: string; endDate?: string };
  claudeCode?: {
    sessions?: number;
    linesAdded?: number;
    linesRemoved?: number;
    pullRequests?: number;
    estimatedCostUsd?: number;
  };
};

function formatPeriod(period: GetDailyUsageDataPayload["period"]): string | undefined {
  if (!period?.startDate && !period?.endDate) return undefined;
  return `${period?.startDate ?? "?"} – ${period?.endDate ?? "?"}`;
}

function formatLinesChanged(added?: number, removed?: number): string | undefined {
  if (added == null && removed == null) return undefined;
  return `+${added ?? 0} / -${removed ?? 0}`;
}

function formatCostUsd(amount?: number): string | undefined {
  if (amount == null) return undefined;
  return `$${amount.toFixed(2)}`;
}

function formatCount(value?: number): string | undefined {
  return value != null ? String(value) : undefined;
}

function executedAt(execution: ExecutionInfo): string {
  const timestamp = execution.updatedAt || execution.createdAt;
  return timestamp ? new Date(timestamp).toLocaleString() : "-";
}

function detailEntry(label: string, value: string | undefined): [string, string] | undefined {
  return value != null ? [label, value] : undefined;
}

// The five metrics most relevant to a usage/cost review at a glance: the
// covered period, Claude Code sessions, lines changed, pull requests opened,
// and estimated spend.
function usageDetailFields(data: GetDailyUsageDataPayload | undefined): Record<string, string> {
  const entries = [
    detailEntry("Period", formatPeriod(data?.period)),
    detailEntry("Sessions", formatCount(data?.claudeCode?.sessions)),
    detailEntry("Lines Changed", formatLinesChanged(data?.claudeCode?.linesAdded, data?.claudeCode?.linesRemoved)),
    detailEntry("Pull Requests", formatCount(data?.claudeCode?.pullRequests)),
    detailEntry("Estimated Cost", formatCostUsd(data?.claudeCode?.estimatedCostUsd)),
  ];

  return Object.fromEntries(entries.filter((entry): entry is [string, string] => entry != null));
}

export const getDailyUsageDataMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "claude";

    return {
      iconSrc: claudeIcon,
      iconSlug: context.componentDefinition?.icon ?? "bar-chart",
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
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as GetDailyUsageDataPayload | undefined;

    return {
      "Executed At": executedAt(context.execution),
      ...usageDetailFields(data),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
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
