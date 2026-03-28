import type { EventSection } from "@/ui/componentBase";
import type { ExecutionInfo, NodeInfo } from "../types";
import { getTriggerRenderer, getState } from "..";
import { renderTimeAgo } from "@/components/TimeAgo";

/**
 * Generates base event sections for GPU droplet mappers
 * @param nodes - Array of node information
 * @param execution - Execution information
 * @param componentName - Name of the component
 * @returns Array of event sections
 */
export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.rootEvent || !execution.createdAt) {
    return [];
  }

  const rootEvent = execution.rootEvent;
  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id ?? "",
    },
  ];
}
