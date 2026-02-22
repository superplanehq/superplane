import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import { CanvasesCanvasNodeExecution } from "@/api-client";
import { formatTimeAgo } from "@/utils/date";
import { IncidentRecord, STATE_LABELS, URGENCY_LABELS, IMPACT_LABELS } from "./types";

export function getIncidentFromExecution(execution: CanvasesCanvasNodeExecution): IncidentRecord | null {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  if (!outputs || !outputs.default || outputs.default.length === 0) {
    return null;
  }
  return outputs.default[0].data as IncidentRecord;
}

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

export function buildIncidentExecutionDetails(
  execution: CanvasesCanvasNodeExecution,
  instanceUrl?: string,
): Record<string, any> {
  const details: Record<string, any> = {};
  if (execution.createdAt) {
    details["Executed at"] = new Date(execution.createdAt).toLocaleString();
  }
  const incident = getIncidentFromExecution(execution);
  if (incident) {
    if (incident.number) details["Number"] = incident.number;
    if (incident.sys_id) {
      if (instanceUrl) {
        details["Incident URL"] = `${instanceUrl}/incident.do?sys_id=${incident.sys_id}`;
      } else {
        details["Sys ID"] = incident.sys_id;
      }
    }
    if (incident.short_description) details["Short Description"] = incident.short_description;
    if (incident.state) details["State"] = STATE_LABELS[incident.state] || incident.state;
    if (incident.urgency) details["Urgency"] = URGENCY_LABELS[incident.urgency] || incident.urgency;
    if (incident.impact) details["Impact"] = IMPACT_LABELS[incident.impact] || incident.impact;
    if (incident.sys_created_on) details["Created On"] = incident.sys_created_on;
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

export function instanceUrlToLabel(instanceUrl: string): string {
  return instanceUrl.replace(/^https?:\/\//, "").replace(/\.service-now\.com$/, "");
}
