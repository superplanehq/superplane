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
import type { AlertPolicyOutput, CreateAlertPolicyConfiguration } from "./types";

const METRIC_TYPE_LABELS: Record<string, string> = {
  "v1/insights/droplet/cpu": "CPU Usage",
  "v1/insights/droplet/memory_utilization_percent": "Memory Usage",
  "v1/insights/droplet/disk_read": "Disk Read",
  "v1/insights/droplet/disk_write": "Disk Write",
  "v1/insights/droplet/public_outbound_bandwidth": "Public Outbound BW",
  "v1/insights/droplet/public_inbound_bandwidth": "Public Inbound BW",
  "v1/insights/droplet/private_outbound_bandwidth": "Private Outbound BW",
  "v1/insights/droplet/private_inbound_bandwidth": "Private Inbound BW",
  "v1/insights/droplet/load_1": "Load Avg (1 min)",
  "v1/insights/droplet/load_5": "Load Avg (5 min)",
  "v1/insights/droplet/load_15": "Load Avg (15 min)",
};

export const createAlertPolicyMapper: ComponentBaseMapper = {
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
    const policy = outputs?.default?.[0]?.data as AlertPolicyOutput | undefined;
    if (!policy) return details;

    details["Policy UUID"] = policy.uuid || "-";
    details["Description"] = policy.description || "-";
    details["Metric"] = policy.type ? METRIC_TYPE_LABELS[policy.type] || policy.type : "-";
    details["Comparison"] = policy.compare || "-";
    details["Threshold"] = policy.value?.toString() ?? "-";
    details["Window"] = policy.window || "-";
    details["Enabled"] = policy.enabled ? "Yes" : "No";

    const emails: string[] = policy.alerts?.email ?? [];
    if (emails.length > 0) {
      details["Email Notifications"] = emails.join(", ");
    }

    const slackChannels = policy.alerts?.slack ?? [];
    if (slackChannels.length > 0) {
      details["Slack Channel"] = slackChannels.map((s) => s.channel).join(", ");
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
  const configuration = node.configuration as CreateAlertPolicyConfiguration;

  if (configuration?.description) {
    metadata.push({ icon: "bell", label: configuration.description });
  }

  if (configuration?.type) {
    const label = METRIC_TYPE_LABELS[configuration.type] || configuration.type;
    metadata.push({ icon: "chart-line", label });
  }

  if (configuration?.compare && configuration?.value !== undefined) {
    const op = configuration.compare === "GreaterThan" ? ">" : "<";
    metadata.push({ icon: "gauge", label: `${op} ${configuration.value}` });
  }

  return metadata;
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.rootEvent || !execution.createdAt) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id ?? "",
    },
  ];
}
