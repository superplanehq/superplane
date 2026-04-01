import type { QueryClient } from "@tanstack/react-query";
import { Puzzle } from "lucide-react";
import type {
  BlueprintsBlueprint,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
} from "@/api-client";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { CanvasNode } from "@/ui/CanvasPage";
import type { CompositeProps, LastRunState } from "@/ui/composite";
import { getComponentAdditionalDataBuilder, getComponentBaseMapper, getTriggerRenderer } from "../mappers";
import { buildComponentFallbackCanvasNode, buildTriggerFallbackCanvasNode } from "./canvasNodeFallback";
import {
  buildComponentDefinition,
  buildEventInfo,
  buildExecutionInfo,
  buildNodeInfo,
  buildQueueItemInfo,
  getNextInQueueInfo,
} from "../utils";

const BUNDLE_ICON_SLUG = "component";
const BUNDLE_COLOR = "gray";

function getRunItemState(execution: CanvasesCanvasNodeExecution): LastRunState {
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

export function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const displayLabel = node.name || triggerMetadata?.label || node.trigger?.name || "Trigger";
  const position = {
    x: node.position?.x ?? 0,
    y: node.position?.y ?? 0,
  };

  try {
    const renderer = getTriggerRenderer(node.trigger?.name || "");
    const lastEvent = nodeEventsMap[node.id!]?.[0];
    const triggerProps = renderer.getTriggerProps({
      node: buildNodeInfo(node),
      definition: buildComponentDefinition(triggerMetadata),
      lastEvent: buildEventInfo(lastEvent),
    });

    return {
      id: node.id!,
      position,
      data: {
        type: "trigger",
        label: displayLabel,
        state: "pending" as const,
        outputChannels: ["default"],
        _triggerName: node.trigger?.name,
        trigger: {
          ...triggerProps,
          collapsed: node.isCollapsed,
          error: node.errorMessage || triggerProps.error,
          warning: node.warningMessage || triggerProps.warning,
        },
      },
    };
  } catch (error) {
    console.error(`[CanvasPage] Failed to prepare trigger node "${node.id}":`, error);
    return buildTriggerFallbackCanvasNode({ node, displayLabel, triggerMetadata });
  }
}

// eslint-disable-next-line complexity
export function prepareCompositeNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  blueprints: BlueprintsBlueprint[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
): CanvasNode {
  const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
  const isMissing = !blueprintMetadata;
  const executions = nodeExecutionsMap[node.id!] || [];
  const displayLabel = node.name || blueprintMetadata?.name || "Composite";
  const position = {
    x: node.position?.x ?? 0,
    y: node.position?.y ?? 0,
  };

  const configurationFields = blueprintMetadata?.configuration || [];
  const fieldLabelMap = configurationFields.reduce<Record<string, string>>((acc, field) => {
    if (field.name) {
      acc[field.name] = field.label || field.name;
    }
    return acc;
  }, {});

  const canvasNode: CanvasNode = {
    id: node.id!,
    position,
    data: {
      type: "composite",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: blueprintMetadata?.outputChannels?.map((c) => c.name!) || ["default"],
      composite: {
        iconSlug: BUNDLE_ICON_SLUG,
        iconColor: getColorClass(BUNDLE_COLOR),
        collapsedBackground: getBackgroundColorClass(BUNDLE_COLOR),
        collapsed: node.isCollapsed,
        title: displayLabel,
        description: blueprintMetadata?.description,
        isMissing,
        error: node.errorMessage,
        warning: node.warningMessage,
        paused: !!node.paused,
        parameters:
          Object.keys(node.configuration!).length > 0
            ? [
                {
                  icon: "cog",
                  items: Object.keys(node.configuration!).reduce(
                    (acc, key) => {
                      const displayKey = fieldLabelMap[key] || key;
                      acc[displayKey] = `${node.configuration![key]}`;
                      return acc;
                    },
                    {} as Record<string, string>,
                  ),
                },
              ]
            : [],
      },
    },
  };

  if (executions.length > 0) {
    const execution = executions[0];
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");
    const eventInfo = buildEventInfo(execution.rootEvent!);
    const { title, subtitle } = rootTriggerRenderer.getTitleAndSubtitle({ event: eventInfo });
    (canvasNode.data.composite as CompositeProps).lastRunItem = {
      title,
      subtitle,
      id: execution.rootEvent?.id,
      receivedAt: new Date(execution.createdAt!),
      state: getRunItemState(execution),
      values: rootTriggerRenderer.getRootEventValues({ event: eventInfo }),
      childEventsInfo: {
        count: execution.childExecutions?.length || 0,
        waitingInfos: [],
      },
    };
  }

  const nextInQueueInfo = getNextInQueueInfo(nodeQueueItemsMap, node.id!, nodes);
  if (nextInQueueInfo) {
    (canvasNode.data.composite as CompositeProps).nextInQueue = nextInQueueInfo;
  }

  return canvasNode;
}

