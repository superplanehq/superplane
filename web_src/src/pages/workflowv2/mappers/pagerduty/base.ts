import { Incident, ResourceRef } from "./types";

export function getDetailsForIncident(incident: Incident, agent?: ResourceRef): Record<string, string> {
  const details: Record<string, string> = {};
  Object.assign(details, {
    "Created At": incident?.created_at ? new Date(incident.created_at).toLocaleString() : "-",
    "Updated At": incident?.updated_at ? new Date(incident.updated_at).toLocaleString() : "-",
  });

  details.ID = incident?.id || "-";
  details.Key = incident?.incident_key || "-";
  details.Title = incident?.title || "-";
  details.Urgency = incident?.urgency || "-";
  details.Status = incident?.status || "-";
  details["Incident URL"] = incident?.html_url || "-";

  if (incident?.incident_number) {
    details.Number = incident.incident_number;
  }

  if (incident?.service) {
    details.Service = incident?.service.summary || "-";
    details["Service URL"] = incident?.service.html_url || "-";
  }

  if (incident?.escalation_policy) {
    details["Escalation Policy"] = incident.escalation_policy.summary || "-";
    details["Escalation Policy URL"] = incident.escalation_policy.html_url || "-";
  }

  if (incident?.assignments) {
    details["Assignments"] = incident.assignments.map((i) => i.assignee.summary).join(", ");
  }

  if (incident?.last_status_change_at) {
    details["Last Status Change"] = new Date(incident.last_status_change_at).toLocaleString();
  }

  if (incident?.resolved_at) {
    details["Resolved At"] = new Date(incident.resolved_at).toLocaleString();
  }

  if (agent) {
    details["Agent"] = agent.summary || "-";
    details["Agent URL"] = agent.html_url || "-";
  }

  return details;
}
