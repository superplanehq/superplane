/* eslint-disable @typescript-eslint/no-explicit-any */
import {
  ComponentsNode,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { SidebarEvent } from "@/ui/CanvasPage";
import { formatTimeAgo } from "@/utils/date";
import { getTriggerRenderer } from "./renderers";

export function mapTriggerEventsToSidebarEvents(
  events: WorkflowsWorkflowEvent[],
  node: ComponentsNode,
  limit?: number,
): SidebarEvent[] {
  const eventsToMap = limit ? events.slice(0, limit) : events;
  return eventsToMap.map((event) => {
    const triggerRenderer = getTriggerRenderer(node.trigger?.name || "");
    const { title, subtitle } = triggerRenderer.getTitleAndSubtitle(event);
    const values = triggerRenderer.getRootEventValues(event);

    return {
      id: event.id!,
      title,
      subtitle: subtitle || formatTimeAgo(new Date(event.createdAt!)),
      state: "processed" as const,
      isOpen: false,
      receivedAt: event.createdAt ? new Date(event.createdAt) : undefined,
      values,
      triggerEventId: event.id!,
      kind: "trigger",
      nodeId: node.id,
    };
  });
}

export function mapExecutionsToSidebarEvents(
  executions: WorkflowsWorkflowNodeExecution[],
  nodes: ComponentsNode[],
  limit?: number,
): SidebarEvent[] {
  const executionsToMap = limit ? executions.slice(0, limit) : executions;
  return executionsToMap.map((execution) => {
    const state =
      execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED"
        ? ("processed" as const)
        : execution.state === "STATE_FINISHED" && execution.result === "RESULT_FAILED"
          ? ("discarded" as const)
          : execution.state === "STATE_STARTED"
            ? ("running" as const)
            : ("waiting" as const);

    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title, subtitle } = execution.rootEvent
      ? rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent)
      : {
          title: execution.id || "Execution",
          subtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)).replace(" ago", "") : "",
        };

    const values = execution.rootEvent ? rootTriggerRenderer.getRootEventValues(execution.rootEvent) : {};

    return {
      id: execution.id!,
      title,
      subtitle: subtitle || formatTimeAgo(new Date(execution.createdAt!)),
      state,
      isOpen: false,
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      values,
      executionId: execution.id!,
      kind: "execution",
      nodeId: execution?.nodeId,
    };
  });
}

export function mapQueueItemsToSidebarEvents(
  queueItems: WorkflowsWorkflowNodeQueueItem[],
  nodes: ComponentsNode[],
  limit?: number,
): SidebarEvent[] {
  const queueItemsToMap = limit ? queueItems.slice(0, limit) : queueItems;
  return queueItemsToMap.map((item) => {
    const anyItem = item as any;
    let title =
      anyItem?.name ||
      anyItem?.input?.title ||
      anyItem?.input?.name ||
      anyItem?.input?.eventTitle ||
      item.id ||
      "Queued";
    const onlyTrigger = nodes.filter((n) => n.type === "TYPE_TRIGGER");
    if (title === item.id || title === "Queued") {
      if (onlyTrigger.length === 1 && onlyTrigger[0]?.trigger?.name === "schedule") {
        title = "Event emitted by schedule";
      }
    }
    const timestamp = item.createdAt ? formatTimeAgo(new Date(item.createdAt)).replace(" ago", "") : "";
    const subtitle: string = (typeof anyItem?.input?.subtitle === "string" && anyItem.input.subtitle) || timestamp;

    return {
      id: item.id!,
      title,
      subtitle,
      state: "waiting" as const,
      isOpen: false,
      receivedAt: item.createdAt ? new Date(item.createdAt) : undefined,
      kind: "queue",
    };
  });
}
