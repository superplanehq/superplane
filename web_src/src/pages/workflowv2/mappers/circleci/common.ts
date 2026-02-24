import { ExecutionInfo, NodeInfo } from "../types";
import { EventSection } from "@/ui/componentBase";
import { getState, getTriggerRenderer } from "..";
import { formatTimeAgo } from "@/utils/date";

/**
 * Builds the event sections for a CircleCI read-only component.
 *
 * Shared across all read-only CircleCI mapper files (getWorkflow, getLastWorkflow,
 * getRecentWorkflowRuns, getTestMetrics, getFlakyTests) which use the same logic
 * to render a single event section from the root trigger.
 */
export function getEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  const rootTriggerNode = nodes.find((n) => n.id === rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");

  const title = rootEvent
    ? rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent }).title
    : "Event received";

  const receivedAt = execution.createdAt ? new Date(execution.createdAt) : new Date(0);

  return [
    {
      receivedAt,
      eventTitle: title,
      eventSubtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)) : "",
      eventState: getState(componentName)(execution),
      eventId: rootEvent?.id ?? "",
    },
  ];
}
