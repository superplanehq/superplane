import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass } from "@/utils/colors";
import doIcon from "@/assets/icons/integrations/digitalocean.svg";
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
import type { DatabaseNodeMetadata, GetClusterConfiguration } from "./types";

const CONFIG_LABELS: Record<string, string> = {
  autovacuum_analyze_scale_factor: "Analyze Scale",
  autovacuum_analyze_threshold: "Analyze Threshold",
  autovacuum_naptime: "Autovacuum Interval",
  autovacuum_vacuum_cost_delay: "Vacuum Cost Delay",
  autovacuum_vacuum_cost_limit: "Vacuum Cost Limit",
  idle_in_transaction_session_timeout: "Idle Tx Timeout",
  default_toast_compression: "Toast Compression",
  max_parallel_workers: "Parallel Workers",
};

export const getClusterConfigurationMapper: ComponentBaseMapper = {
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
    const payload = outputs?.default?.[0]?.data as Record<string, unknown> | undefined;
    if (!payload) return details;

    details["Cluster"] = String(payload.databaseClusterName || "-");

    const config = payload.config as Record<string, unknown> | undefined;
    if (!config) return details;

    for (const key of Object.keys(config).sort()) {
      const value = config[key];
      if (detailsCount(details) >= 6) break;
      if (value == null || typeof value === "object") continue;
      details[toLabel(key)] = String(value);
    }

    return details;
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },
};

function metadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const nodeMetadata = node.metadata as DatabaseNodeMetadata | undefined;
  const configuration = node.configuration as GetClusterConfiguration;

  if (nodeMetadata?.databaseClusterName) {
    metadata.push({ icon: "server", label: nodeMetadata.databaseClusterName });
  } else if (configuration?.databaseCluster) {
    metadata.push({ icon: "info", label: `Cluster ID: ${configuration.databaseCluster}` });
  }

  return metadata;
}

function detailsCount(details: Record<string, string>): number {
  return Object.keys(details).length;
}

function toLabel(key: string): string {
  if (CONFIG_LABELS[key]) {
    return CONFIG_LABELS[key];
  }

  return key
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) return [];

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}
