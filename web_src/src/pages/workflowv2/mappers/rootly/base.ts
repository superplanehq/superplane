import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { Incident, IncidentEvent } from "./types";

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent! });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

export function getDetailsForIncident(incident: Incident | undefined): Record<string, string> {
  const details: Record<string, string> = {};

  details.ID = incident?.id || "-";
  details.Title = incident?.title || "-";
  details.Summary = incident?.summary || "-";
  details.Status = incident?.status || "-";
  details.Severity = incident?.severity || "-";

  if (incident?.started_at) {
    details["Started At"] = new Date(incident.started_at).toLocaleString();
  }

  if (incident?.mitigated_at) {
    details["Mitigated At"] = new Date(incident.mitigated_at).toLocaleString();
  }

  if (incident?.resolved_at) {
    details["Resolved At"] = new Date(incident.resolved_at).toLocaleString();
  }

  if (incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  }

  if (incident?.url) {
    details["Incident URL"] = incident.url;
  }

  return details;
}

export function getDetailsForIncidentEvent(incidentEvent: IncidentEvent): Record<string, string> {
  const details: Record<string, string> = {};

  if (incidentEvent?.created_at) {
    details["Created At"] = new Date(incidentEvent.created_at).toLocaleString();
  }

  if (incidentEvent?.id) {
    details["Event ID"] = incidentEvent.id;
  }

  if (incidentEvent?.event) {
    details["Event"] = incidentEvent.event;
  }

  if (incidentEvent?.visibility) {
    details["Visibility"] = incidentEvent.visibility;
  }

  return details;
}
