import type { MetadataItem } from "@/ui/metadataList";
import type { ExecutionDetailsContext, NodeInfo, OutputPayload } from "../types";
import { formatTimestamp } from "../utils";
import type {
  AlertRuleNodeMetadata,
  CreateAlertRuleConfiguration,
  GrafanaAlertRule,
  UpdateAlertRuleConfiguration,
} from "./types";

export { buildGrafanaEventSections } from "./base";

export function buildAlertRuleMetadata(
  node: NodeInfo,
  options?: {
    includeUid?: boolean;
    includeGroup?: boolean;
    includePausedState?: boolean;
  },
): MetadataItem[] {
  const configuration = node.configuration as
    | (CreateAlertRuleConfiguration & { alertRule?: string })
    | UpdateAlertRuleConfiguration
    | undefined;
  const nodeMetadata = node.metadata as AlertRuleNodeMetadata | undefined;

  const primaryItem =
    buildAlertRuleTitleItem(configuration?.title) ||
    buildAlertRuleTitleItem(nodeMetadata?.alertRuleTitle) ||
    buildAlertRuleUidItem(configuration?.alertRule, options?.includeUid);

  return [
    primaryItem,
    nodeMetadata?.folderTitle ? { icon: "folder", label: nodeMetadata.folderTitle } : undefined,
    buildPausedStateItem(configuration?.isPaused, options?.includePausedState),
    configuration?.ruleGroup && options?.includeGroup
      ? { icon: "layers-3", label: configuration.ruleGroup }
      : undefined,
  ]
    .filter(isMetadataItem)
    .slice(0, 3);
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
  const nodeMetadata = context.node.metadata as AlertRuleNodeMetadata | undefined;

  const details: Record<string, string> = {
    [actionLabel]: formatTimestamp(payload?.timestamp ?? context.execution.createdAt),
  };

  addOptionalDetail(details, "Title", alertRule.title);
  addOptionalDetail(details, "Folder", alertRule.folderTitle || nodeMetadata?.folderTitle);
  addOptionalDetail(details, "Rule Group", alertRule.ruleGroup);
  addOptionalDetail(details, "For", alertRule.for);
  addOptionalDetail(details, "Paused", formatPausedState(alertRule.isPaused));

  return details;
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
    folderTitle: asString(record.folderTitle),
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

function addOptionalDetail(details: Record<string, string>, key: string, value: string | undefined): void {
  if (value) {
    details[key] = value;
  }
}

function buildAlertRuleTitleItem(title: string | undefined): MetadataItem | undefined {
  const trimmedTitle = title?.trim();
  if (!trimmedTitle) {
    return undefined;
  }

  return { icon: "bell", label: trimmedTitle };
}

function buildAlertRuleUidItem(uid: string | undefined, includeUid?: boolean): MetadataItem | undefined {
  if (!uid || !includeUid) {
    return undefined;
  }

  return { icon: "hash", label: uid };
}

function buildPausedStateItem(isPaused: boolean | undefined, includePausedState?: boolean): MetadataItem | undefined {
  if (!includePausedState || isPaused !== true) {
    return undefined;
  }

  return {
    icon: "pause-circle",
    label: "Paused",
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
