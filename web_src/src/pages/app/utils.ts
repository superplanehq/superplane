import type {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasRunRef,
  ActionsAction,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
  SuperplaneMeUser,
} from "@/api-client";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimeAgo } from "@/lib/date";
import { flattenObject } from "@/lib/utils";
import type { LogEntry } from "@/ui/CanvasLogSidebar";
import type { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { createElement, Fragment } from "react";
import { getComponentBaseMapper, getExecutionDetails, getState, getTriggerRenderer } from "./mappers";
import type { ComponentDefinition, EventInfo, ExecutionInfo, NodeInfo, QueueItemInfo, User } from "./mappers/types";

const logEntryLinkClassName =
  "text-blue-600 underline hover:text-blue-700 dark:text-indigo-300 dark:hover:text-indigo-200";

export function generateNodeId(blockName: string, nodeName: string): string {
  const randomChars = Math.random().toString(36).substring(2, 8);
  const sanitizedBlock = blockName.toLowerCase().replace(/[^a-z0-9]/g, "-");
  const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, "-");
  return `${sanitizedBlock}-${sanitizedName}-${randomChars}`;
}

export function getComparableIntegrationId(node: Record<string, unknown>): string | null {
  const integration = node.integration;
  if (integration && typeof integration === "object" && "id" in integration) {
    const integrationId = integration.id;
    return typeof integrationId === "string" && integrationId ? integrationId : null;
  }

  const integrationId = node.integrationId;
  return typeof integrationId === "string" && integrationId ? integrationId : null;
}

function normalizeNodeForSaveSignature(node: ComponentsNode): ComponentsNode {
  if (!node.integration) {
    return node;
  }

  return {
    ...node,
    integration: node.integration.id ? { id: node.integration.id } : undefined,
  };
}

export function getWorkflowSaveSignature(workflow: CanvasesCanvas | null | undefined): string {
  if (!workflow) {
    return "";
  }

  return JSON.stringify({
    name: workflow.metadata?.name ?? "",
    description: workflow.metadata?.description ?? "",
    nodes: (workflow.spec?.nodes ?? []).map(normalizeNodeForSaveSignature),
    edges: workflow.spec?.edges ?? [],
  });
}

/**
 * Generates a unique node name based on component name + ordinal number.
 * First instance: "if", second: "if2", third: "if3", etc.
 *
 * @param componentName - The component name (e.g., "semaphore.onPipelineDone")
 * @param existingNodeNames - Array of existing node names on the canvas
 * @returns A unique node name (e.g., "semaphore.onPipelineDone" or "semaphore.onPipelineDone2")
 */
export function generateUniqueNodeName(componentName: string, existingNodeNames: string[]): string {
  const nameMatch = componentName.match(/^(.*?)(?:\s+(\d+))?$/);
  const baseName = nameMatch?.[1] || componentName;

  // Escape special regex characters in the base name
  const escapedBaseName = baseName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

  // Check if the base name already exists
  const baseNameExists = existingNodeNames.includes(baseName);

  // Find all existing nodes with this base name pattern (base + space + number)
  const pattern = new RegExp(`^${escapedBaseName}\\s+(\\d+)$`);
  const existingOrdinals: number[] = [];

  for (const name of existingNodeNames) {
    const match = name.match(pattern);
    if (match) {
      existingOrdinals.push(parseInt(match[1], 10));
    }
  }

  // If no existing nodes with this name, return the original name
  if (!baseNameExists && existingOrdinals.length === 0) {
    return componentName;
  }

  // Find the next available ordinal (starting from 2)
  const nextOrdinal = existingOrdinals.length > 0 ? Math.max(...existingOrdinals) + 1 : 2;

  return `${baseName} ${nextOrdinal}`;
}

export function mapTriggerEventsToSidebarEvents(
  events: CanvasesCanvasEvent[],
  node: ComponentsNode,
  limit?: number,
): SidebarEvent[] {
  const eventsToMap = limit ? events.slice(0, limit) : events;
  return eventsToMap.map((event) => mapTriggerEventToSidebarEvent(event, node));
}

