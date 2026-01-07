import { Incident, ResourceRef } from "./types";

export function getDetailsForIncident(incident: Incident, agent?: ResourceRef): Record<string, string> {
  const details: Record<string, string> = {
    ID: incident?.id || "-",
    Key: incident?.incident_key || "-",
    Title: incident?.title || "-",
    Urgency: incident?.urgency || "-",
    Status: incident?.status || "-",
    "Incident URL": incident?.html_url || "-"
  };

  if (incident?.incident_number) {
    details.Number = incident.incident_number
  }

  if (incident?.service) {
    details.Service = incident?.service.summary || "-"
    details["Service URL"] = incident?.service.html_url || "-"
  }

  if (incident?.escalation_policy) {
    details["Escalation Policy"] = incident.escalation_policy.summary || "-"
    details["Escalation Policy URL"] = incident.escalation_policy.html_url || "-"
  }

  if (incident?.assignments) {
    details["Assignments"] = incident.assignments.map(i => i.assignee.summary).join(", ")
  }

  if (agent) {
    details["Agent"] = agent.summary || "-"
    details["Agent URL"] = agent.html_url || "-"
  }

  return details;
}