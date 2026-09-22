import type { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "../mapperLookup";
import type { ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { Incident } from "./types";

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;

  if (!rootEvent || createdAt == null) {
    return [
      {
        receivedAt: createdAt ? new Date(createdAt) : new Date(),
        eventTitle: "Event",
        eventSubtitle: createdAt ? renderTimeAgo(new Date(createdAt)) : "",
        eventState: getState(componentName)(execution),
        eventId: execution.id ?? rootEvent?.id ?? "",
      },
    ];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(createdAt),
      eventTitle: title,
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

function incidentDashDefaultFields(incident: Incident | undefined): Record<string, string> {
  return {
    ID: incident?.id || "-",
    Name: incident?.name || "-",
    Summary: incident?.summary || "-",
    Severity: incident?.severity || "-",
    Priority: incident?.priority || "-",
  };
}

export function getDetailsForIncident(incident: Incident | undefined): Record<string, string> {
  const { ID, Name, Summary, Severity, Priority } = incidentDashDefaultFields(incident);
  const details: Record<string, string> = { ID, Name };

  if (incident?.number != null) {
    details.Number = String(incident.number);
  }

  details.Summary = Summary;
  details.Severity = Severity;
  details.Priority = Priority;

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
