import { CanvasesCanvasNodeExecution } from "@/api-client";
import { OutputPayload } from "../types";
import { Incident, ResourceRef } from "./types";

/**
 * Extracts an incident from execution outputs with proper null checks.
 * Returns null if outputs are missing or empty (e.g., when execution failed with an error).
 */
export function getIncidentFromExecution(execution: CanvasesCanvasNodeExecution): Incident | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return null;
  }

  return outputs.default[0].data.incident as Incident;
}

export function getDetailsForIncident(incident: Incident | undefined, agent?: ResourceRef): Record<string, string> {
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

/**
 * Builds execution details for PagerDuty incident components.
 * Includes incident details if available, and adds error in the proper format if execution failed.
 * This ensures errors are displayed as key/value pairs, not raw text.
 */
export function buildIncidentExecutionDetails(execution: CanvasesCanvasNodeExecution): Record<string, any> {
  const details: Record<string, any> = {};

  // Add execution timestamp
  if (execution.createdAt) {
    details["Executed at"] = new Date(execution.createdAt).toLocaleString();
  }

  // Add incident details if available
  const incident = getIncidentFromExecution(execution);
  if (incident) {
    Object.assign(details, getDetailsForIncident(incident));
  }

  // Add error in the proper format (if present) - placed at the end
  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
  ) {
    details["Error"] = {
      __type: "error",
      message: execution.resultMessage,
    };
  }

  return details;
}
