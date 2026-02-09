import { formatTimeAgo } from "@/utils/date";
import { Incident } from "./types";

export function getDetailsForIncident(incident: Incident): Record<string, string> {
  const details: Record<string, string> = {};

  details.ID = incident?.id || "-";
  details.Title = incident?.title || "-";
  details.Summary = incident?.summary || "-";
  details.Status = incident?.status || "-";
  details.Severity = incident?.severity?.name || "-";

  if (incident?.started_at) {
    details["Started At"] = new Date(incident.started_at).toLocaleString();
  }

  if (incident?.mitigated_at) {
    details["Mitigated At"] = new Date(incident.mitigated_at).toLocaleString();
  }

  if (incident?.resolved_at) {
    details["Resolved At"] = new Date(incident.resolved_at).toLocaleString();
  }

  if (incident?.url) {
    details["Incident URL"] = incident.url;
  }

  return details;
}

// Map event values to display labels (matching backend configuration)
export const eventLabels: Record<string, string> = {
  "incident.created": "Created",
  "incident.updated": "Updated",
  "incident.mitigated": "Mitigated",
  "incident.resolved": "Resolved",
  "incident.cancelled": "Cancelled",
  "incident.deleted": "Deleted",
};

export function formatEventLabel(event: string): string {
  return (
    eventLabels[event] ||
    event.replace("incident.", "").charAt(0).toUpperCase() + event.replace("incident.", "").slice(1)
  );
}

export function buildSubtitle(content: string, createdAt?: string): string {
  const timeAgo = createdAt ? formatTimeAgo(new Date(createdAt)) : "";
  if (content && timeAgo) {
    return `${content} · ${timeAgo}`;
  }

  return content || timeAgo;
}
