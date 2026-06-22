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
import monitoringIcon from "@/assets/icons/integrations/gcp.monitoring.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import { baseEventSections } from "./event_helpers";

interface AlertConditionConfig {
  metricType?: string;
  comparison?: string;
  threshold?: number;
}

interface CreateAlertingPolicyConfiguration {
  displayName?: string;
  conditions?: AlertConditionConfig[];
  severity?: string;
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
  severity?: string;
  conditionsCount?: number;
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

export function subtitle(context: SubtitleContext): string | React.ReactNode {
  const timestamp = context.execution.updatedAt || context.execution.createdAt;
  return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
}

export function baseProps(
  context: ComponentBaseContext,
  iconSlug: string,
  fallbackTitle: string,
  metadata: MetadataItem[],
): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name ?? "gcp";

  return {
    iconSrc: monitoringIcon,
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

function getPolicyOutput(context: ExecutionDetailsContext): AlertingPolicyOutputData | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as AlertingPolicyOutputData | undefined;
}

interface PolicyDetailsOptions {
  includeId?: boolean;
  includeFirstCondition?: boolean;
}

function policyDetails(
  context: ExecutionDetailsContext,
  { includeId = true, includeFirstCondition = true }: PolicyDetailsOptions = {},
): Record<string, string> {
  const details: Record<string, string> = {};
  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }
  const result = getPolicyOutput(context);
  if (!result) return details;

  if (result.displayName) details["Display Name"] = result.displayName;
  if (includeId && result.id) details["Policy ID"] = result.id;
  if (result.enabled !== undefined) details["Enabled"] = result.enabled ? "Yes" : "No";
  if (result.severity) details["Severity"] = result.severity;
  if (result.conditionsCount !== undefined) details["Conditions"] = String(result.conditionsCount);
  if (includeFirstCondition && result.comparison && result.thresholdValue !== undefined) {
    details["First Condition"] = `${comparisonLabels[result.comparison] || result.comparison} ${result.thresholdValue}`;
  }
  if (result.duration) details["Duration"] = result.duration;
  return details;
}

export const createAlertingPolicyMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context, "bell", "Create Alerting Policy", createMetadata(context.node));
  },
  getExecutionDetails: (context) => policyDetails(context, { includeId: false }),
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
  getExecutionDetails: (context) => policyDetails(context, { includeId: false, includeFirstCondition: false }),
  subtitle,
};

function conditionSummary(conditions: AlertConditionConfig[]): string | undefined {
  const first = conditions[0];
  if (!first?.metricType) {
    return conditions.length > 0 ? `${conditions.length} condition${conditions.length > 1 ? "s" : ""}` : undefined;
  }
  const label = metricLabels[first.metricType] || first.metricType;
  const cmp = first.comparison ? comparisonLabels[first.comparison] || "" : "";
  const suffix = cmp && first.threshold !== undefined ? ` ${cmp} ${first.threshold}` : "";
  const more = conditions.length > 1 ? ` +${conditions.length - 1}` : "";
  return `${label}${suffix}${more}`;
}

function createMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as CreateAlertingPolicyConfiguration | undefined;
  if (config?.displayName) metadata.push({ icon: "bell", label: config.displayName });

  const summary = conditionSummary(config?.conditions ?? []);
  if (summary) metadata.push({ icon: "chart-line", label: summary });

  if (config?.severity) metadata.push({ icon: "triangle-alert", label: config.severity });
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

// --- Snooze helpers (mappers live in create_snooze.ts / get_snooze.ts / expire_snooze.ts) ---

interface SnoozeOutputData {
  name?: string;
  id?: string;
  displayName?: string;
  policiesCount?: number;
  startTime?: string;
  endTime?: string;
}

interface CreateSnoozeConfiguration {
  displayName?: string;
  policies?: string[];
  duration?: string;
}

interface SnoozeSelectorConfiguration {
  snooze?: string;
}

// Resolved at Setup time by the backend so the collapsed node can show the
// snooze's display name instead of its numeric ID.
interface SnoozeNodeMetadata {
  snoozeName?: string;
  displayName?: string;
  id?: string;
}

export function snoozeDetails(context: ExecutionDetailsContext): Record<string, string> {
  const details: Record<string, string> = {};
  if (context.execution.createdAt) {
    details["Executed At"] = new Date(context.execution.createdAt).toLocaleString();
  }
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  const out = outputs?.default?.[0]?.data as SnoozeOutputData | undefined;
  if (!out) return details;

  if (out.displayName) details["Display Name"] = out.displayName;
  if (out.policiesCount !== undefined) details["Policies"] = String(out.policiesCount);
  if (out.startTime) details["Start"] = new Date(out.startTime).toLocaleString();
  if (out.endTime) details["End"] = new Date(out.endTime).toLocaleString();
  return details;
}

export function snoozeCreateMetadata(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const config = node.configuration as CreateSnoozeConfiguration | undefined;
  if (config?.displayName) metadata.push({ icon: "bell-off", label: config.displayName });
  const count = config?.policies?.length ?? 0;
  if (count > 0) metadata.push({ icon: "bell", label: `${count} ${count > 1 ? "policies" : "policy"}` });
  if (config?.duration) metadata.push({ icon: "clock", label: config.duration });
  return metadata;
}

export function snoozeSelectorMetadata(node: NodeInfo): MetadataItem[] {
  const config = node.configuration as SnoozeSelectorConfiguration | undefined;
  const nodeMeta = node.metadata as SnoozeNodeMetadata | undefined;
  // Prefer the resolved display name; fall back to the snooze ID from the value.
  const label = nodeMeta?.displayName || nodeMeta?.id || lastSegment(config?.snooze);
  return label ? [{ icon: "bell-off", label }] : [];
}
