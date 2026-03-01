import {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
} from "@/api-client";
import { flattenObject } from "@/lib/utils";
import { LogEntry, LogRunItem } from "@/ui/CanvasLogSidebar";
import { TabData } from "@/ui/componentSidebar/SidebarEventItem/SidebarEventItem";
import { SidebarEvent } from "@/ui/componentSidebar/types";
import { formatTimeAgo } from "@/utils/date";
import { createElement, Fragment, type ReactNode } from "react";
import { getComponentBaseMapper, getState, getTriggerRenderer } from "./mappers";
import { ComponentDefinition, EventInfo, ExecutionInfo, NodeInfo, QueueItemInfo } from "./mappers/types";

export function generateNodeId(blockName: string, nodeName: string): string {
  const randomChars = Math.random().toString(36).substring(2, 8);
  const sanitizedBlock = blockName.toLowerCase().replace(/[^a-z0-9]/g, "-");
  const sanitizedName = nodeName.toLowerCase().replace(/[^a-z0-9]/g, "-");
  return `${sanitizedBlock}-${sanitizedName}-${randomChars}`;
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
  const triggerRenderer = getTriggerRenderer(node.trigger?.name || "");
  const { title, subtitle } = triggerRenderer.getTitleAndSubtitle({ event: buildEventInfo(event) });
  const values = triggerRenderer.getRootEventValues({ event: buildEventInfo(event) });

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
  executions: CanvasesCanvasNodeExecution[],
  nodes: ComponentsNode[],
  limit?: number,
  additionalData?: unknown,
): SidebarEvent[] {
  const executionsToMap = limit ? executions.slice(0, limit) : executions;

  return executionsToMap.map((execution) => {
    const currentComponentNode = nodes.find((n) => n.id === execution.nodeId);
    const stateResolver = getState(currentComponentNode?.component?.name || "");
    const state = stateResolver(buildExecutionInfo(execution));
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const componentName = currentComponentNode?.component?.name || "";
    const componentMapper = getComponentBaseMapper(componentName);
    const componentSubtitle = componentMapper.subtitle?.({
      node: buildNodeInfo(currentComponentNode as ComponentsNode),
      execution: buildExecutionInfo(execution),
      additionalData,
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
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]> | undefined,
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
    ? rootTriggerRenderer.getTitleAndSubtitle({
        event: buildEventInfo(queueItem.rootEvent!),
      })
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
  queueItems: CanvasesCanvasNodeQueueItem[],
  nodes: ComponentsNode[],
  limit?: number,
): SidebarEvent[] {
  const queueItemsToMap = limit ? queueItems.slice(0, limit) : queueItems;
  return queueItemsToMap.map((item) => {
    const rootTriggerNode = nodes.find((n) => n.id === item.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

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

export function mapExecutionStateToLogType(
  execution: CanvasesCanvasNodeExecution,
  state?: string,
): "success" | "error" | "resolved-error" {
  if (execution.resultReason === "RESULT_REASON_ERROR_RESOLVED") {
    return "resolved-error";
  }
  return state === "error" ? "error" : "success";
}

export function buildRunItemFromExecution(options: {
  execution: CanvasesCanvasNodeExecution;
  nodes: ComponentsNode[];
  onNodeSelect: (nodeId: string) => void;
  onExecutionSelect?: (options: {
    nodeId: string;
    eventId: string;
    executionId: string;
    triggerEvent?: SidebarEvent;
  }) => void;
  event?: CanvasesCanvasEvent;
  timestampOverride?: string;
}): LogRunItem {
  const { execution, nodes, onNodeSelect, timestampOverride } = options;
  const { onExecutionSelect, event } = options;
  const componentNode = nodes.find((node) => node.id === execution.nodeId);
  const componentName = componentNode?.component?.name || "";
  const stateResolver = getState(componentName);
  const state = stateResolver(buildExecutionInfo(execution));
  const executionState = execution.resultReason === "RESULT_REASON_ERROR_RESOLVED" ? "error" : state || "unknown";
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
    type: mapExecutionStateToLogType(execution, state),
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
  event: CanvasesCanvasEvent;
  nodes: ComponentsNode[];
  runItems?: LogRunItem[];
}): LogEntry {
  const { event, nodes, runItems = [] } = options;
  const triggerNode = nodes.find((node) => node.id === event.nodeId);
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const { title, subtitle } = triggerRenderer.getTitleAndSubtitle({ event: buildEventInfo(event) });
  const rootValues = triggerRenderer.getRootEventValues({ event: buildEventInfo(event) });

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
  events: CanvasesCanvasEventWithExecutions[];
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
        execution: execution as CanvasesCanvasNodeExecution,
        nodes,
        onNodeSelect,
        onExecutionSelect,
        event: event as CanvasesCanvasEvent,
        timestampOverride: event.createdAt || "",
      }),
    );

    return buildRunEntryFromEvent({
      event: event as CanvasesCanvasEvent,
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
            className: "text-blue-600 underline hover:text-blue-700",
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
            className: "text-blue-600 underline hover:text-blue-700",
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

export function buildCanvasStatusLogEntry(options: {
  id: string;
  message: string;
  type: "success" | "error" | "warning";
  timestamp: string;
  detail?: ReactNode;
  searchText?: string;
}): LogEntry {
  const { id, message, type, timestamp, detail, searchText } = options;
  const resolvedSearchText = searchText || message;

  return {
    id,
    source: "canvas",
    timestamp,
    title: message,
    type,
    searchText: resolvedSearchText,
    detail,
  };
}

function normalizeNodeConfiguration(node: ComponentsNode): Record<string, unknown> {
  return node.configuration ? JSON.parse(JSON.stringify(node.configuration)) : {};
}

function getNodeLabel(node: ComponentsNode): string {
  if (node.name && node.id) {
    return `${node.name} (${node.id})`;
  }

  return node.name || node.id || "unnamed node";
}

function getEdgeKey(edge: ComponentsEdge, index: number): string {
  const source = edge.sourceId || "unknown";
  const target = edge.targetId || "unknown";
  const channel = edge.channel || "";

  return `${source}:${channel}->${target}:${index}`;
}

function buildConnectionListItems(
  edges: ComponentsEdge[],
  nodesById: Map<string, ComponentsNode>,
  onNodeSelect: (nodeId: string) => void,
  options?: {
    maxItems?: number;
    linkIds?: boolean;
    existingNodesById?: Map<string, ComponentsNode>;
    listContext?: boolean;
  },
): { items: ReactNode[]; text: string } {
  const maxItems = options?.maxItems ?? 2;
  const linkIds = options?.linkIds ?? true;
  const existingNodesById = options?.existingNodesById;
  const listContext = options?.listContext ?? false;
  const edgePairCounts = new Map<string, number>();
  edges.forEach((edge) => {
    const sourceId = edge.sourceId || "";
    const targetId = edge.targetId || "";
    if (!sourceId && !targetId) {
      return;
    }
    const key = `${sourceId}::${targetId}`;
    edgePairCounts.set(key, (edgePairCounts.get(key) ?? 0) + 1);
  });
  const shouldShowChannel = (edge: ComponentsEdge) => {
    if (!edge.channel) {
      return false;
    }
    const key = `${edge.sourceId || ""}::${edge.targetId || ""}`;
    return (edgePairCounts.get(key) ?? 0) > 1;
  };
  const displayEdges = edges.slice(0, maxItems);
  const labels = displayEdges
    .map((edge) => {
      const sourceLabel = edge.sourceId
        ? nodesById.get(edge.sourceId)
          ? getNodeLabel(nodesById.get(edge.sourceId)!)
          : edge.sourceId
        : "";
      const targetLabel = edge.targetId
        ? nodesById.get(edge.targetId)
          ? getNodeLabel(nodesById.get(edge.targetId)!)
          : edge.targetId
        : "";
      const channelSuffix = shouldShowChannel(edge) ? ` [${edge.channel}]` : "";

      if (sourceLabel && targetLabel) {
        return `${sourceLabel}${channelSuffix} -> ${targetLabel}`;
      }
      if (sourceLabel) {
        return `${sourceLabel}${channelSuffix}`;
      }
      if (targetLabel) {
        return `${targetLabel}`;
      }
      return "";
    })
    .filter(Boolean);
  const remainingCount = edges.length - displayEdges.length;

  const edgeItems = edges
    .map((edge, index) => {
      const sourceId = edge.sourceId || "";
      const targetId = edge.targetId || "";
      const sourceNode = sourceId ? nodesById.get(sourceId) : undefined;
      const targetNode = targetId ? nodesById.get(targetId) : undefined;
      const sourceName = sourceNode?.name || "";
      const targetName = targetNode?.name || "";

      const renderNodePart = (nodeId: string, nodeName: string, allowLink: boolean) => {
        if (!nodeId) {
          return null;
        }

        if (!allowLink) {
          return nodeName ? `${nodeName} (${nodeId})` : nodeId;
        }

        const idButton = createElement(
          "button",
          {
            type: "button",
            className: "text-blue-600 underline hover:text-blue-700",
            onClick: () => onNodeSelect(nodeId),
          },
          nodeId,
        );

        if (nodeName) {
          return createElement(Fragment, null, nodeName, " (", idButton, ")");
        }

        return idButton;
      };

      const allowSourceLink = linkIds && !!sourceId && (!existingNodesById || existingNodesById.has(sourceId));
      const allowTargetLink = linkIds && !!targetId && (!existingNodesById || existingNodesById.has(targetId));
      const sourcePart = renderNodePart(sourceId, sourceName, allowSourceLink);
      const targetPart = renderNodePart(targetId, targetName, allowTargetLink);
      const channelSuffix = shouldShowChannel(edge) ? ` [${edge.channel}]` : "";
      if (!sourcePart && !targetPart) {
        return null;
      }

      if (!sourcePart) {
        return {
          key: `${sourceId}-${targetId}-${index}`,
          content: createElement(Fragment, null, targetPart),
        };
      }

      if (!targetPart) {
        return {
          key: `${sourceId}-${targetId}-${index}`,
          content: createElement(Fragment, null, sourcePart, channelSuffix),
        };
      }

      return {
        key: `${sourceId}-${targetId}-${index}`,
        content: createElement(Fragment, null, sourcePart, channelSuffix, " -> ", targetPart),
      };
    })
    .filter(Boolean) as Array<{ key: string; content: ReactNode }>;

  const buildListItem = (item: { key: string; content: ReactNode }) => {
    if (listContext) {
      return createElement("li", { key: item.key }, item.content);
    }
    return createElement(Fragment, { key: item.key }, item.content);
  };

  const items = edgeItems.slice(0, maxItems).map(buildListItem);
  const hiddenItems = edgeItems.slice(maxItems);
  if (remainingCount > 0 && hiddenItems.length > 0) {
    items.push(
      listContext
        ? createElement(
            "details",
            { key: "more", className: "group contents" },
            createElement(
              "summary",
              { className: "list-item cursor-pointer text-slate-900 underline group-open:hidden" },
              `+${remainingCount} more`,
            ),
            ...hiddenItems.map((item, index) =>
              createElement(
                "li",
                { key: `more-edge-${index}`, className: "hidden group-open:list-item" },
                item.content,
              ),
            ),
          )
        : createElement(
            "details",
            { key: "more", className: "group inline-block" },
            createElement(
              "summary",
              { className: "cursor-pointer text-slate-900 underline group-open:hidden" },
              `+${remainingCount} more`,
            ),
            createElement(
              "ul",
              { className: "mt-1 list-disc pl-5 space-y-1 text-slate-600" },
              ...hiddenItems.map((item, index) => createElement("li", { key: `more-edge-${index}` }, item.content)),
            ),
          ),
    );
    labels.push(`+${remainingCount} more`);
  }

  return {
    items,
    text: labels.join(", "),
  };
}

function buildNodeListItems(
  nodes: ComponentsNode[],
  onNodeSelect: (nodeId: string) => void,
  options?: { maxItems?: number; linkIds?: boolean; listContext?: boolean },
): { items: ReactNode[]; text: string } {
  const maxItems = options?.maxItems ?? 2;
  const linkIds = options?.linkIds ?? true;
  const listContext = options?.listContext ?? false;
  const displayNodes = nodes.slice(0, maxItems);
  const labels = displayNodes.map((node) => getNodeLabel(node));
  const remainingCount = nodes.length - displayNodes.length;

  const nodeItems = nodes.map((node, index) => {
    const nodeId = node.id || "";
    const label = getNodeLabel(node);
    const name = node.name || "";
    if (!nodeId || !linkIds) {
      return { key: `${nodeId || label}-${index}`, content: createElement("span", null, label) };
    }

    const idButton = createElement(
      "button",
      {
        type: "button",
        className: "text-blue-600 underline hover:text-blue-700",
        onClick: () => onNodeSelect(nodeId),
      },
      nodeId,
    );

    if (name) {
      return { key: `${nodeId}-${index}`, content: createElement("span", null, name, " (", idButton, ")") };
    }

    return { key: `${nodeId}-${index}`, content: createElement("span", null, idButton) };
  });

  const buildListItem = (item: { key: string; content: ReactNode }) => {
    if (listContext) {
      return createElement("li", { key: item.key }, item.content);
    }
    return createElement(Fragment, { key: item.key }, item.content);
  };

  const items = nodeItems.slice(0, maxItems).map(buildListItem);
  const hiddenItems = nodeItems.slice(maxItems);
  if (remainingCount > 0 && hiddenItems.length > 0) {
    items.push(
      listContext
        ? createElement(
            "details",
            { key: "more", className: "group contents" },
            createElement(
              "summary",
              { className: "list-item cursor-pointer text-slate-900 underline group-open:hidden" },
              `+${remainingCount} more`,
            ),
            ...hiddenItems.map((item, index) =>
              createElement(
                "li",
                { key: `more-node-${index}`, className: "hidden group-open:list-item" },
                item.content,
              ),
            ),
          )
        : createElement(
            "details",
            { key: "more", className: "group inline-block" },
            createElement(
              "summary",
              { className: "cursor-pointer text-slate-900 underline group-open:hidden" },
              `+${remainingCount} more`,
            ),
            createElement(
              "ul",
              { className: "mt-1 list-disc pl-5 space-y-1 text-slate-600" },
              ...hiddenItems.map((item, index) => createElement("li", { key: `more-node-${index}` }, item.content)),
            ),
          ),
    );
    labels.push(`+${remainingCount} more`);
  }

  return {
    items,
    text: labels.join(", "),
  };
}

function renderConnectionSublist(
  edges: ComponentsEdge[],
  nodesById: Map<string, ComponentsNode>,
  onNodeSelect: (nodeId: string) => void,
  options?: {
    maxItems?: number;
    linkIds?: boolean;
    existingNodesById?: Map<string, ComponentsNode>;
  },
): { content: ReactNode; text: string } {
  const { items, text } = buildConnectionListItems(edges, nodesById, onNodeSelect, { ...options, listContext: true });

  return {
    content: createElement("ul", { className: "mt-1 list-disc pl-5 space-y-1" }, ...items),
    text,
  };
}

function renderNodeSublist(
  nodes: ComponentsNode[],
  onNodeSelect: (nodeId: string) => void,
  options?: { maxItems?: number; linkIds?: boolean },
): { content: ReactNode; text: string } {
  const { items, text } = buildNodeListItems(nodes, onNodeSelect, { ...options, listContext: true });

  return {
    content: createElement("ul", { className: "mt-1 list-disc pl-5 space-y-1" }, ...items),
    text,
  };
}

function formatSummaryEntry(
  action: string,
  noun: string,
  nodes: ComponentsNode[],
  onNodeSelect: (nodeId: string) => void,
  options?: { linkIds?: boolean },
): { title: string; body: ReactNode; text: string } | undefined {
  if (nodes.length === 0) {
    return undefined;
  }

  const count = nodes.length;
  const label = count === 1 ? noun : `${noun}s`;
  const title = `${action} ${count} ${label}`;
  const list = renderNodeSublist(nodes, onNodeSelect, { linkIds: options?.linkIds });

  return {
    title,
    body: list.content,
    text: `${title}: ${list.text}`,
  };
}

function formatRemovedNodesEntry(options: {
  nodes: ComponentsNode[];
  onNodeSelect: (nodeId: string) => void;
}): { title: string; body: ReactNode; text: string } | undefined {
  const { nodes, onNodeSelect } = options;
  if (nodes.length === 0) {
    return undefined;
  }

  const count = nodes.length;
  const label = count === 1 ? "component" : "components";
  const title = `Removed ${count} ${label}`;
  const list = renderNodeSublist(nodes, onNodeSelect, { linkIds: false });

  return {
    title,
    body: list.content,
    text: `${title}: ${list.text}`,
  };
}

function formatSummaryConnectionEntry(
  action: string,
  noun: string,
  edges: ComponentsEdge[],
  nodesById: Map<string, ComponentsNode>,
  onNodeSelect: (nodeId: string) => void,
  options?: { linkIds?: boolean; existingNodesById?: Map<string, ComponentsNode> },
): { title: string; body: ReactNode; text: string } | undefined {
  if (edges.length === 0) {
    return undefined;
  }

  const count = edges.length;
  const label = count === 1 ? noun : `${noun}s`;
  const title = `${action} ${count} ${label}`;
  const list = renderConnectionSublist(edges, nodesById, onNodeSelect, {
    linkIds: options?.linkIds,
    existingNodesById: options?.existingNodesById,
  });

  return {
    title,
    body: list.content,
    text: `${title}: ${list.text}`,
  };
}

export function summarizeWorkflowChanges(options: {
  before: CanvasesCanvas | null;
  after: CanvasesCanvas | null;
  onNodeSelect: (nodeId: string) => void;
}): { detail?: ReactNode; searchText?: string; changeCount?: number } {
  const { before, after, onNodeSelect } = options;

  if (!before || !after) {
    return {};
  }

  const beforeNodes = new Map<string, ComponentsNode>();
  const afterNodes = new Map<string, ComponentsNode>();
  const beforeEdges = new Map<string, ComponentsEdge>();
  const afterEdges = new Map<string, ComponentsEdge>();

  (before.spec?.nodes || []).forEach((node, index) => {
    if (!node.id) {
      beforeNodes.set(`node-${index}`, node);
      return;
    }
    beforeNodes.set(node.id, node);
  });

  (after.spec?.nodes || []).forEach((node, index) => {
    if (!node.id) {
      afterNodes.set(`node-${index}`, node);
      return;
    }
    afterNodes.set(node.id, node);
  });

  (before.spec?.edges || []).forEach((edge, index) => {
    beforeEdges.set(getEdgeKey(edge, index), edge);
  });

  (after.spec?.edges || []).forEach((edge, index) => {
    afterEdges.set(getEdgeKey(edge, index), edge);
  });

  const addedNodes: ComponentsNode[] = [];
  const removedNodes: ComponentsNode[] = [];
  const movedNodes: ComponentsNode[] = [];
  const updatedNodes: ComponentsNode[] = [];
  const addedConnections: ComponentsEdge[] = [];
  const removedConnections: ComponentsEdge[] = [];
  const updatedConnections: ComponentsEdge[] = [];

  afterNodes.forEach((node, id) => {
    const beforeNode = beforeNodes.get(id);
    if (!beforeNode) {
      addedNodes.push(node);
      return;
    }

    const beforePosition = beforeNode.position;
    const afterPosition = node.position;
    if (
      beforePosition &&
      afterPosition &&
      (beforePosition.x !== afterPosition.x || beforePosition.y !== afterPosition.y)
    ) {
      movedNodes.push(node);
    }

    const beforeConfiguration = JSON.stringify(normalizeNodeConfiguration(beforeNode));
    const afterConfiguration = JSON.stringify(normalizeNodeConfiguration(node));
    if (beforeConfiguration !== afterConfiguration) {
      updatedNodes.push(node);
    }
  });

  beforeNodes.forEach((node, id) => {
    if (!afterNodes.has(id)) {
      removedNodes.push(node);
    }
  });

  afterEdges.forEach((edge, id) => {
    const beforeEdge = beforeEdges.get(id);
    if (!beforeEdge) {
      addedConnections.push(edge);
      return;
    }

    if (JSON.stringify(beforeEdge) !== JSON.stringify(edge)) {
      updatedConnections.push(edge);
    }
  });

  beforeEdges.forEach((edge, id) => {
    if (!afterEdges.has(id)) {
      removedConnections.push(edge);
    }
  });

  const removedNodeIds = new Set(removedNodes.map((node) => node.id).filter(Boolean) as string[]);
  const removedConnectionsStandalone = removedConnections.filter((edge) => {
    const sourceId = edge.sourceId || "";
    const targetId = edge.targetId || "";
    const isConnectedToRemovedNode = removedNodeIds.has(sourceId) || removedNodeIds.has(targetId);

    return !isConnectedToRemovedNode;
  });

  const changeCount =
    addedNodes.length +
    removedNodes.length +
    updatedNodes.length +
    movedNodes.length +
    addedConnections.length +
    removedConnectionsStandalone.length +
    updatedConnections.length;

  const summaryParts = [
    formatSummaryEntry("Added", "component", addedNodes, onNodeSelect, { linkIds: true }),
    formatRemovedNodesEntry({ nodes: removedNodes, onNodeSelect }),
    formatSummaryEntry("Updated", "component", updatedNodes, onNodeSelect, { linkIds: true }),
    formatSummaryEntry("Moved", "component", movedNodes, onNodeSelect, { linkIds: true }),
    formatSummaryConnectionEntry("Added", "connection", addedConnections, afterNodes, onNodeSelect, { linkIds: true }),
    removedNodeIds.size === 0
      ? formatSummaryConnectionEntry("Removed", "connection", removedConnectionsStandalone, beforeNodes, onNodeSelect, {
          linkIds: true,
          existingNodesById: afterNodes,
        })
      : undefined,
    formatSummaryConnectionEntry("Updated", "connection", updatedConnections, afterNodes, onNodeSelect, {
      linkIds: true,
    }),
  ].filter(Boolean) as Array<{ title: string; body?: ReactNode; text: string }>;

  if (summaryParts.length === 0) {
    return {};
  }

  const detail = createElement(
    "div",
    { className: "space-y-2" },
    ...summaryParts.map((part, index) =>
      createElement(
        "div",
        { key: `${part.text}-${index}` },
        createElement("div", { className: "text-gray-500" }, part.title),
        part.body,
      ),
    ),
  );

  return {
    detail,
    searchText: summaryParts.map((part) => part.text).join(". "),
    changeCount,
  };
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
    const triggerRenderer = getTriggerRenderer(node.trigger?.name || "");
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
      const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
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

  // Current tab: use outputs if available and non-empty, otherwise use metadata
  const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
  const dataSource = hasOutputs ? execution.outputs : execution.metadata || {};
  const flattened = flattenObject(dataSource);

  const currentData = {
    ...flattened,
  };

  // Filter out undefined and empty values
  tabData.current = Object.fromEntries(
    Object.entries(currentData).filter(([_, value]) => value !== undefined && value !== "" && value !== null),
  );

  // Root tab: root event data
  if (execution.rootEvent) {
    const rootTriggerNode = workflowNodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
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

export function buildExecutionInfo(execution: CanvasesCanvasNodeExecution): ExecutionInfo {
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
    input: execution.input!,
    outputs: execution.outputs!,
    rootEvent: buildEventInfo(execution.rootEvent!),
  };
}

export function buildComponentDefinition(component?: Partial<ComponentsComponent>): ComponentDefinition {
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
    componentName: node.type === "TYPE_TRIGGER" ? node.trigger?.name || "" : node.component?.name || "",
    isCollapsed: node.isCollapsed || false,
    configuration: node.configuration,
    metadata: node.metadata,
  };
}
