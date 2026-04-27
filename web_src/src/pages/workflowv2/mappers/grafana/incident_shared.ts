import type { ExecutionDetailsContext, OutputPayload } from "../types";
import type { MetadataItem } from "@/ui/metadataList";
import type { GrafanaIncident, GrafanaIncidentNodeMetadata } from "./types";

export type Details = Record<string, string>;

export function buildIncidentSelectionMetadata(
  nodeMetadata: GrafanaIncidentNodeMetadata | undefined,
  incident: string | undefined,
): MetadataItem[] {
  const label = nodeMetadata?.label || nodeMetadata?.title || incident;
  if (!label) {
    return [];
  }

  return [{ icon: "alert-triangle", label: `Incident: ${truncate(label, 70)}` }];
}

export function buildIncidentDetails(
  context: ExecutionDetailsContext,
  incident: GrafanaIncident | undefined,
  timestampLabel: string,
  keys: string[],
): Details {
  const details: Details = {
    [timestampLabel]: formatDetailTimestamp(getFirstOutputTimestamp(context), context.execution.createdAt),
  };

  if (!incident) {
    return details;
  }

  const labels = formatIncidentLabels(incident);
  const values: Details = {
    Title: incident.title || "",
    Severity: incident.severity || "",
    Status: incident.status || "",
    Labels: labels || "",
    "Created At": formatDetailTimestamp(incident.createdTime),
    "Modified At": formatDetailTimestamp(incident.modifiedTime),
    "Closed At": formatDetailTimestamp(incident.closedTime),
    "Incident URL": incident.incidentUrl || incident.overviewURL || "",
  };

  for (const key of keys) {
    addIfPresent(details, key, values[key]);
  }

  return limitDetails(details, 6);
}

export function getFirstOutputData<T>(context: ExecutionDetailsContext): T | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data as T | undefined;
}

export function getFirstOutputTimestamp(context: ExecutionDetailsContext): string | undefined {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.timestamp;
}

export function addIfPresent(details: Details, key: string, value: string | undefined): void {
  if (value && value !== "-") {
    details[key] = value;
  }
}

export function limitDetails(details: Details, maxItems: number): Details {
  return Object.fromEntries(Object.entries(details).slice(0, maxItems));
}

export function formatDetailTimestamp(value?: string, fallback?: string): string {
  const timestamp = value || fallback;
  if (!timestamp) {
    return "-";
  }

  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  return date.toLocaleString();
}

export function truncate(value: string | undefined, maxLength: number): string | undefined {
  if (!value) {
    return undefined;
  }

  return value.length > maxLength ? value.substring(0, maxLength).trimEnd() + "..." : value;
}

function formatIncidentLabels(incident: GrafanaIncident): string | undefined {
  if (!Array.isArray(incident.labels) || incident.labels.length === 0) {
    return undefined;
  }

  const labels = incident.labels
    .map((label) => label.label || label.key)
    .filter((label): label is string => Boolean(label));

  if (labels.length === 0) {
    return undefined;
  }

  const visible = labels.slice(0, 3);
  const remaining = labels.length - visible.length;
  return visible.join(", ") + (remaining > 0 ? ` +${remaining}` : "");
}
