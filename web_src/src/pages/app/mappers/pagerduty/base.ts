import type { ExecutionInfo, OutputPayload } from "../types";
import type { Incident, ResourceRef } from "./types";

/**
 * Extracts an incident from execution outputs with proper null checks.
 * Returns null if outputs are missing or empty (e.g., when execution failed with an error).
 */
export function getIncidentFromExecution(execution: ExecutionInfo): Incident | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return null;
  }

  const payload = outputs.default[0].data as { incident?: Incident } | undefined;
  return payload?.incident ?? null;
}

export function getDetailsForIncident(incident: Incident | undefined, agent?: ResourceRef): Record<string, string> {
  const details: Record<string, string> = {};
  const record = incident ?? ({} as Incident);
  const createdAt = formatIncidentTime(record.created_at);
  details["Created At"] = createdAt;
  const updatedAt = formatIncidentTime(record.updated_at);
  details["Updated At"] = updatedAt;
  const id = textOrDash(record.id);
  details.ID = id;
  const key = textOrDash(record.incident_key);
  details.Key = key;
  const title = textOrDash(record.title);
  details.Title = title;
  const urgency = textOrDash(record.urgency);
  details.Urgency = urgency;
  const status = textOrDash(record.status);
  details.Status = status;
  const htmlUrl = textOrDash(record.html_url);
  details["Incident URL"] = htmlUrl;
  const incidentNumber = record.incident_number;
  if (incidentNumber) {
    details.Number = incidentNumber;
  }

  assignIncidentRelatedResources(details, incident, agent);

  const lastStatusChangeAt = record.last_status_change_at;
  if (lastStatusChangeAt) {
    details["Last Status Change"] = new Date(lastStatusChangeAt).toLocaleString();
  }

  const resolvedAt = record.resolved_at;
  if (resolvedAt) {
    details["Resolved At"] = new Date(resolvedAt).toLocaleString();
  }

  return details;
}

function textOrDash(value: string | undefined): string {
  return value || "-";
}

function formatIncidentTime(value: string | undefined): string {
  return value ? new Date(value).toLocaleString() : "-";
}

function assignIncidentRelatedResources(
  details: Record<string, string>,
  incident: Incident | undefined,
  agent?: ResourceRef,
) {
  if (incident?.service) {
    details.Service = incident.service.summary || "-";
    details["Service URL"] = incident.service.html_url || "-";
  }

  if (incident?.escalation_policy) {
    details["Escalation Policy"] = incident.escalation_policy.summary || "-";
    details["Escalation Policy URL"] = incident.escalation_policy.html_url || "-";
  }

  if (incident?.assignments) {
    details["Assignments"] = incident.assignments.map((i) => i.assignee.summary).join(", ");
  }

  if (agent) {
    details["Agent"] = agent.summary || "-";
    details["Agent URL"] = agent.html_url || "-";
  }
}

/**
 * Builds execution details for PagerDuty incident components.
 * Includes incident details if available, and adds error in the proper format if execution failed.
 * This ensures errors are displayed as key/value pairs, not raw text.
 */
export function buildIncidentExecutionDetails(execution: ExecutionInfo): Record<string, string> {
  const details: Record<string, string> = {};

  // Add execution timestamp
  if (execution.createdAt) {
    details["Executed at"] = new Date(execution.createdAt).toLocaleString();
  }

  // Add incident details if available
  const incident = getIncidentFromExecution(execution);
  if (incident) {
    Object.assign(details, getDetailsForIncident(incident));
  }

  return details;
}
