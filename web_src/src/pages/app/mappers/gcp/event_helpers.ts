import type { EventSection } from "@/ui/componentBase";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getState, getTriggerRenderer } from "..";
import type { ExecutionInfo, NodeInfo } from "../types";

// baseEventSections builds the single event section shown on an action node from
// its most recent execution, deriving the title/subtitle from the root trigger.
// Pass eventStateOverride when a component resolves a custom event state (e.g.
// per-operation power states); otherwise the default action state is used.
export function baseEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
  eventStateOverride?: string,
): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.nodeId) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title, subtitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });
  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const fallbackSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle: subtitle || fallbackSubtitle,
      eventState: eventStateOverride ?? getState(componentName)(execution),
      eventId: rootEvent.id!,
    },
  ];
}
