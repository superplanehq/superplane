import type { EventSection } from "@/pages/workflowv2/mappers/types";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getState } from "..";
import type { ExecutionInfo, NodeInfo } from "../types";

export function formatIndexingStatus(status: string): string {
  const lower = status.toLowerCase().replace(/^index_job_status_/, "");
  const map: Record<string, string> = {
    completed: "Completed",
    successful: "Successful",
    no_changes: "No changes",
    partial: "Partially completed",
    running: "Running",
    pending: "Pending",
    failed: "Failed",
    cancelled: "Cancelled",
    in_progress: "In progress",
  };
  return map[lower] ?? status;
}

export function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) return [];

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}
