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

type GetDailyUsagePayload = {
  period?: { startDate?: string; endDate?: string };
  messages?: {
    inputTokens?: number;
    outputTokens?: number;
  };
  claudeCode?: {
    sessions?: number;
    pullRequests?: number;
    estimatedCostUsd?: number;
  };
};

type GetDailyUsageConfiguration = {
  startDate?: string;
  endDate?: string;
};

function formatPeriod(period: GetDailyUsagePayload["period"]): string | undefined {
  if (!period?.startDate && !period?.endDate) return undefined;
  return `${period?.startDate ?? "?"} – ${period?.endDate ?? "?"}`;
}

function formatMessageTokens(inputTokens?: number, outputTokens?: number): string | undefined {
  if (inputTokens == null && outputTokens == null) return undefined;
  return `${(inputTokens ?? 0).toLocaleString()} in / ${(outputTokens ?? 0).toLocaleString()} out`;
}

// Mirrors the backend default in resolveDateRange (get_daily_usage.go):
// the last 7 days, ending today, in UTC.
function defaultDateRange(): { startDate: string; endDate: string } {
  const endDate = new Date();
  const startDate = new Date(endDate);
  startDate.setUTCDate(startDate.getUTCDate() - 7);

  return {
    startDate: startDate.toISOString().slice(0, 10),
    endDate: endDate.toISOString().slice(0, 10),
  };
}

function dateRangeMetadata(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as GetDailyUsageConfiguration | undefined;
  const defaults = defaultDateRange();
  const startDate = config?.startDate || defaults.startDate;
  const endDate = config?.endDate || defaults.endDate;

  return [{ icon: "calendar", label: `${startDate} → ${endDate}` }];
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
// covered period, Claude Code sessions, Messages API token usage, pull
// requests opened, and estimated spend.
function usageDetailFields(data: GetDailyUsagePayload | undefined): Record<string, string> {
  const entries = [
    detailEntry("Period", formatPeriod(data?.period)),
    detailEntry("Sessions", formatCount(data?.claudeCode?.sessions)),
    detailEntry("Message Tokens", formatMessageTokens(data?.messages?.inputTokens, data?.messages?.outputTokens)),
    detailEntry("Pull Requests", formatCount(data?.claudeCode?.pullRequests)),
    detailEntry("Estimated Cost", formatCostUsd(data?.claudeCode?.estimatedCostUsd)),
  ];

  return Object.fromEntries(entries.filter((entry): entry is [string, string] => entry != null));
}

export const getDailyUsageMapper: ComponentBaseMapper = {
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
        "Get Daily Usage",
      eventSections: lastExecution
        ? getDailyUsageEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      metadata: dateRangeMetadata(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const data = outputs?.default?.[0]?.data as GetDailyUsagePayload | undefined;

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

function getDailyUsageEventSections(
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