export function mapTriggerEventToSidebarEvent(event: CanvasesCanvasEvent, node: ComponentsNode): SidebarEvent {
  const triggerRenderer = getTriggerRenderer(getNodeComponentName(node));
  const eventInfo = buildEventInfo(event);
  const { title, subtitle } = triggerRenderer.getTitleAndSubtitle({ event: eventInfo });
  const values = triggerRenderer.getRootEventValues({ event: eventInfo });
  const state = triggerRenderer.getEventState?.({ event: eventInfo }) || "triggered";

  return {
    id: event.id!,
    title,
    subtitle: subtitle || renderTimeAgo(new Date(event.createdAt!)),
    state,
    isOpen: false,
    receivedAt: event.createdAt ? new Date(event.createdAt) : undefined,
    values,
    triggerEventId: event.id!,
    kind: "trigger",
    nodeId: node.id,
    originalEvent: event,
    runId: event.runId,
  };
}

export function mapExecutionsToSidebarEvents(
  executions: CanvasesCanvasNodeExecution[],
  nodes: ComponentsNode[],
  limit?: number,
): SidebarEvent[] {
  const executionsToMap = limit ? executions.slice(0, limit) : executions;

  return executionsToMap.map((execution) => {
    const currentComponentNode = nodes.find((n) => n.id === execution.nodeId);
    const componentName = getNodeComponentName(currentComponentNode);
    const stateResolver = getState(componentName);
    const state = stateResolver(buildExecutionInfo(execution));
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(getNodeComponentName(rootTriggerNode));

    const componentMapper = getComponentBaseMapper(componentName);
    const componentSubtitle = componentMapper.subtitle?.({
      node: buildNodeInfo(currentComponentNode as ComponentsNode),
      execution: buildExecutionInfo(execution),
    });

    const { title, subtitle } = execution.rootEvent
      ? rootTriggerRenderer.getTitleAndSubtitle({ event: buildEventInfo(execution.rootEvent!) })
      : {
          title: execution.id || "Execution",
          subtitle: execution.createdAt ? formatTimeAgo(new Date(execution.createdAt)).replace(" ago", "") : "",
        };

    const values = execution.rootEvent
      ? rootTriggerRenderer.getRootEventValues({ event: buildEventInfo(execution.rootEvent!) })
      : {};

    return {
      id: execution.id!,
      title,
      subtitle: componentSubtitle || subtitle || renderTimeAgo(new Date(execution.createdAt!)),
      state,
      isOpen: false,
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      values,
      executionId: execution.id!,
      kind: "execution",
      nodeId: execution?.nodeId,
      originalExecution: execution,
      triggerEventId: execution.rootEvent?.id,
      runId: execution.runId || execution.rootEvent?.runId,
    };
  });
}

