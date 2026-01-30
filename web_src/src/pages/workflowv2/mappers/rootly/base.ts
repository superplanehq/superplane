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
