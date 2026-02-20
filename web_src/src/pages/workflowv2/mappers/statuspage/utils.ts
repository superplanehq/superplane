import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo } from "../types";
import { StatuspageIncident, StatuspageIncidentUpdate } from "./types";
import { formatTimeAgo } from "@/utils/date";

export function stringOrDash(value?: string | null): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return value;
}

/** Truncates long values for display in component node meta (e.g. incident ID or expression). */
export function truncateForDisplay(value: string, maxLen = 40): string {
  if (!value || value.length <= maxLen) return value;
  return value.substring(0, maxLen) + "...";
}

/** Timeline entry format for ChainItem's isApprovalTimeline renderer. */
export type IncidentTimelineEntry = {
  label: string;
  status: string;
  timestamp?: string;
  comment?: string;
};

/** Human-readable status labels for incident_updates. */
const STATUS_LABELS: Record<string, string> = {
  investigating: "Investigating",
  identified: "Identified",
  monitoring: "Monitoring",
  resolved: "Resolved",
  scheduled: "Scheduled",
  in_progress: "In Progress",
  verifying: "Verifying",
  completed: "Completed",
};

/**
 * Maps incident status to a value that ChainItem's getApprovalStatusColor recognizes,
 * so we get colored timeline dots without modifying core UI.
 * approved=green, degraded=yellow, rejected=red, critical=red, default=gray
 */
const STATUS_TO_COLOR: Record<string, string> = {
  resolved: "Approved",
  completed: "Approved",
  investigating: "Degraded",
  identified: "Degraded",
  monitoring: "Degraded",
  in_progress: "Degraded",
  verifying: "Degraded",
  scheduled: "Scheduled", // no match → gray
};

function humanizeStatus(status?: string): string {
  if (!status) return "Update";
  return STATUS_LABELS[status] ?? status.charAt(0).toUpperCase() + status.slice(1).replace(/_/g, " ");
}

/**
 * Builds a timeline from incident_updates for the Details tab.
 * API returns incident_updates in chronological order; first is creation.
 * Uses status values that map to ChainItem's existing colors (Approved=green, Degraded=yellow).
 */
export function buildIncidentTimeline(updates: StatuspageIncidentUpdate[]): IncidentTimelineEntry[] {
  const timeline: IncidentTimelineEntry[] = [];

  for (const update of updates) {
    const rawStatus = update.status ?? "";
    const label = humanizeStatus(rawStatus);
    const statusForColor = STATUS_TO_COLOR[rawStatus] ?? label;
    const timestamp = update.created_at ? formatTimeAgo(new Date(update.created_at)) : undefined;
    const comment = update.body?.trim() || undefined;

    timeline.push({
      label,
      status: statusForColor,
      timestamp,
      comment,
    });
  }

  return timeline;
}

export type GetDetailsForIncidentOptions = {
  componentName?: string;
  execution?: ExecutionInfo;
};

/**
 * Returns human-readable execution details for a Statuspage incident.
 * API returns incident_updates in chronological order; last element is latest.
 * Timestamp and timeline vary by component: create (Created At, no timeline),
 * get (Fetched At, with timeline), update (Updated At, with timeline).
 */
export function getDetailsForIncident(
  incident: StatuspageIncident,
  options?: GetDetailsForIncidentOptions,
): Record<string, string | IncidentTimelineEntry[]> {
  const details: Record<string, string | IncidentTimelineEntry[]> = {};
  const componentName = options?.componentName;
  const execution = options?.execution;

  // Timestamp first (single timestamp per component). Always show at least one timestamp per component-design.
  if (componentName === "statuspage.createIncident" && incident?.created_at) {
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  } else if (componentName === "statuspage.getIncident" && execution?.createdAt) {
    details["Fetched At"] = new Date(execution.createdAt).toLocaleString();
  } else if (componentName === "statuspage.updateIncident" && incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  } else if (!componentName && incident?.created_at) {
    // Fallback when options not passed (e.g. from tests)
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  } else if (execution?.createdAt) {
    // Fallback: always show timestamp when incident is null or timestamp fields missing
    details["Started At"] = new Date(execution.createdAt).toLocaleString();
  }

  details["ID"] = stringOrDash(incident?.id);
  details["Name"] = stringOrDash(incident?.name);
  details["Status"] = stringOrDash(incident?.status);
  details["Impact"] = stringOrDash(incident?.impact);
  details["Incident URL"] = stringOrDash(incident?.shortlink);

  // Timeline only for get and update, omit completely for create
  if (componentName !== "statuspage.createIncident") {
    const updates = incident?.incident_updates;
    if (updates && updates.length > 0) {
      details["Timeline"] = buildIncidentTimeline(updates);
    } else {
      details["Updates"] = "No updates recorded";
    }
  }

  return details;
}

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !execution.rootEvent?.id) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
        eventState: getState(componentName)(execution),
        eventId: execution.id ?? "",
      },
    ];
  }
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  if (!rootTriggerRenderer) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
        eventState: getState(componentName)(execution),
        eventId: execution.rootEvent.id,
      },
    ];
  }
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
