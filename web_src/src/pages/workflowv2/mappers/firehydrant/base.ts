import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import { formatTimeAgo } from "@/utils/date";
import { Incident } from "./types";

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  const createdAt = execution.createdAt;

  if (!rootEvent || createdAt == null) {
    return [
      {
        receivedAt: createdAt ? new Date(createdAt) : new Date(),
        eventTitle: "Event",
        eventSubtitle: createdAt ? formatTimeAgo(new Date(createdAt)) : "",
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
      eventSubtitle: formatTimeAgo(new Date(createdAt)),
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

  if (incident?.created_at) {
    details["Created At"] = new Date(incident.created_at).toLocaleString();
  }

  if (incident?.started_at) {
    details["Started At"] = new Date(incident.started_at).toLocaleString();
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
