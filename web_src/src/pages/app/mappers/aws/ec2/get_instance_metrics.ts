import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { renderTimeAgo } from "@/components/TimeAgo";
import { stringOrDash } from "../../utils";
import awsEc2Icon from "@/assets/icons/integrations/aws.ec2.svg";

interface Configuration {
  region?: string;
  instance?: string;
  lookbackPeriod?: string;
  includeMemory?: boolean;
}

interface GetInstanceMetricsNodeMetadata {
  region?: string;
  instanceId?: string;
  instanceName?: string;
}

interface MetricsOutput {
  instanceId?: string;
  region?: string;
  lookbackPeriod?: string;
  start?: string;
  end?: string;
  avgCpuUsagePercent?: number;
  totalNetworkInBytes?: number;
  totalNetworkOutBytes?: number;
  avgNetworkInBytesPerSec?: number;
  avgNetworkOutBytesPerSec?: number;
  avgMemoryUsagePercent?: number | null;
}

const lookbackPeriodLabels: Record<string, string> = {
  "1h": "Last 1 hour",
  "6h": "Last 6 hours",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "14d": "Last 14 days",
};

function formatBytes(bytes: number | undefined): string {
  if (bytes == null || isNaN(bytes)) return "-";
  if (bytes >= 1_073_741_824) return `${(bytes / 1_073_741_824).toFixed(2)} GB`;
  if (bytes >= 1_048_576) return `${(bytes / 1_048_576).toFixed(2)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${bytes.toFixed(2)} B`;
}

function formatBytesPerSec(bps: number | undefined): string {
  if (bps == null || isNaN(bps)) return "-";
  return `${formatBytes(bps)}/s`;
}

export const getInstanceMetricsMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const componentName = context.componentDefinition.name || "unknown";

    return {
      title: context.node.name || context.componentDefinition.label || "Unnamed component",
      iconSrc: awsEc2Icon,
      iconColor: getColorClass(context.componentDefinition.color),
      collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
      collapsed: context.node.isCollapsed,
      eventSections: lastExecution
        ? getInstanceMetricsEventSections(context.nodes, lastExecution, componentName)
        : undefined,
      includeEmptyState: !lastExecution,
      metadata: getInstanceMetricsMetadata(context.node),
      eventStateMap: getStateMap(componentName),
    };
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const configuration = context.node.configuration as Configuration | undefined;
    const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
    const output = outputs?.default?.[0]?.data as MetricsOutput | undefined;

    const retrievedAt = resolveRetrievedAt(context.execution);
    const lookbackLabel = resolveLookbackLabel(configuration);

    if (!output) {
      return {
        "Retrieved At": stringOrDash(retrievedAt),
        Region: stringOrDash(configuration?.region),
        "Lookback Period": lookbackLabel,
        "Avg CPU": "-",
        "Avg Net In": "-",
        "Avg Net Out": "-",
      };
    }

    const details: Record<string, string> = {
      "Retrieved At": stringOrDash(retrievedAt),
      Region: stringOrDash(output.region ?? configuration?.region),
      "Lookback Period": lookbackLabel,
      "Avg CPU": output.avgCpuUsagePercent != null ? `${output.avgCpuUsagePercent}%` : "-",
      "Avg Net In": formatBytesPerSec(output.avgNetworkInBytesPerSec),
      "Avg Net Out": formatBytesPerSec(output.avgNetworkOutBytesPerSec),
    };

    if (output.avgMemoryUsagePercent != null) {
      details["Avg Memory"] = `${output.avgMemoryUsagePercent}%`;
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) {
      return "";
    }

    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function resolveRetrievedAt(execution: ExecutionInfo): string | undefined {
  const ts = execution.updatedAt ?? execution.createdAt;
  return ts ? new Date(ts).toLocaleString() : undefined;
}

function resolveLookbackLabel(configuration: Configuration | undefined): string {
  const period = configuration?.lookbackPeriod;
  return (period && lookbackPeriodLabels[period]) ?? period ?? "-";
}

function getInstanceMetricsMetadata(node: NodeInfo): MetadataItem[] {
  const configuration = node.configuration as Configuration | undefined;
  const nodeMetadata = node.metadata as GetInstanceMetricsNodeMetadata | undefined;

  const metaInstanceId = nodeMetadata?.instanceId;
  const metaInstanceName = nodeMetadata?.instanceName;
  const lookbackPeriod = configuration?.lookbackPeriod;

  const metadata: MetadataItem[] = [];

  const instanceLabel = metaInstanceName || metaInstanceId || configuration?.instance;
  if (instanceLabel) {
    metadata.push({ icon: "server", label: instanceLabel });
  }

  if (metaInstanceId && metaInstanceName && metaInstanceName !== metaInstanceId) {
    metadata.push({ icon: "hash", label: metaInstanceId });
  }

  if (lookbackPeriod) {
    metadata.push({ icon: "clock", label: lookbackPeriodLabels[lookbackPeriod] ?? lookbackPeriod });
  }

  return metadata;
}

function getInstanceMetricsEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
