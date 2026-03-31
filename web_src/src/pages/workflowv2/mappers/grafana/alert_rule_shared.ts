import type { EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatOptionalIsoTimestamp } from "@/lib/timezone";
import { getState, getTriggerRenderer } from "..";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import type { CreateAlertRuleConfiguration, GrafanaAlertRule, UpdateAlertRuleConfiguration } from "./types";

export function buildAlertRuleMetadata(
  node: NodeInfo,
  options?: {
    includeUid?: boolean;
    includeGroup?: boolean;
    includePausedState?: boolean;
  },
): MetadataItem[] {
  const configuration = node.configuration as
    | (CreateAlertRuleConfiguration & { alertRuleUid?: string })
    | UpdateAlertRuleConfiguration
    | undefined;

  const primaryItem =
    configuration?.title != null
      ? { icon: "bell", label: configuration.title }
      : buildAlertRuleUidItem(configuration?.alertRuleUid, options?.includeUid);

  return [
    primaryItem,
    configuration?.folderUID ? { icon: "folder", label: configuration.folderUID } : undefined,
    configuration?.ruleGroup && options?.includeGroup
      ? { icon: "layers-3", label: configuration.ruleGroup }
      : undefined,
    buildPausedStateItem(configuration?.isPaused, options?.includePausedState),
  ].filter(isMetadataItem);
}

export function buildAlertRuleExecutionDetails(
  context: ExecutionDetailsContext,
  actionLabel: string,
): Record<string, string> {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;

  if (!outputs?.default?.length) {
    return { Response: "No data returned" };
  }

  const payload = outputs.default[0];
  const alertRule = asAlertRule(payload?.data);
  if (!alertRule) {
    return { Response: "No data returned" };
  }

  const details: Record<string, string> = {
    [actionLabel]: formatOptionalIsoTimestamp(payload?.timestamp ?? context.execution.createdAt),
  };

  addOptionalDetail(details, "Title", alertRule.title);
  addOptionalDetail(details, "UID", alertRule.uid);
  addOptionalDetail(details, "Folder", alertRule.folderUID);
  addOptionalDetail(details, "Rule Group", alertRule.ruleGroup);
  addOptionalDetail(details, "Condition", alertRule.condition);
  addOptionalDetail(details, "For", alertRule.for);
  addOptionalDetail(details, "No Data State", alertRule.noDataState);
  addOptionalDetail(details, "Exec Error State", alertRule.execErrState);
  addOptionalDetail(details, "Queries", alertRule.data ? String(alertRule.data.length) : undefined);
  addOptionalDetail(details, "Labels", formatOptionalRecord(alertRule.labels));
  addOptionalDetail(details, "Annotations", formatOptionalRecord(alertRule.annotations));
  addOptionalDetail(details, "Paused", formatPausedState(alertRule.isPaused));

  return details;
}

export function buildGrafanaEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  if (!execution.rootEvent?.id || !execution.createdAt) {
    return [];
  }

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode?.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title || "Trigger event",
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}

export function asAlertRule(value: unknown): GrafanaAlertRule | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }

  const record = value as Record<string, unknown>;
  return {
    uid: asString(record.uid),
    title: asString(record.title),
    folderUID: asString(record.folderUID),
    ruleGroup: asString(record.ruleGroup),
    condition: asString(record.condition),
    noDataState: asString(record.noDataState),
    execErrState: asString(record.execErrState),
    for: asString(record.for),
    isPaused: typeof record.isPaused === "boolean" ? record.isPaused : undefined,
    labels: asStringRecord(record.labels),
    annotations: asStringRecord(record.annotations),
    data: Array.isArray(record.data) ? record.data.filter(isRecord) : undefined,
  };
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() !== "" ? value : undefined;
}

function asStringRecord(value: unknown): Record<string, string> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }

  const record = value as Record<string, unknown>;
  const entries = Object.entries(record)
    .filter(([, entryValue]) => typeof entryValue === "string")
    .map(([key, entryValue]) => [key, entryValue as string]);

  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}

function formatRecord(value: Record<string, string>): string {
  return Object.entries(value)
    .map(([key, entryValue]) => `${key}=${entryValue}`)
    .join(", ");
}

function formatOptionalRecord(value: Record<string, string> | undefined): string | undefined {
  return value && Object.keys(value).length > 0 ? formatRecord(value) : undefined;
}

function addOptionalDetail(details: Record<string, string>, key: string, value: string | undefined): void {
  if (value) {
    details[key] = value;
  }
}

function buildAlertRuleUidItem(uid: string | undefined, includeUid?: boolean): MetadataItem | undefined {
  if (!uid || !includeUid) {
    return undefined;
  }

  return { icon: "hash", label: uid };
}

function buildPausedStateItem(isPaused: boolean | undefined, includePausedState?: boolean): MetadataItem | undefined {
  if (!includePausedState || isPaused === undefined) {
    return undefined;
  }

  return {
    icon: isPaused ? "pause-circle" : "play-circle",
    label: isPaused ? "Paused" : "Active",
  };
}

function formatPausedState(isPaused: boolean | undefined): string | undefined {
  if (isPaused === undefined) {
    return undefined;
  }

  return isPaused ? "Yes" : "No";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function isMetadataItem(value: MetadataItem | undefined): value is MetadataItem {
  return value !== undefined;
}
