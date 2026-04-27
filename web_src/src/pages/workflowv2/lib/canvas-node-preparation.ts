import type { QueryClient } from "@tanstack/react-query";
import { Puzzle } from "lucide-react";
import {
  canvasesInvokeNodeExecutionHook,
  type CanvasesCanvasEvent,
  type CanvasesCanvasNodeExecution,
  type CanvasesCanvasNodeQueueItem,
  type SuperplaneActionsAction,
  type SuperplaneComponentsEdge as ComponentsEdge,
  type SuperplaneComponentsNode as ComponentsNode,
  type TriggersTrigger,
} from "@/api-client";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { CanvasNode } from "@/ui/CanvasPage";
import type { ActionContext, ComponentBaseMapper, User } from "../mappers/types";
import { getComponentBaseMapper, getTriggerRenderer } from "../mappers";
import { buildComponentFallbackCanvasNode, buildTriggerFallbackCanvasNode } from "./canvas-node-fallback";

import {
  buildComponentDefinition,
  buildEventInfo,
  buildExecutionInfo,
  buildNodeInfo,
  buildQueueItemInfo,
  buildUserInfo,
} from "../utils";
import { canvasKeys } from "@/hooks/useCanvasData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

type PrepareComponentNodeArgs = {
  nodes: ComponentsNode[];
  node: ComponentsNode;
  components: SuperplaneActionsAction[];
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
  canvasId: string;
  queryClient: QueryClient;
  organizationId?: string;
  currentUser?: User;
  edges?: ComponentsEdge[];
  canvasMode?: "live" | "edit";
};

type PrepareComponentBaseNodeArgs = {
  nodes: ComponentsNode[];
  node: ComponentsNode;
  components: SuperplaneActionsAction[];
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
  canvasId: string;
  queryClient: QueryClient;
  currentUser?: User;
  edges?: ComponentsEdge[];
  canvasMode?: "live" | "edit";
};

type NodePosition = {
  x: number;
  y: number;
};

function getNodePosition(node: ComponentsNode): NodePosition {
  return {
    x: node.position?.x ?? 0,
    y: node.position?.y ?? 0,
  };
}

function getTriggerDisplayLabel(node: ComponentsNode, triggerMetadata?: TriggersTrigger): string {
  return node.name || triggerMetadata?.label || node.component || "Trigger";
}

function buildPreparedTriggerCanvasNode(args: {
  node: ComponentsNode;
  triggerMetadata?: TriggersTrigger;
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  displayLabel: string;
  position: NodePosition;
  canvasMode?: "live" | "edit";
}): CanvasNode {
  const { node, triggerMetadata, nodeEventsMap, displayLabel, position, canvasMode = "live" } = args;
  const renderer = getTriggerRenderer(node.component || "");
  const lastEvent = nodeEventsMap[node.id!]?.[0];
  const triggerProps = renderer.getTriggerProps({
    node: buildNodeInfo(node),
    definition: buildComponentDefinition(triggerMetadata),
    lastEvent: buildEventInfo(lastEvent),
    canvasMode,
  });

  return {
    id: node.id!,
    position,
    data: {
      type: "trigger",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: ["default"],
      trigger: {
        ...triggerProps,
        collapsed: node.isCollapsed,
        error: node.errorMessage,
        warning: node.warningMessage,
      },
    },
  };
}

function buildPlaceholderComponentNode(node: ComponentsNode): CanvasNode {
  return {
    id: node.id!,
    position: getNodePosition(node),
    data: {
      type: "component",
      label: "New Component",
      state: "pending" as const,
      outputChannels: ["default"],
      component: {
        iconSlug: "box-dashed",
        iconColor: getColorClass("gray"),
        collapsedBackground: getBackgroundColorClass("gray"),
        collapsed: false,
        title: "New Component",
        includeEmptyState: true,
        emptyStateProps: {
          icon: Puzzle,
          title: "Select a component from the sidebar",
          purpose: "setup",
        },
        error: "Select a component from the sidebar",
        parameters: [],
      },
    },
  };
}

