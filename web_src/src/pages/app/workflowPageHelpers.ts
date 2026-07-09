import type {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
  IntegrationsIntegrationDefinition,
  ActionsAction,
  SuperplaneMeUser,
  TriggersTrigger,
} from "@/api-client";
import type { QueryClient } from "@tanstack/react-query";
import type { CanvasEdge, CanvasNode, SidebarData } from "@/ui/CanvasPage";
import type { LogEntry } from "@/ui/CanvasLogSidebar";
import { getColorClass } from "@/lib/colors";
import { prepareAnnotationNode } from "./lib/canvas-annotation-node";
import { prepareComponentNode, prepareTriggerNode } from "./lib/canvas-node-preparation";
import { getNodeIntegrationName } from "./lib/node-integrations";
import type { TriggerActionModal, User } from "./mappers/types";
import {
  buildUserInfo,
  mapCanvasNodesToLogEntries,
  mapExecutionsToSidebarEvents,
  mapQueueItemsToSidebarEvents,
  mapTriggerEventsToSidebarEvents,
} from "./utils";

export const NO_INCOMING_CONNECTIONS_WARNING = "This node has no incoming connections and will never be triggered.";

type CanvasRunsPage = {
  totalCount?: number;
};

type InfiniteCanvasRunsData = {
  pages?: CanvasRunsPage[];
};

export function getRunningRunsCount(data: InfiniteCanvasRunsData | undefined, visible: boolean): number {
  if (!visible) return 0;
  return data?.pages?.[0]?.totalCount ?? 0;
}

export function getNodeAnalyticsProps(
  node: ComponentsNode,
  availableIntegrations: IntegrationsIntegrationDefinition[],
): { nodeType: string; integration: string | undefined; nodeRef: string | undefined } {
  return {
    nodeType: node.type === "TYPE_TRIGGER" ? "trigger" : node.type === "TYPE_WIDGET" ? "annotation" : "action",
    integration: getNodeIntegrationName(node, availableIntegrations),
    nodeRef: node.component,
  };
}

export function getCanvasLogNodesSignature(nodes: ComponentsNode[]): string {
  return JSON.stringify(
    nodes.map((node) => ({
      id: node.id,
      name: node.name,
      type: node.type,
      component: node.component,
      errorMessage: node.errorMessage,
      warningMessage: node.warningMessage,
    })),
  );
}

export function prepareCanvasLogNodes(
  nodes: ComponentsNode[],
  edges: ComponentsEdge[],
  components: ActionsAction[],
  includeDerivedWarnings: boolean,
): ComponentsNode[] {
  if (!includeDerivedWarnings) {
    return nodes;
  }

  return withDerivedNodeWarnings(nodes, edges, components);
}

export function buildCanvasLogEntries(
  nodes: ComponentsNode[],
  workflowUpdatedAt: string,
  onNodeSelect: (nodeId: string) => void,
): LogEntry[] {
  return mapCanvasNodesToLogEntries({ nodes, workflowUpdatedAt, onNodeSelect }).sort((a, b) => {
    const aTime = Date.parse(a.timestamp || "") || 0;
    const bTime = Date.parse(b.timestamp || "") || 0;
    return aTime - bTime;
  });
}

export function isCanvasPrepLoading(
  canvas: CanvasesCanvas | null | undefined,
  canvasLoading: boolean,
  triggersLoading: boolean,
  componentsLoading: boolean,
  integrationsLoading: boolean,
): boolean {
  return !canvas || canvasLoading || triggersLoading || componentsLoading || integrationsLoading;
}

// Merge a run's lightweight execution ref with the matching full execution (preferred from the
// prefetched event-executions query, falling back to the live store), so mappers receive metadata
// (approval records) and outputs (wait/timegate "pushed through" detection).
export function hydrateRunExecution(
  ref: CanvasesCanvasNodeExecution,
  prefetched: CanvasesCanvasNodeExecution[] | undefined,
  storeExecutions: CanvasesCanvasNodeExecution[] | undefined,
  rootEvent: CanvasesCanvasEvent | undefined,
): CanvasesCanvasNodeExecution {
  const full = prefetched?.find((e) => e.id === ref.id) ?? storeExecutions?.find((e) => e.id === ref.id);
  return {
    ...(full ?? {}),
    ...ref,
    ...(full?.metadata && { metadata: full.metadata }),
    ...(full?.outputs && { outputs: full.outputs }),
    rootEvent,
  } as CanvasesCanvasNodeExecution;
}