// eslint-disable-next-line max-params
export function prepareComponentNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId?: string,
  currentUser?: { id?: string; email?: string },
  edges?: ComponentsEdge[],
): CanvasNode {
  const isPlaceholder = !node.component?.name && node.name === "New Component";

  if (isPlaceholder) {
    return {
      id: node.id!,
      position: { x: node.position?.x ?? 0, y: node.position?.y ?? 0 },
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
          },
          error: "Select a component from the sidebar",
          parameters: [],
        },
      },
    };
  }

  return prepareComponentBaseNode(
    nodes,
    node,
    components,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    organizationId || "",
    currentUser,
    edges,
  );
}

// eslint-disable-next-line max-params, complexity
export function prepareComponentBaseNode(
  nodes: ComponentsNode[],
  node: ComponentsNode,
  components: ComponentsComponent[],
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  workflowId: string,
  queryClient: QueryClient,
  organizationId: string,
  currentUser?: { id?: string; email?: string },
  edges?: ComponentsEdge[],
): CanvasNode {
  const executions = nodeExecutionsMap[node.id!] || [];
  const metadata = components.find((c) => c.name === node.component?.name);
  const displayLabel = node.name || metadata?.label || node.component?.name || "Component";
  const componentDef = components.find((c) => c.name === node.component?.name);
  const fallbackComponentDef = componentDef || {
    name: node.component?.name,
    label: node.name,
  };
  const nodeQueueItems = nodeQueueItemsMap?.[node.id!];

  try {
    const additionalData = componentDef
      ? getComponentAdditionalDataBuilder(node.component?.name || "")?.buildAdditionalData({
          nodes: nodes.map((n) => buildNodeInfo(n)),
          node: buildNodeInfo(node),
          componentDefinition: buildComponentDefinition(componentDef),
          lastExecutions: executions.map((e) => buildExecutionInfo(e)),
          edges,
          canvasId: workflowId,
          queryClient,
          organizationId,
          currentUser,
        })
      : undefined;

    const componentBaseProps = getComponentBaseMapper(node.component?.name || "").props({
      nodes: nodes.map((n) => buildNodeInfo(n)),
      node: buildNodeInfo(node),
      componentDefinition: buildComponentDefinition(fallbackComponentDef),
      lastExecutions: executions.map((e) => buildExecutionInfo(e)),
      nodeQueueItems: nodeQueueItems?.map((q) => buildQueueItemInfo(q)),
      additionalData,
    });

    if (!componentBaseProps.iconSrc) {
      const resolvedIconSrc = getHeaderIconSrc(node.component?.name);
      if (resolvedIconSrc) {
        componentBaseProps.iconSrc = resolvedIconSrc;
      }
    }

    const hasError = !!node.errorMessage;
    const showingEmptyState = componentBaseProps.includeEmptyState;
    const emptyStateProps =
      hasError && showingEmptyState
        ? {
            ...componentBaseProps.emptyStateProps,
            icon: componentBaseProps.emptyStateProps?.icon || Puzzle,
            title: "Finish configuring this component",
          }
        : componentBaseProps.emptyStateProps;

    return {
      id: node.id!,
      position: { x: node.position?.x || 0, y: node.position?.y || 0 },
      data: {
        type: "component",
        label: displayLabel,
        state: "pending" as const,
        outputChannels: metadata?.outputChannels?.map((channel) => channel.name) || ["default"],
        _componentName: node.component?.name,
        component: {
          ...componentBaseProps,
          emptyStateProps,
          error: node.errorMessage || componentBaseProps.error,
          warning: node.warningMessage || componentBaseProps.warning,
          paused: !!node.paused,
        },
      },
    };
  } catch (error) {
    console.error(`[CanvasPage] Failed to prepare component node "${node.id}":`, error);
    return buildComponentFallbackCanvasNode({ node, displayLabel, metadata });
  }
}