function resolveComponentEmptyStateProps(
  componentBaseProps: ReturnType<ComponentBaseMapper["props"]>,
  node: ComponentsNode,
) {
  const hasError = !!node.errorMessage;
  const showingEmptyState = componentBaseProps.includeEmptyState;

  if (!hasError || !showingEmptyState) {
    return componentBaseProps.emptyStateProps;
  }

  return {
    ...componentBaseProps.emptyStateProps,
    icon: componentBaseProps.emptyStateProps?.icon || Puzzle,
    title: "Finish configuring this component",
    purpose: "setup",
  };
}

export function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
  canvasMode: "live" | "edit" = "live",
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.component);
  const displayLabel = getTriggerDisplayLabel(node, triggerMetadata);
  const position = getNodePosition(node);

  try {
    return buildPreparedTriggerCanvasNode({
      node,
      triggerMetadata,
      nodeEventsMap,
      displayLabel,
      position,
      canvasMode,
    });
  } catch (error) {
    console.error(`[CanvasPage] Failed to prepare trigger node "${node.id}":`, error);
    return buildTriggerFallbackCanvasNode({ node, displayLabel, triggerMetadata });
  }
}

export function prepareComponentNode(args: PrepareComponentNodeArgs): CanvasNode {
  const { nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap, canvasId, queryClient, currentUser, edges } =
    args;
  const isPlaceholder = !node.component && node.name === "New Component";

  if (isPlaceholder) {
    return buildPlaceholderComponentNode(node);
  }

  return prepareComponentBaseNode({
    nodes,
    node,
    components,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    canvasId,
    queryClient,
    currentUser,
    edges,
    canvasMode: args.canvasMode,
  });
}

export function prepareComponentBaseNode(args: PrepareComponentBaseNodeArgs): CanvasNode {
  const { nodes, node, components, nodeExecutionsMap, nodeQueueItemsMap, canvasId, queryClient, currentUser } = args;
  const executions = nodeExecutionsMap[node.id!] || [];
  const metadata = components.find((c) => c.name === node.component);
  const displayLabel = node.name || metadata?.label || node.component || "Component";
  const componentDef = components.find((c) => c.name === node.component);
  const fallbackComponentDef = componentDef || {
    name: node.component,
    label: node.name,
  };
  const nodeQueueItems = nodeQueueItemsMap?.[node.id!];

  try {
    const componentBaseProps = getComponentBaseMapper(node.component || "").props({
      nodes: nodes.map((n) => buildNodeInfo(n)),
      node: buildNodeInfo(node),
      componentDefinition: buildComponentDefinition(fallbackComponentDef),
      lastExecutions: executions.map((e) => buildExecutionInfo(e)),
      nodeQueueItems: nodeQueueItems?.map((q) => buildQueueItemInfo(q)),
      currentUser: buildUserInfo(currentUser),
      actions: buildActionContext(queryClient, canvasId, node.id!),
      canvasMode: args.canvasMode,
    });

    if (!componentBaseProps.iconSrc) {
      const resolvedIconSrc = getHeaderIconSrc(node.component);
      if (resolvedIconSrc) {
        componentBaseProps.iconSrc = resolvedIconSrc;
      }
    }

    const emptyStateProps = resolveComponentEmptyStateProps(componentBaseProps, node);

    return {
      id: node.id!,
      position: getNodePosition(node),
      data: {
        type: "component",
        label: displayLabel,
        state: "pending" as const,
        outputChannels: metadata?.outputChannels?.map((channel) => channel.name) || ["default"],
        component: {
          ...componentBaseProps,
          emptyStateProps,
          error: node.errorMessage,
          warning: node.warningMessage,
          paused: !!node.paused,
        },
      },
    };
  } catch (error) {
    console.error(`[CanvasPage] Failed to prepare component node "${node.id}":`, error);
    return buildComponentFallbackCanvasNode({ node, displayLabel, metadata });
  }
}

function buildActionContext(queryClient: QueryClient, canvasId: string, nodeId: string): ActionContext {
  return {
    invokeNodeExecutionHook: async (executionId: string, hookName: string, parameters: unknown) => {
      try {
        await canvasesInvokeNodeExecutionHook(
          withOrganizationHeader({
            path: {
              canvasId,
              executionId,
              hookName,
            },
            body: {
              parameters,
            },
          }),
        );
        queryClient.invalidateQueries({
          queryKey: canvasKeys.nodeExecution(canvasId, nodeId),
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "failed to invoke hook"));
      }
    },
  };
}
