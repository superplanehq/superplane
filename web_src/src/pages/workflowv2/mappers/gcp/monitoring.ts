import type { ComponentBaseProps } from "@/ui/componentBase";
import type React from "react";
import { getStateMap } from "..";
import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import gcpIcon from "@/assets/icons/integrations/gcp.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";

interface CreateAlertingPolicyConfiguration {
  displayName?: string;
  metricType?: string;
  comparison?: string;
  threshold?: number;
}

interface AlertPolicySelectorConfiguration {
  alertPolicy?: string;
  displayName?: string;
}

// Resolved at Setup time by the backend so the collapsed node can show the
// policy's display name instead of its numeric ID.
interface AlertPolicyNodeMetadata {
  policyName?: string;
  displayName?: string;
  id?: string;
}

interface AlertingPolicyOutputData {
  name?: string;
  id?: string;
  displayName?: string;
  enabled?: boolean;
  comparison?: string;
  thresholdValue?: number;
  duration?: string;
}

const metricLabels: Record<string, string> = {
  "compute.googleapis.com/instance/cpu/utilization": "CPU utilization",
  "compute.googleapis.com/instance/network/sent_bytes_count": "Network sent",
  "compute.googleapis.com/instance/network/received_bytes_count": "Network received",
  "compute.googleapis.com/instance/disk/read_bytes_count": "Disk read",
  "compute.googleapis.com/instance/disk/write_bytes_count": "Disk write",
};

const comparisonLabels: Record<string, string> = {
  COMPARISON_GT: "above",
  COMPARISON_LT: "below",
};

function lastSegment(value: string | undefined): string | undefined {
  if (!value || value.includes("{{")) return undefined;
  const trimmed = value.trim();
  const idx = trimmed.lastIndexOf("/");
  return idx >= 0 ? trimmed.slice(idx + 1) : trimmed;
}

function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

function baseProps(
  context: ComponentBaseContext,
  iconSlug: string,
  fallbackTitle: string,
  metadata: MetadataItem[],
): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name ?? "gcp";

  return {
    iconSrc: gcpIcon,
    iconSlug: context.componentDefinition?.icon ?? iconSlug,
    collapsedBackground: "bg-white",
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition?.label || fallbackTitle,
    eventSections: lastExecution ? baseEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function policyDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const result = outputs?.default?.[0]?.data as AlertingPolicyOutputData | undefined;
  if (!result) return details;

  if (result.displayName) details["Display Name"] = result.displayName;
  if (result.id) details["Policy ID"] = result.id;
  if (result.enabled !== undefined) details["Enabled"] = result.enabled ? "Yes" : "No";
  if (result.comparison && result.thresholdValue !== undefined) {
    details["Condition"] = `${comparisonLabels[result.comparison] || result.comparison} ${result.thresholdValue}`;
  }
  if (result.duration) details["Duration"] = result.duration;
  return details;
}

export const createAlertingPolicyMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "bell", "Create Alerting Policy", createMetadata(context.node));
  },
  getExecutionDetails: policyDetails,
  subtitle,
};

export const getAlertingPolicyMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "bell", "Get Alerting Policy", selectorMetadata(context.node));
  },
  getExecutionDetails: policyDetails,
  subtitle,
};

export const deleteAlertingPolicyMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "trash-2", "Delete Alerting Policy", selectorMetadata(context.node));
  },
  getExecutionDetails: policyDetails,
  subtitle,
};

export const updateAlertingPolicyMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "bell", "Update Alerting Policy", selectorMetadata(context.node));
  },
  getExecutionDetails: policyDetails,
  subtitle,
};

function createMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as CreateAlertingPolicyConfiguration | undefined;
  if (config?.displayName) metadata.push({ icon: "bell", label: config.displayName });
  if (config?.metricType) {
    const label = metricLabels[config.metricType] || config.metricType;
    const cmp = config.comparison ? comparisonLabels[config.comparison] || "" : "";
    metadata.push({
      icon: "chart-line",
      label: cmp && config.threshold !== undefined ? `${label} ${cmp} ${config.threshold}` : label,
    });
  }
  return metadata;
}

function selectorMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as AlertPolicySelectorConfiguration | undefined;
  const nodeMeta = node.metadata as AlertPolicyNodeMetadata | undefined;
  // Prefer the resolved display name; fall back to the policy ID from the value.
  const label = nodeMeta?.displayName || nodeMeta?.id || lastSegment(config?.alertPolicy);
  if (label) metadata.push({ icon: "bell", label });
  if (config?.displayName) metadata.push({ icon: "pencil", label: config.displayName });
  return metadata;
}
