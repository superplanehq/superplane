import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/utils/colors";
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
import { MetadataItem } from "@/ui/metadataList";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
import { formatTimeAgo } from "@/utils/date";
import { DropletNodeMetadata, GetDropletMetricsConfiguration } from "./types";

const LOOKBACK_PERIOD_LABELS: Record<string, string> = {
  "1h": "Last 1 hour",
  "6h": "Last 6 hours",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as Record<string, any> | undefined;
    if (!result) return details;

    details["Droplet ID"] = result.dropletId?.toString() || "-";
    details["Period"] = LOOKBACK_PERIOD_LABELS[result.lookbackPeriod] || result.lookbackPeriod || "-";
    details["From"] = result.start ? new Date(result.start * 1000).toLocaleString() : "-";
    details["To"] = result.end ? new Date(result.end * 1000).toLocaleString() : "-";

    const cpuSeries = result.cpu?.data?.result?.[0]?.values;
    if (Array.isArray(cpuSeries) && cpuSeries.length > 0) {
      const latest = cpuSeries[cpuSeries.length - 1];
      details["CPU (latest)"] = latest?.[1] !== undefined ? `${(parseFloat(latest[1]) * 100).toFixed(1)}%` : "-";
    }

    const memSeries = result.memory?.data?.result?.[0]?.values;
    if (Array.isArray(memSeries) && memSeries.length > 0) {
      const latest = memSeries[memSeries.length - 1];
      details["Memory (latest)"] = latest?.[1] !== undefined ? `${parseFloat(latest[1]).toFixed(1)}%` : "-";
    }

    const outSeries = result.publicOutboundBandwidth?.data?.result?.[0]?.values;
    if (Array.isArray(outSeries) && outSeries.length > 0) {
      const latest = outSeries[outSeries.length - 1];
      details["Outbound BW (latest)"] = latest?.[1] !== undefined ? `${parseFloat(latest[1]).toFixed(2)} Mbps` : "-";
    }

    const inSeries = result.publicInboundBandwidth?.data?.result?.[0]?.values;
    if (Array.isArray(inSeries) && inSeries.length > 0) {
      const latest = inSeries[inSeries.length - 1];
      details["Inbound BW (latest)"] = latest?.[1] !== undefined ? `${parseFloat(latest[1]).toFixed(2)} Mbps` : "-";
    }

    return details;
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
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
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}