export function prepareData(
  workflow: CanvasesCanvas,
  triggers: TriggersTrigger[],
  components: ActionsAction[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  user?: SuperplaneMeUser | null,
  canvasMode: "live" | "edit" = "live",
  openModal?: (modal: TriggerActionModal) => void,
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const currentUser = buildUserInfo(user);
  const edges = workflow?.spec?.edges?.map(prepareEdge) || [];
  const workflowEdges = workflow?.spec?.edges || [];
  const workflowNodes = withDerivedNodeWarnings(workflow?.spec?.nodes || [], workflowEdges, components);
  const nodes =
    workflowNodes
      ?.map((node) => {
        return prepareNode(
          workflowNodes,
          node,
          triggers,
          components,
          nodeEventsMap,
          nodeExecutionsMap,
          nodeQueueItemsMap,
          workflowId,
          queryClient,
          currentUser,
          workflowEdges,
          canvasMode,
          openModal,
        );
      })
      .map((node) => ({
        ...node,
        dragHandle: ".canvas-node-drag-handle",
      })) || [];

  return { nodes, edges };
}

export function withDerivedNodeWarnings(
  nodes: ComponentsNode[],
  edges: ComponentsEdge[],
  components: ActionsAction[],
): ComponentsNode[] {
  const nodesById = nodesByDefinedId(nodes);
  const componentsByName = componentsByDefinedName(components);

  return nodes.map((node) => {
    if (
      node.type !== "TYPE_ACTION" ||
      node.warningMessage ||
      hasValidIncomingConnection(node, edges, nodesById, componentsByName)
    ) {
      return node;
    }

    return {
      ...node,
      warningMessage: NO_INCOMING_CONNECTIONS_WARNING,
    };
  });
}

function hasValidIncomingConnection(
  node: ComponentsNode,
  edges: ComponentsEdge[],
  nodesById: Map<string, ComponentsNode>,
  componentsByName: Map<string, ActionsAction>,
): boolean {
  return edges.some((edge) => {
    if (!node.id || edge.targetId !== node.id || !edge.sourceId) {
      return false;
    }

    const sourceNode = nodesById.get(edge.sourceId);
    if (!sourceNode) {
      return false;
    }

    return getSourceOutputChannels(sourceNode, componentsByName).has(edge.channel || "default");
  });
}

function getSourceOutputChannels(node: ComponentsNode, componentsByName: Map<string, ActionsAction>): Set<string> {
  if (node.type !== "TYPE_ACTION") {
    return new Set(["default"]);
  }

  const outputChannels = componentsByName
    .get(node.component || "")
    ?.outputChannels?.map((channel) => channel.name)
    .filter((name): name is string => !!name);
  return new Set(outputChannels?.length ? outputChannels : ["default"]);
}

function nodesByDefinedId(nodes: ComponentsNode[]): Map<string, ComponentsNode> {
  const nodesById = new Map<string, ComponentsNode>();
  for (const node of nodes) {
    if (node.id) {
      nodesById.set(node.id, node);
    }
  }
  return nodesById;
}

function componentsByDefinedName(components: ActionsAction[]): Map<string, ActionsAction> {
  const componentsByName = new Map<string, ActionsAction>();
  for (const component of components) {
    if (component.name) {
      componentsByName.set(component.name, component);
    }
  }
  return componentsByName;
}

export function prepareNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  components: ActionsAction[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  currentUser?: User,
  edges?: ComponentsEdge[],
  canvasMode: "live" | "edit" = "live",
  openModal?: (modal: TriggerActionModal) => void,
): CanvasNode {
  switch (node.type) {
    case "TYPE_TRIGGER":
      return prepareTriggerNode(node, triggers, nodeEventsMap, canvasMode, {
        canvasId: workflowId,
        openModal,
      });
    case "TYPE_WIDGET":
      return prepareAnnotationNode(node);

    default:
      return prepareComponentNode({
        nodes,
        node,
        components,
        nodeExecutionsMap,
        nodeQueueItemsMap,
        canvasId: workflowId,
        queryClient,
        currentUser,
        edges,
        canvasMode,
      });
  }
}

export function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}-targets->${edge.targetId!}-using->${edge.channel!}`;

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
    sourceHandle: edge.channel!,
  };
}

export function prepareSidebarData(
  node: ComponentsNode,
  nodes: ComponentsNode[],
  components: ActionsAction[],
  triggers: TriggersTrigger[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  totalHistoryCount?: number,
  totalQueueCount?: number,
): SidebarData {
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];
  const events = nodeEventsMap[node.id!] || [];

  // Get metadata based on node type
  const componentMetadata = node.type === "TYPE_ACTION" ? components.find((c) => c.name === node.component) : undefined;
  const triggerMetadata = node.type === "TYPE_TRIGGER" ? triggers.find((t) => t.name === node.component) : undefined;

  const nodeTitle = componentMetadata?.label || triggerMetadata?.label || node.name || "Unknown";
  let iconSlug = "boxes";
  let color = "indigo";

  if (componentMetadata) {
    iconSlug = componentMetadata.icon || iconSlug;
    color = componentMetadata.color || color;
  } else if (triggerMetadata) {
    iconSlug = triggerMetadata.icon || iconSlug;
    color = triggerMetadata.color || color;
  }

  const latestEvents =
    node.type === "TYPE_TRIGGER"
      ? mapTriggerEventsToSidebarEvents(events, node, 5)
      : mapExecutionsToSidebarEvents(executions, nodes, 5);

  // Convert queue items to sidebar events (next in queue)
  const nextInQueueEvents = mapQueueItemsToSidebarEvents(queueItems, nodes, 5);
  const hideQueueEvents = node.type === "TYPE_TRIGGER";

  return {
    latestEvents,
    nextInQueueEvents,
    title: nodeTitle,
    iconSlug,
    iconColor: getColorClass(color),
    totalInHistoryCount: totalHistoryCount ? totalHistoryCount : 0,
    totalInQueueCount: totalQueueCount ? totalQueueCount : 0,
    hideQueueEvents,
    isComposite: false,
  };
}

function readErrorField(error: unknown, key: string): unknown {
  if (typeof error !== "object" || error === null) {
    return undefined;
  }
  return (error as Record<string, unknown>)[key];
}

/**
 * True when an API call failed because the targeted resource does not exist
 * (HTTP 404 / gRPC NOT_FOUND / "not found" message). Used to turn a stale
 * draft-version id into a graceful recovery instead of an opaque error.
 */
export function isNotFoundError(error: unknown): boolean {
  if (readErrorField(error, "status") === 404) {
    return true;
  }

  const response = readErrorField(error, "response");
  if (typeof response === "object" && response !== null && readErrorField(response, "status") === 404) {
    return true;
  }

  if (readErrorField(error, "code") === "NOT_FOUND") {
    return true;
  }

  const message = readErrorField(error, "message");
  if (typeof message !== "string") {
    return false;
  }

  return message.includes("not found") || message.includes("404");
}

/**
 * True when an API call failed because the request was invalid
 * (HTTP 400 / gRPC INVALID_ARGUMENT).
 */
export function isInvalidArgumentError(error: unknown): boolean {
  if (readErrorField(error, "status") === 400) {
    return true;
  }

  const response = readErrorField(error, "response");
  if (typeof response === "object" && response !== null && readErrorField(response, "status") === 400) {
    return true;
  }

  if (readErrorField(error, "code") === "INVALID_ARGUMENT") {
    return true;
  }

  return false;
}

/** True when a canvas fetch failed because the canvas does not exist. */
export function isCanvasLoadNotFoundError(error: unknown): boolean {
  return isNotFoundError(error);
}

const RUN_ID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function isValidRunId(runId: string): boolean {
  return RUN_ID_PATTERN.test(runId);
}

export function shouldClearStaleRunUrl({
  selectedRunId,
  isRunInspectionMode,
  selectedRun,
  isRunResolveLoading,
  describeRunSettled,
}: {
  selectedRunId: string | null;
  isRunInspectionMode: boolean;
  selectedRun: unknown;
  isRunResolveLoading: boolean;
  describeRunSettled: boolean;
}): boolean {
  if (!selectedRunId || !isRunInspectionMode) return false;
  if (isRunResolveLoading) return false;
  if (selectedRun) return false;
  if (!isValidRunId(selectedRunId)) return true;
  return describeRunSettled;
}

export function shouldClearRunDetailNode({
  runDetailNodeId,
  participantNodeIds,
  runCanvasLoading,
  runCanvasSettled,
}: {
  runDetailNodeId: string | null;
  participantNodeIds: string[];
  runCanvasLoading: boolean;
  runCanvasSettled: boolean;
}): boolean {
  if (!runDetailNodeId || runCanvasLoading || !runCanvasSettled) return false;
  if (participantNodeIds.length === 0) return true;
  return !participantNodeIds.includes(runDetailNodeId);
}

export function clearRunDetailNodeSearchParams(searchParams: URLSearchParams, nodeId: string): URLSearchParams {
  const next = new URLSearchParams(searchParams);
  if (next.get("sidebar") === "1" && next.get("node") === nodeId) {
    next.delete("sidebar");
    next.delete("node");
  }
  return next;
}
