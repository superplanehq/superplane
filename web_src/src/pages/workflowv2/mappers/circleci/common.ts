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
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: formatTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}
