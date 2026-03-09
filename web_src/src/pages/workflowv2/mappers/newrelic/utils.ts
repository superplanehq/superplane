import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { ExecutionInfo, NodeInfo } from "../types";
import { formatTimeAgo } from "@/utils/date";

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
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt || Date.now())),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
