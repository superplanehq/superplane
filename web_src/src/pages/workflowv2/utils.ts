import {
  ComponentsNode,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { SidebarEvent } from "@/ui/CanvasPage";
import { formatTimeAgo } from "@/utils/date";
import { getTriggerRenderer } from "./mappers";

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
      originalEvent: event,
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
        : execution.state === "STATE_FINISHED" &&
            (execution.result === "RESULT_FAILED" || execution.result === "RESULT_CANCELLED")
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
      originalExecution: execution,
    };
  });
}

export function getNextInQueueInfo(
  nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]> | undefined,
  nodeId: string,
  nodes: ComponentsNode[],
): { title: string; subtitle: string; receivedAt: Date } | undefined {
  if (!nodeQueueItemsMap || !nodeQueueItemsMap[nodeId] || nodeQueueItemsMap[nodeId].length === 0) {
    return undefined;
  }

  const queueItem = nodeQueueItemsMap[nodeId]?.at(-1);
  if (!queueItem) {
    return undefined;
  }

  const rootTriggerNode = nodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

  const { title, subtitle } = queueItem.rootEvent
    ? rootTriggerRenderer.getTitleAndSubtitle(queueItem.rootEvent)
    : {
        title: queueItem.id || "Execution",
        subtitle: queueItem.createdAt ? formatTimeAgo(new Date(queueItem.createdAt)).replace(" ago", "") : "",
      };

  return {
    title,
    subtitle: subtitle || (queueItem.createdAt ? formatTimeAgo(new Date(queueItem.createdAt)) : ""),
    receivedAt: queueItem.createdAt ? new Date(queueItem.createdAt) : new Date(),
  };
}

export function mapQueueItemsToSidebarEvents(
  queueItems: WorkflowsWorkflowNodeQueueItem[],
  nodes: ComponentsNode[],
  limit?: number,
): SidebarEvent[] {
  const queueItemsToMap = limit ? queueItems.slice(0, limit) : queueItems;
  return queueItemsToMap.map((item) => {
    const rootTriggerNode = nodes.find((n) => n.id === item.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title, subtitle } = item.rootEvent
      ? rootTriggerRenderer.getTitleAndSubtitle(item.rootEvent)
      : {
          title: item.id || "Execution",
          subtitle: item.createdAt ? formatTimeAgo(new Date(item.createdAt)).replace(" ago", "") : "",
        };

    const values = item.rootEvent ? rootTriggerRenderer.getRootEventValues(item.rootEvent) : {};

    return {
      id: item.id!,
      title,
      subtitle: subtitle || formatTimeAgo(new Date(item.createdAt!)),
      state: "waiting" as const,
      isOpen: false,
      receivedAt: item.createdAt ? new Date(item.createdAt) : undefined,
      kind: "queue",
      values,
    };
  });
}
