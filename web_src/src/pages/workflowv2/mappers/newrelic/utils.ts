import type { EventSection } from "@/ui/componentBase";
import { getState } from "..";
import type { ExecutionInfo, NodeInfo } from "../types";
import { renderTimeAgo } from "@/components/TimeAgo";

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  if (!rootTriggerNode) {
    return [];
  }

  return [
    {
      receivedAt: new Date(execution.createdAt || Date.now()),
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt || Date.now())),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id || "",
    },
  ];
}
