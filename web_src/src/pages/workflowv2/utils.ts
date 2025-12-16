import {
  ComponentsNode,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { SidebarEvent } from "@/ui/CanvasPage";
import { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import { getComponentBaseMapper, getState, getTriggerRenderer } from "./mappers";

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
      state: "triggered" as const,
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
  additionalData?: unknown,
): SidebarEvent[] {
  const executionsToMap = limit ? executions.slice(0, limit) : executions;

  return executionsToMap.map((execution) => {
    const currentComponentNode = nodes.find((n) => n.id === execution.nodeId);
    const stateResolver = getState(currentComponentNode?.component?.name || "");
    const state = stateResolver(execution);
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const componentSubtitle = getComponentBaseMapper(currentComponentNode?.component?.name || "").subtitle?.(
      currentComponentNode as ComponentsNode,
      execution,
      additionalData,
    );

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
      subtitle: componentSubtitle || subtitle || formatTimeAgo(new Date(execution.createdAt!)),
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
      state: "queued" as const,
      isOpen: false,
      receivedAt: item.createdAt ? new Date(item.createdAt) : undefined,
      kind: "queue",
      values,
    };
  });
}

export function buildTabData(
  nodeId: string,
  event: SidebarEvent,
  options: {
    workflowNodes: ComponentsNode[];
    nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>;
    nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>;
    nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>;
  }
): TabData | undefined {
  const { workflowNodes, nodeEventsMap, nodeExecutionsMap, nodeQueueItemsMap } = options;
  const node = workflowNodes.find((n) => n.id === nodeId);
  if (!node) return undefined;

  if (node.type === "TYPE_TRIGGER") {
    const events = nodeEventsMap[nodeId] || [];
    const triggerEvent = events.find((evt) => evt.id === event.id);

    if (!triggerEvent) return undefined;

    const tabData: TabData = {};
    const triggerRenderer = getTriggerRenderer(node.trigger?.name || "");

    const eventValues = triggerRenderer.getRootEventValues(triggerEvent);

    tabData.current = {
      ...eventValues,
      "Event ID": triggerEvent.id,
      "Node ID": triggerEvent.nodeId,
      "Created At": triggerEvent.createdAt ? new Date(triggerEvent.createdAt).toLocaleString() : undefined,
    };

    // Payload tab: raw event data
    let payload: Record<string, unknown> = {};

    if (triggerEvent.data) {
      payload = triggerEvent.data;
    }

    tabData.payload = payload;

    return Object.keys(tabData).length > 0 ? tabData : undefined;
  }

  if (event.kind === "queue") {
    // Handle queue items - get the queue item data
    const queueItems = nodeQueueItemsMap[nodeId] || [];
    const queueItem = queueItems.find((item: WorkflowsWorkflowNodeQueueItem) => item.id === event.id);

    if (!queueItem) return undefined;

    const tabData: TabData = {};

    if (queueItem.rootEvent) {
      const rootTriggerNode = workflowNodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
      const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
      const rootEventValues = rootTriggerRenderer.getRootEventValues(queueItem.rootEvent);

      tabData.root = {
        ...rootEventValues,
        "Event ID": queueItem.rootEvent.id,
        "Node ID": queueItem.rootEvent.nodeId,
        "Created At": queueItem.rootEvent.createdAt
          ? new Date(queueItem.rootEvent.createdAt).toLocaleString()
          : undefined,
      };
    }

    tabData.current = {
      "Queue Item ID": queueItem.id,
      "Node ID": queueItem.nodeId,
      "Created At": queueItem.createdAt ? new Date(queueItem.createdAt).toLocaleString() : undefined,
    };

    tabData.payload = queueItem.input || {};

    return Object.keys(tabData).length > 0 ? tabData : undefined;
  }

  // Handle other components (non-triggers) - get execution for this event
  const executions = nodeExecutionsMap[nodeId] || [];
  const execution = executions.find((exec: WorkflowsWorkflowNodeExecution) => exec.id === event.id);

  if (!execution) return undefined;

  // Extract tab data from execution
  const tabData: TabData = {};

  // Current tab: use outputs if available and non-empty, otherwise use metadata
  const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
  const dataSource = hasOutputs ? execution.outputs : execution.metadata || {};
  const flattened = flattenObject(dataSource);

  const currentData = {
    ...flattened,
    "Execution ID": execution.id,
    "Execution State": execution.state?.replace("STATE_", "").toLowerCase(),
    "Execution Result": execution.result?.replace("RESULT_", "").toLowerCase(),
    "Execution Started": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : undefined,
  };

  // Filter out undefined and empty values
  tabData.current = Object.fromEntries(
    Object.entries(currentData).filter(([_, value]) => value !== undefined && value !== "" && value !== null),
  );

  // Root tab: root event data
  if (execution.rootEvent) {
    const rootTriggerNode = workflowNodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
    const rootEventValues = rootTriggerRenderer.getRootEventValues(execution.rootEvent);

    tabData.root = {
      ...rootEventValues,
      "Event ID": execution.rootEvent.id,
      "Node ID": execution.rootEvent.nodeId,
      "Created At": execution.rootEvent.createdAt
        ? new Date(execution.rootEvent.createdAt).toLocaleString()
        : undefined,
    };
  }

  // Payload tab: execution inputs and outputs (raw data)
  let payload: Record<string, unknown> = {};

  if (execution.outputs) {
    const outputData: unknown[] = Object.values(execution.outputs)?.find((output) => {
      return Array.isArray(output) && output?.length > 0;
    }) as unknown[];

    if (outputData?.length > 0) {
      payload = outputData?.[0] as Record<string, unknown>;
    }
  }

  tabData.payload = payload;

  return Object.keys(tabData).length > 0 ? tabData : undefined;
}
