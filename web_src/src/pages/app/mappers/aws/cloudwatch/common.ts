import type { ComponentBaseContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import type React from "react";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import awsCloudwatchIcon from "@/assets/icons/integrations/aws.cloudwatch.svg";
import type { CloudWatchAlarm } from "./types";

export const MAX_METADATA_ITEMS = 3;

const COMPARISON_OPERATOR_SYMBOLS: Record<string, string> = {
  GreaterThanThreshold: ">",
  GreaterThanOrEqualToThreshold: ">=",
  LessThanThreshold: "<",
  LessThanOrEqualToThreshold: "<=",
};

/** The console serving a region. China regions are on a different host. */
function consoleHost(region: string): string {
  return region.startsWith("cn-") ? "console.amazonaws.cn" : "console.aws.amazon.com";
}

/** CloudWatch Console URL for an alarm. */
export function alarmConsoleUrl(region?: string, alarmName?: string): string {
  if (!region || !alarmName) {
    return "";
  }

  return `https://${region}.${consoleHost(region)}/cloudwatch/home?region=${region}#alarmsV2:alarm/${encodeURIComponent(alarmName)}`;
}

/** "AWS/EC2 / CPUUtilization" */
export function formatMetric(namespace?: string, metricName?: string): string {
  const parts = [namespace, metricName].filter(Boolean);
  return parts.length > 0 ? parts.join(" / ") : "-";
}

/** "Average > 80" */
export function formatCondition(alarm?: CloudWatchAlarm): string {
  const statistic = alarm?.extendedStatistic || alarm?.statistic;
  const threshold = alarm?.threshold;
  if (!statistic || threshold === undefined || threshold === null) {
    return "-";
  }

  const operator = alarm?.comparisonOperator
    ? (COMPARISON_OPERATOR_SYMBOLS[alarm.comparisonOperator] ?? alarm.comparisonOperator)
    : "";

  return [statistic, operator, String(threshold)].filter(Boolean).join(" ");
}

export function formatTimestamp(value?: string): string | undefined {
  return value ? new Date(value).toLocaleString() : undefined;
}

/** First value that is neither undefined nor empty. */
export function firstValue(...values: (string | undefined)[]): string | undefined {
  return values.find((value) => value !== undefined && value !== "");
}

/** The alarm emitted on the default output channel, if the execution produced one. */
export function alarmFromOutputs(execution: ExecutionInfo): CloudWatchAlarm | undefined {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as CloudWatchAlarm | undefined;
}

export function buildCloudWatchProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "unknown";

  return {
    title: context.node.name || context.componentDefinition.label || "Unnamed component",
    iconSrc: awsCloudwatchIcon,
    iconColor: getColorClass(context.componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    eventSections: lastExecution
      ? buildCloudWatchEventSections(context.nodes, lastExecution, componentName)
      : undefined,
    includeEmptyState: !lastExecution,
    metadata,
    eventStateMap: getStateMap(componentName),
  };
}

export function buildCloudWatchEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}

export function cloudwatchSubtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) {
    return "";
  }

  return renderTimeAgo(new Date(context.execution.createdAt));
}
