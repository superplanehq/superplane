import { getState } from "..";
import type { ExecutionInfo, OutputPayload, RendererEventSection } from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { Incident } from "./types";

export function baseEventSections(execution: ExecutionInfo, componentName: string): RendererEventSection[] {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;

  if (!rootEvent || createdAt == null) {
    return [
      {
        receivedAt: createdAt ? new Date(createdAt) : new Date(),
        eventSubtitle: createdAt ? renderTimeAgo(new Date(createdAt)) : "",
        eventState: getState(componentName)(execution),
        eventId: execution.id ?? rootEvent?.id ?? "",
      },
    ];
  }

  return [
    {
      receivedAt: new Date(createdAt),
      eventSubtitle: renderTimeAgo(new Date(createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id ?? execution.id ?? "",
    },
  ];
}

/**
 * Extracts an incident from execution outputs with proper null checks.
 */
export function getIncidentFromExecution(execution: ExecutionInfo): Incident | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;

  if (!outputs?.default || outputs.default.length === 0) {
    return null;
  }

  return outputs.default[0].data as Incident;
}

export function getDetailsForIncident(incident: Incident | undefined): Record<string, string> {
  const details: Record<string, string> = {};

  details.ID = incident?.id || "-";
  details.Name = incident?.name || "-";

  if (incident?.number != null) {
    details.Number = String(incident.number);
  }

  details.Summary = incident?.summary || "-";
  details.Severity = incident?.severity || "-";
  details.Priority = incident?.priority || "-";

  if (incident?.current_milestone) {
    details.Milestone = incident.current_milestone;
  }
  if (incident?.incident_url) {
    details.URL = incident.incident_url;
  }

  return details;
}

/**
 * Builds execution details for FireHydrant integration components.
 */
export function buildFireHydrantExecutionDetails(execution: ExecutionInfo): Record<string, unknown> {
  const details: Record<string, unknown> = {};

  if (execution.createdAt) {
    details["Executed at"] = new Date(execution.createdAt).toLocaleString();
  }

  const incident = getIncidentFromExecution(execution);
  if (incident) {
    Object.assign(details, getDetailsForIncident(incident));
  }

  return details;
}
