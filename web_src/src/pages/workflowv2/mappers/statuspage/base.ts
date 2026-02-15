import { CanvasesCanvasNodeExecution } from "@/api-client";
import { OutputPayload } from "../types";
import { StatuspageIncident } from "./types";

/**
 * Extracts an incident from execution outputs with proper null checks.
 * Returns null if outputs are missing or empty (e.g., when execution failed with an error).
 */
export function getIncidentFromExecution(execution: CanvasesCanvasNodeExecution): StatuspageIncident | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return null;
  }

  return outputs.default[0].data as StatuspageIncident;
}

export function getDetailsForIncident(incident: StatuspageIncident): Record<string, string> {
  const details: Record<string, string> = {};

  Object.assign(details, {
    "Created At": incident?.created_at ? new Date(incident.created_at).toLocaleString() : "-",
    "Updated At": incident?.updated_at ? new Date(incident.updated_at).toLocaleString() : "-",
  });

  details.ID = incident?.id || "-";
  details.Name = incident?.name || "-";
  details.Status = incident?.status || "-";
  details.Impact = incident?.impact || "-";

  if (incident?.shortlink) {
    details["Incident URL"] = incident.shortlink;
  }

  if (incident?.resolved_at) {
    details["Resolved At"] = new Date(incident.resolved_at).toLocaleString();
  }

  if (incident?.components && incident.components.length > 0) {
    details["Affected Components"] = incident.components.map((c) => c.name).join(", ");
  }

  return details;
}

/**
 * Builds execution details for Statuspage incident components.
 * Includes incident details if available, and adds error in the proper format if execution failed.
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
