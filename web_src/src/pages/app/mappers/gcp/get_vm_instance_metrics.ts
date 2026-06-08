import type { ComponentBaseProps } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import gcpIcon from "@/assets/icons/integrations/gcp.compute.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";
import { defaultStateFunction } from "../stateRegistry";

// A successful metrics fetch shows a "FETCHED" badge rather than the generic
// "COMPLETED" used by mutating actions.
const FETCHED_STATE = "fetched";

export const GET_VM_INSTANCE_METRICS_STATE_REGISTRY: EventStateRegistry = {
  stateMap: {
    ...DEFAULT_EVENT_STATE_MAP,
    [FETCHED_STATE]: { ...DEFAULT_EVENT_STATE_MAP.success, label: "FETCHED" },
  },
  getState: (execution: ExecutionInfo) => {
    const state = defaultStateFunction(execution);
    return state === "success" ? FETCHED_STATE : state;
  },
};

interface VMInstanceNodeMetadata {
  instanceName?: string;
  zone?: string;
}

interface GetVMInstanceMetricsConfiguration {
  instance?: string;
  lookbackPeriod?: string;
}

interface GetVMInstanceMetricsOutputData {
  name?: string;
  zone?: string;
  lookbackPeriod?: string;
  avgCpuUsagePercent?: number;
  avgNetworkInboundBytesPerSec?: number;
  avgNetworkOutboundBytesPerSec?: number;
}

const lookbackLabels: Record<string, string> = {
  "1h": "Last 1 hour",
  "6h": "Last 6 hours",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "14d": "Last 14 days",
};

export const getVMInstanceMetricsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name ?? "gcp";

    return {
      iconSrc: gcpIcon,
      iconSlug: context.componentDefinition?.icon ?? "chart-line",
      collapsedBackground: "bg-white",
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition?.label || "Get VM Metrics",
      eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
      metadata: metadataList(context.node),
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};

    if (context.execution.createdAt) {
      details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
    }

    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const result = outputs?.default?.[0]?.data as GetVMInstanceMetricsOutputData | undefined;
    if (!result) return details;

    if (result.name) details["Instance Name"] = result.name;
    if (result.lookbackPeriod) details["Lookback"] = lookbackLabels[result.lookbackPeriod] || result.lookbackPeriod;
    if (result.avgCpuUsagePercent !== undefined) details["Avg CPU"] = `${result.avgCpuUsagePercent}%`;
    if (result.avgNetworkInboundBytesPerSec !== undefined) {
      details["Avg Inbound"] = `${result.avgNetworkInboundBytesPerSec} B/s`;
    }
    if (result.avgNetworkOutboundBytesPerSec !== undefined) {
      details["Avg Outbound"] = `${result.avgNetworkOutboundBytesPerSec} B/s`;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as VMInstanceNodeMetadata | undefined;
  const configuration = node.configuration as GetVMInstanceMetricsConfiguration | undefined;

  const instanceName = nodeMetadata?.instanceName || configuration?.instance;
  if (instanceName) {
    metadata.push({ icon: "server", label: instanceName });
  }
  if (nodeMetadata?.zone) {
    metadata.push({ icon: "map-pin", label: nodeMetadata.zone });
  }
  if (configuration?.lookbackPeriod) {
    metadata.push({
      icon: "clock",
      label: lookbackLabels[configuration.lookbackPeriod] || configuration.lookbackPeriod,
    });
  }

  return metadata;
}
