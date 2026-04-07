import type { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import type { ExecutionInfo, NodeInfo } from "../types";
import type { StatuspageIncident } from "./types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatRelativeTime } from "@/lib/timezone";

export function stringOrDash(value?: string | null): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }
  return value;
}

/** Truncates long values for display in component node meta (e.g. incident ID or expression). */
export function truncateForDisplay(value: unknown, maxLen = 40): string {
  const str = typeof value === "string" ? value : value == null ? "" : String(value);
  if (!str || str.length <= maxLen) return str;
  return str.substring(0, maxLen) + "...";
}

export type GetDetailsForIncidentOptions = {
  componentName?: string;
  execution?: ExecutionInfo;
};

/**
 * Returns human-readable execution details for a Statuspage incident.
 * API returns incident_updates in chronological order; last element is latest.
 * Timestamp and timeline vary by component: create (Created At, no timeline),
 * get (Fetched At, with timeline), update (Updated At, with timeline).
 */
export function getDetailsForIncident(
  incident: StatuspageIncident,
  options?: GetDetailsForIncidentOptions,
): Record<string, string> {
  const details: Record<string, string> = {};
  const componentName = options?.componentName;
  const execution = options?.execution;

  // Timestamp first (single timestamp per component). Always show at least one timestamp per component-design.
  if (componentName === "statuspage.createIncident" && incident?.created_at) {
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  } else if (componentName === "statuspage.getIncident" && execution?.createdAt) {
    details["Fetched At"] = new Date(execution.createdAt).toLocaleString();
  } else if (componentName === "statuspage.updateIncident" && incident?.updated_at) {
    details["Updated At"] = new Date(incident.updated_at).toLocaleString();
  } else if (!componentName && incident?.created_at) {
    // Fallback when options not passed (e.g. from tests)
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  } else if (execution?.createdAt) {
    // Fallback: always show timestamp when incident is null or timestamp fields missing
    details["Started At"] = new Date(execution.createdAt).toLocaleString();
  }

  details["ID"] = stringOrDash(incident?.id);
  details["Name"] = stringOrDash(incident?.name);
  details["Status"] = stringOrDash(incident?.status);
  details["Impact"] = stringOrDash(incident?.impact);
  details["Incident URL"] = stringOrDash(incident?.shortlink);

  // Timeline only for get and update, omit completely for create
  if (componentName === "statuspage.createIncident") {
    return details;
  }

  const updates = incident?.incident_updates;
  if (!updates || updates.length === 0) {
    details["Updates"] = "No updates recorded";
    return details;
  }

  details["Updates"] = updates
    .map((update) => {
      const status = stringOrDash(update.status);
      const timestamp = update.created_at ? formatRelativeTime(update.created_at, true) : "-";
      const comment = stringOrDash(update.body?.trim());
      return `${status} (${timestamp}): ${comment}`;
    })
    .join(" | ");

  return details;
}

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !execution.rootEvent?.id) {
    return [
      {
        receivedAt: new Date(execution.createdAt!),
        eventTitle: "Execution",
        eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
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
        eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
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
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}
