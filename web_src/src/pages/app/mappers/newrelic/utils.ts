import type { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import type { ExecutionInfo, NodeInfo } from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt || Date.now()),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt || Date.now())),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
