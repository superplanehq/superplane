import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { getBackgroundColorClass } from "@/utils/colors";
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
import type { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { DropletNodeMetadata, GetDropletMetricsConfiguration, GetDropletMetricsOutput } from "./types";

const LOOKBACK_PERIOD_LABELS: Record<string, string> = {
  "1h": "Last 1 hour",
  "6h": "Last 6 hours",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "14d": "Last 14 days",
};

export const getDropletMetricsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "digitalocean";

    return {
      iconSrc: doIcon,
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, unknown> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetDropletMetricsOutput | undefined;
    if (!result) return details;

    details["Droplet ID"] = result.dropletId?.toString() || "-";
    details["Period"] = result.lookbackPeriod
      ? LOOKBACK_PERIOD_LABELS[result.lookbackPeriod] || result.lookbackPeriod
      : "-";
    details["From"] = result.start ? new Date(result.start).toLocaleString() : "-";
    details["To"] = result.end ? new Date(result.end).toLocaleString() : "-";

    details["Avg. CPU Usage"] = result.avgCpuUsagePercent !== undefined ? `${result.avgCpuUsagePercent}%` : "-";
    details["Avg. Memory Usage"] =
      result.avgMemoryUsagePercent !== undefined ? `${result.avgMemoryUsagePercent}%` : "-";
    details["Avg. Outbound Bandwidth"] =
      result.avgPublicOutboundBandwidthMbps !== undefined ? `${result.avgPublicOutboundBandwidthMbps} Mbps` : "-";
    details["Avg. Inbound Bandwidth"] =
      result.avgPublicInboundBandwidthMbps !== undefined ? `${result.avgPublicInboundBandwidthMbps} Mbps` : "-";

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DropletNodeMetadata | undefined;
  const configuration = node.configuration as GetDropletMetricsConfiguration;

  if (nodeMetadata?.dropletName) {
    metadata.push({ icon: "hard-drive", label: nodeMetadata.dropletName });
  } else if (configuration?.droplet) {
    metadata.push({ icon: "info", label: `Droplet: ${configuration.droplet}` });
  }

  if (configuration?.lookbackPeriod) {
    const label = LOOKBACK_PERIOD_LABELS[configuration.lookbackPeriod] || configuration.lookbackPeriod;
    metadata.push({ icon: "clock", label });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id!,
    },
  ];
}
