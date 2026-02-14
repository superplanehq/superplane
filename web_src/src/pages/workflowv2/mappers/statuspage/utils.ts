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

/**
 * Returns human-readable execution details for a Statuspage incident.
 * API returns incident_updates in chronological order; last element is latest.
 */
export function getDetailsForIncident(incident: StatuspageIncident): Record<string, string | IncidentTimelineEntry[]> {
  const details: Record<string, string | IncidentTimelineEntry[]> = {};

  details["ID"] = stringOrDash(incident?.id);
  details["Name"] = stringOrDash(incident?.name);
  details["Status"] = stringOrDash(incident?.status);
  details["Impact"] = stringOrDash(incident?.impact);
  details["Incident URL"] = stringOrDash(incident?.shortlink);

  if (incident?.created_at) {
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  }
  if (incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  }

  const updates = incident?.incident_updates;
  if (updates && updates.length > 0) {
    details["Timeline"] = buildIncidentTimeline(updates);
  } else {
    details["Updates"] = "No updates recorded";
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
