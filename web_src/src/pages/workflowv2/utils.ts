import {
  ComponentsNode,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowEventWithExecutions,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import { flattenObject } from "@/lib/utils";
import { formatTimeAgo } from "@/utils/date";
import { createElement, Fragment } from "react";
import { getComponentBaseMapper, getState, getTriggerRenderer } from "./mappers";
import { SidebarEvent } from "@/ui/componentSidebar/types";
import { LogEntry, LogRunItem } from "@/ui/CanvasLogSidebar";

export function mapTriggerEventsToSidebarEvents(
  events: WorkflowsWorkflowEvent[],
  node: ComponentsNode,
  limit?: number,
): SidebarEvent[] {
  const eventsToMap = limit ? events.slice(0, limit) : events;
  return eventsToMap.map((event) => mapTriggerEventToSidebarEvent(event, node));
}

export function mapTriggerEventToSidebarEvent(event: WorkflowsWorkflowEvent, node: ComponentsNode): SidebarEvent {
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

    const primaryComponentName = currentComponentNode?.component?.name
      ? currentComponentNode?.component?.name.split(".")[0]
      : "";
    const componentSubtitle = getComponentBaseMapper(primaryComponentName).subtitle?.(
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
      triggerEventId: execution.rootEvent?.id,
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
      triggerEventId: item.rootEvent?.id,
    };
  });
}

export function mapExecutionStateToLogType(state?: string): "success" | "error" {
  if (state === "failed" || state === "error" || state === "cancelled") {
    return "error";
  }
  return "success";
}

export function buildRunItemFromExecution(options: {
  execution: WorkflowsWorkflowNodeExecution;
  nodes: ComponentsNode[];
  onNodeSelect: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  event?: WorkflowsWorkflowEvent;
  timestampOverride?: string;
}): LogRunItem {
  const { execution, nodes, onNodeSelect, timestampOverride } = options;
  const { onExecutionSelect, event } = options;
  const componentNode = nodes.find((node) => node.id === execution.nodeId);
  const componentName = componentNode?.component?.name || "";
  const stateResolver = getState(componentName);
  const state = stateResolver(execution);
  const executionState = state || "unknown";
  const nodeId = componentNode?.id || execution.nodeId || "";
  const detail = execution.resultMessage;
  const triggerNode = event ? nodes.find((node) => node.id === event.nodeId) : undefined;
  const triggerEvent = event && triggerNode ? mapTriggerEventToSidebarEvent(event, triggerNode) : undefined;
  const executionId = execution.id;
  const title = createElement(
    Fragment,
    null,
    componentNode?.name || componentNode?.id || execution.nodeId || "Execution",
    nodeId
      ? createElement(
          Fragment,
          null,
          " · ",
          createElement(
            "button",
            {
              type: "button",
              className: "text-blue-600 underline hover:text-blue-700",
              onClick: () => {
                if (onExecutionSelect && event?.id && executionId) {
                  onExecutionSelect({
                    nodeId,
                    eventId: event.id,
                    executionId,
                    triggerEvent,
                  });
                  return;
                }

                onNodeSelect(nodeId);
              },
            },
            nodeId,
          ),
        )
      : null,
    " · ",
    executionState,
  );

  return {
    id: execution.id || `${execution.nodeId}-execution`,
    type: mapExecutionStateToLogType(state),
    title,
    timestamp: timestampOverride || execution.updatedAt || execution.createdAt || execution.rootEvent?.createdAt || "",
    isRunning: execution.state === "STATE_STARTED" || execution.state === "STATE_PENDING",
    detail,
    searchText: [
      componentNode?.name,
      componentNode?.id,
      execution.nodeId,
      executionState,
      execution.resultMessage,
      execution.resultReason,
      execution.result,
    ]
      .filter(Boolean)
      .join(" "),
  };
}

export function buildRunEntryFromEvent(options: {
  event: WorkflowsWorkflowEvent;
  nodes: ComponentsNode[];
  runItems?: LogRunItem[];
}): LogEntry {
  const { event, nodes, runItems = [] } = options;
  const triggerNode = nodes.find((node) => node.id === event.nodeId);
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const { title, subtitle } = triggerRenderer.getTitleAndSubtitle(event);
  const rootValues = triggerRenderer.getRootEventValues(event);

  return {
    id: event.id || `run-${Date.now()}`,
    source: "runs",
    timestamp: event.createdAt || "",
    title: `#${event.id?.slice(0, 4)} ·  ${title}` || "· Run",
    type: "run",
    runItems,
    searchText: [title, subtitle, event.id, event.nodeId, Object.values(rootValues).join(" ")]
      .filter(Boolean)
      .join(" "),
  };
}

export function mapWorkflowEventsToRunLogEntries(options: {
  events: WorkflowsWorkflowEventWithExecutions[];
  nodes: ComponentsNode[];
  onNodeSelect: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
}): LogEntry[] {
  const { events, nodes, onNodeSelect, onExecutionSelect } = options;

  return events.map((event) => {
    const runItems = (event.executions || []).map((execution) =>
      buildRunItemFromExecution({
        execution: execution as WorkflowsWorkflowNodeExecution,
        nodes,
        onNodeSelect,
        onExecutionSelect,
        event: event as WorkflowsWorkflowEvent,
        timestampOverride: event.createdAt || "",
      }),
    );

    return buildRunEntryFromEvent({
      event: event as WorkflowsWorkflowEvent,
      nodes,
      runItems,
    });
  });
}

export function mapCanvasNodesToLogEntries(options: {
  nodes: ComponentsNode[];
  workflowUpdatedAt: string;
  onNodeSelect: (nodeId: string) => void;
}): LogEntry[] {
  const { nodes, workflowUpdatedAt, onNodeSelect } = options;

  return (
    nodes
      .filter((node: ComponentsNode) => node.errorMessage)
      .map((node, index) => {
        const title = createElement(
          Fragment,
          null,
          "Component not configured - ",
          createElement(
            "button",
            {
              type: "button",
              className: "text-blue-600 underline hover:text-blue-700",
              onClick: () => onNodeSelect(node.id || ""),
            },
            node.id,
          ),
          " - ",
          node.errorMessage,
        );

        return {
          id: `log-${index + 1}`,
          source: "canvas",
          timestamp: workflowUpdatedAt,
          title,
          type: "warning",
          searchText: `component not configured ${node.id} ${node.errorMessage}`,
        } as LogEntry;
      }) || []
  );
}

export function buildCanvasStatusLogEntry(options: {
  id: string;
  message: string;
  type: "success" | "error" | "warning";
  timestamp: string;
}): LogEntry {
  const { id, message, type, timestamp } = options;

  return {
    id,
    source: "canvas",
    timestamp,
    title: message,
    type,
    searchText: message,
  };
}

export function buildTabData(
  nodeId: string,
  event: SidebarEvent,
  options: {
    workflowNodes: ComponentsNode[];
    nodeEventsMap: Record<string, WorkflowsWorkflowEvent[]>;
    nodeExecutionsMap: Record<string, WorkflowsWorkflowNodeExecution[]>;
    nodeQueueItemsMap: Record<string, WorkflowsWorkflowNodeQueueItem[]>;
  },
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
