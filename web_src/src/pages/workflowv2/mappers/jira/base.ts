import type { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import type { ExecutionInfo, NodeInfo } from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";

export function jiraBaseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : new Date();
  const subtitleDate = execution.updatedAt ?? execution.createdAt;
  const eventSubtitle = subtitleDate ? renderTimeAgo(new Date(subtitleDate)) : "";
  const eventState = getState(componentName)(execution);

  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode || !execution.rootEvent?.id) {
    return [{ receivedAt, eventTitle: "Execution", eventSubtitle, eventState, eventId: execution.id ?? "" }];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  if (!rootTriggerRenderer) {
    return [{ receivedAt, eventTitle: "Execution", eventSubtitle, eventState, eventId: execution.rootEvent.id }];
  }

  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  return [{ receivedAt, eventTitle: title, eventSubtitle, eventState, eventId: execution.rootEvent.id }];
}
