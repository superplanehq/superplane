import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo } from "../types";
import { StatuspageIncident } from "./types";
import { formatTimeAgo } from "@/utils/date";

export function stringOrDash(value?: string | null): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return value;
}

/**
 * Returns human-readable execution details for a Statuspage incident.
 * API returns incident_updates in chronological order; last element is latest.
 */
export function getDetailsForIncident(incident: StatuspageIncident): Record<string, string> {
  const details: Record<string, string> = {};

  details["ID"] = stringOrDash(incident?.id);
  details["Name"] = stringOrDash(incident?.name);
  details["Status"] = stringOrDash(incident?.status);
  details["Impact"] = stringOrDash(incident?.impact);
  details["Incident URL"] = stringOrDash(incident?.shortlink);

  if (incident?.created_at) {
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  }
  if (incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  }

  const updates = incident?.incident_updates;
  if (updates && updates.length > 0) {
    // API returns chronological order; last element is latest
    const latest = updates[updates.length - 1];
    if (latest?.body) {
      details["Latest Update"] = latest.body;
    }
  }

  return details;
}

export function baseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !execution.rootEvent?.id) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
        eventState: getState(componentName)(execution),
        eventId: execution.id ?? "",
      },
    ];
  }
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  if (!rootTriggerRenderer) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
        eventState: getState(componentName)(execution),
        eventId: execution.rootEvent.id,
      },
    ];
  }
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
