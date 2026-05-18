import type {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  SuperplaneComponentsEdge as ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
  IntegrationsIntegrationDefinition,
  SuperplaneActionsAction,
  SuperplaneMeUser,
  TriggersTrigger,
} from "@/api-client";
import type { QueryClient } from "@tanstack/react-query";
import type { CanvasEdge, CanvasNode, SidebarData } from "@/ui/CanvasPage";
import { getColorClass } from "@/lib/colors";
import { prepareAnnotationNode } from "./lib/canvas-annotation-node";
import { prepareComponentNode, prepareTriggerNode } from "./lib/canvas-node-preparation";
import { getNodeIntegrationName } from "./lib/node-integrations";
import type { TriggerActionModal, User } from "./mappers/types";
import {
  buildUserInfo,
  mapExecutionsToSidebarEvents,
  mapQueueItemsToSidebarEvents,
  mapTriggerEventsToSidebarEvents,
} from "./utils";

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
  components: SuperplaneActionsAction[],
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
  const workflowNodes = workflow?.spec?.nodes || [];
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

export function prepareNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  components: SuperplaneActionsAction[],
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
  components: SuperplaneActionsAction[],
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