export function mapQueueItemsToSidebarEvents(
  queueItems: CanvasesCanvasNodeQueueItem[],
  nodes: ComponentsNode[],
  limit?: number,
): SidebarEvent[] {
  const queueItemsToMap = limit ? queueItems.slice(0, limit) : queueItems;
  return queueItemsToMap.map((item) => {
    const rootTriggerNode = nodes.find((n) => n.id === item.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(getNodeComponentName(rootTriggerNode));

    const { title, subtitle } = item.rootEvent
      ? rootTriggerRenderer.getTitleAndSubtitle({
          event: buildEventInfo(item.rootEvent!),
        })
      : {
          title: item.id || "Execution",
          subtitle: item.createdAt ? formatTimeAgo(new Date(item.createdAt)).replace(" ago", "") : "",
        };

    const values = item.rootEvent
      ? rootTriggerRenderer.getRootEventValues({ event: buildEventInfo(item.rootEvent!) })
      : {};

    return {
      id: item.id!,
      title,
      subtitle: subtitle || renderTimeAgo(new Date(item.createdAt!)),
      state: "queued" as const,
      isOpen: false,
      receivedAt: item.createdAt ? new Date(item.createdAt) : undefined,
      kind: "queue",
      values,
      triggerEventId: item.rootEvent?.id,
    };
  });
}

export function getSidebarEventRootEventId(event: SidebarEvent): string | undefined {
  return (
    event.triggerEventId || event.originalExecution?.rootEvent?.id || (event.kind === "trigger" ? event.id : undefined)
  );
}

export function getSidebarEventExecutionId(event: SidebarEvent): string | undefined {
  if (event.executionId) {
    return event.executionId;
  }

  if (event.kind === "execution") {
    return event.id;
  }

  return undefined;
}

export function findRunIdForSidebarEvent(runs: CanvasesCanvasRun[], event: SidebarEvent): string | null {
  if (event.runId) {
    return event.runId;
  }

  const executionId = getSidebarEventExecutionId(event);
  if (executionId) {
    const run = runs.find((candidate) => candidate.executions?.some((execution) => execution.id === executionId));
    if (run?.id) {
      return run.id;
    }
  }

  const rootEventId = getSidebarEventRootEventId(event);
  if (!rootEventId) {
    return null;
  }

  return runs.find((run) => run.rootEvent?.id === rootEventId)?.id ?? null;
}

export function mapCanvasNodesToLogEntries(options: {
  nodes: ComponentsNode[];
  workflowUpdatedAt: string;
  onNodeSelect: (nodeId: string) => void;
}): LogEntry[] {
  const { nodes, workflowUpdatedAt, onNodeSelect } = options;

  const entries: LogEntry[] = [];

  // Add error entries for nodes with configuration errors
  nodes
    .filter((node: ComponentsNode) => node.errorMessage)
    .forEach((node, index) => {
      const title = createElement(
        Fragment,
        null,
        "Component not configured - ",
        createElement(
          "button",
          {
            type: "button",
            className: logEntryLinkClassName,
            onClick: () => onNodeSelect(node.id || ""),
          },
          node.id,
        ),
        " - ",
        node.errorMessage,
      );

      entries.push({
        id: `error-${index + 1}`,
        source: "canvas",
        timestamp: workflowUpdatedAt,
        title,
        type: "warning",
        searchText: `component not configured ${node.id} ${node.errorMessage}`,
      } as LogEntry);
    });

  // Add warning entries for nodes with warnings (like shadowed names)
  nodes
    .filter((node: ComponentsNode) => node.warningMessage)
    .forEach((node, index) => {
      const title = createElement(
        Fragment,
        null,
        createElement(
          "button",
          {
            type: "button",
            className: logEntryLinkClassName,
            onClick: () => onNodeSelect(node.id || ""),
          },
          node.name || node.id,
        ),
        " - ",
        node.warningMessage,
      );

      entries.push({
        id: `warning-${index + 1}`,
        source: "canvas",
        timestamp: workflowUpdatedAt,
        title,
        type: "warning",
        searchText: `${node.name} ${node.id} ${node.warningMessage}`,
      } as LogEntry);
    });

  return entries;
}

export function buildTabData(
  nodeId: string,
  event: SidebarEvent,
  options: {
    workflowNodes: ComponentsNode[];
    nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
    nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
    nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
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
    const triggerRenderer = getTriggerRenderer(getNodeComponentName(node));
    const eventValues = triggerRenderer.getRootEventValues({ event: buildEventInfo(triggerEvent) });

    tabData.current = {
      ...eventValues,
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
    const queueItem = queueItems.find((item: CanvasesCanvasNodeQueueItem) => item.id === event.id);

    if (!queueItem) return undefined;

    const tabData: TabData = {};

    if (queueItem.rootEvent) {
      const rootTriggerNode = workflowNodes.find((n) => n.id === queueItem.rootEvent?.nodeId);
      const rootTriggerRenderer = getTriggerRenderer(getNodeComponentName(rootTriggerNode));
      const rootEventValues = rootTriggerRenderer.getRootEventValues({ event: buildEventInfo(queueItem.rootEvent!) });

      tabData.root = Object.assign({}, rootEventValues, {
        "Created At": queueItem.rootEvent.createdAt
          ? new Date(queueItem.rootEvent.createdAt).toLocaleString()
          : undefined,
      });
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
  const execution = executions.find((exec: CanvasesCanvasNodeExecution) => exec.id === event.id);

  if (!execution) return undefined;

  // Extract tab data from execution
  const tabData: TabData = {};

  let currentData: Record<string, unknown> = {};
  const componentName = typeof node.component === "string" ? node.component : undefined;
  if (componentName) {
    const customDetails = getExecutionDetails(componentName, execution, node, workflowNodes);
    if (customDetails && Object.keys(customDetails).length > 0) {
      currentData = { ...customDetails };
    }
  }

  if (Object.keys(currentData).length === 0) {
    const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
    const dataSource = hasOutputs ? execution.outputs : execution.metadata || {};
    currentData = { ...flattenObject(dataSource) };
  }

  // Filter out undefined and empty values
  tabData.current = Object.fromEntries(
    Object.entries(currentData).filter(([_, value]) => value !== undefined && value !== "" && value !== null),
  );

  // Root tab: root event data
  if (execution.rootEvent) {
    const rootTriggerNode = workflowNodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(getNodeComponentName(rootTriggerNode));
    const rootEventValues = rootTriggerRenderer.getRootEventValues({ event: buildEventInfo(execution.rootEvent!) });

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

export function buildExecutionInfo(
  execution: CanvasesCanvasNodeExecution,
  options?: { runs?: CanvasesCanvasRunRef[] },
): ExecutionInfo {
  return {
    id: execution.id!,
    createdAt: execution.createdAt!,
    updatedAt: execution.updatedAt!,
    state: execution.state!,
    result: execution.result!,
    resultReason: execution.resultReason!,
    resultMessage: execution.resultMessage!,
    metadata: execution.metadata!,
    configuration: execution.configuration!,
    outputs: execution.outputs!,
    rootEvent: buildEventInfo(execution.rootEvent!),
    runs: options?.runs,
  };
}

export function buildComponentDefinition(component?: Partial<ActionsAction>): ComponentDefinition {
  return {
    name: component?.name || "unknown",
    label: component?.label || "Unknown",
    description: component?.description || "",
    icon: component?.icon || "bolt",
    color: component?.color || "gray",
  };
}

export function buildEventInfo(event: CanvasesCanvasEvent): EventInfo | undefined {
  if (!event) return undefined;

  return {
    id: event.id!,
    createdAt: event.createdAt!,
    customName: event.customName,
    data: event.data?.data || {},
    nodeId: event.nodeId!,
    type: (event.data?.type as string) || "",
  };
}

export function buildQueueItemInfo(queueItem: CanvasesCanvasNodeQueueItem): QueueItemInfo {
  return {
    id: queueItem.id!,
    createdAt: queueItem.createdAt!,
    rootEvent: buildEventInfo(queueItem.rootEvent!),
  };
}

export function buildNodeInfo(node: ComponentsNode): NodeInfo {
  return {
    id: node.id!,
    name: node.name || "",
    componentName: getNodeComponentName(node),
    isCollapsed: node.isCollapsed || false,
    configuration: node.configuration,
    metadata: node.metadata,
  };
}

export function getNodeComponentName(node?: ({ component?: string; componentName?: string } & object) | null): string {
  return node?.component || node?.componentName || "";
}

export function buildUserInfo(user?: SuperplaneMeUser | null): User | undefined {
  if (!user) return undefined;

  return {
    id: user.id!,
    name: user.name || "",
    email: user.email || "",
    roles: user.roles || [],
    groups: user.groups || [],
  };
}
