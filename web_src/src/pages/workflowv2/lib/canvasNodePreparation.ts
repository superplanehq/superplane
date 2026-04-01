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
import type { ComponentBaseMapper } from "../mappers/types";
import { getComponentAdditionalDataBuilder, getComponentBaseMapper, getTriggerRenderer } from "../mappers";
import { CANVAS_BUNDLE_COLOR, CANVAS_BUNDLE_ICON_SLUG } from "./canvasBundle";
import { buildComponentFallbackCanvasNode, buildTriggerFallbackCanvasNode } from "./canvasNodeFallback";
import {
  buildComponentDefinition,
  buildEventInfo,
  buildExecutionInfo,
  buildNodeInfo,
  buildQueueItemInfo,
  getNextInQueueInfo,
} from "../utils";

type PrepareComponentNodeArgs = {
  nodes: ComponentsNode[];
  node: ComponentsNode;
  components: ComponentsComponent[];
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
  workflowId: string;
  queryClient: QueryClient;
  organizationId?: string;
  currentUser?: { id?: string; email?: string };
  edges?: ComponentsEdge[];
};

type PrepareComponentBaseNodeArgs = {
  nodes: ComponentsNode[];
  node: ComponentsNode;
  components: ComponentsComponent[];
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
  workflowId: string;
  queryClient: QueryClient;
  organizationId: string;
  currentUser?: { id?: string; email?: string };
  edges?: ComponentsEdge[];
};

type NodePosition = {
  x: number;
  y: number;
};

function getRunItemState(execution: CanvasesCanvasNodeExecution): LastRunState {
  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

function getNodePosition(node: ComponentsNode): NodePosition {
  return {
    x: node.position?.x ?? 0,
    y: node.position?.y ?? 0,
  };
}

function getTriggerDisplayLabel(node: ComponentsNode, triggerMetadata?: TriggersTrigger): string {
  return node.name || triggerMetadata?.label || node.trigger?.name || "Trigger";
}

function buildPreparedTriggerCanvasNode(args: {
  node: ComponentsNode;
  triggerMetadata?: TriggersTrigger;
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  displayLabel: string;
  position: NodePosition;
}): CanvasNode {
  const { node, triggerMetadata, nodeEventsMap, displayLabel, position } = args;
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
}

function buildCompositeFieldLabelMap(blueprintMetadata?: BlueprintsBlueprint): Record<string, string> {
  const configurationFields = blueprintMetadata?.configuration || [];

  return configurationFields.reduce<Record<string, string>>((acc, field) => {
    if (field.name) {
      acc[field.name] = field.label || field.name;
    }
    return acc;
  }, {});
}

function buildCompositeParameters(
  node: ComponentsNode,
  fieldLabelMap: Record<string, string>,
): CompositeProps["parameters"] {
  const configuration = node.configuration || {};
  const configurationKeys = Object.keys(configuration);

  if (configurationKeys.length === 0) {
    return [];
  }

  return [
    {
      icon: "cog",
      items: configurationKeys.reduce(
        (acc, key) => {
          const displayKey = fieldLabelMap[key] || key;
          acc[displayKey] = `${configuration[key]}`;
          return acc;
        },
        {} as Record<string, string>,
      ),
    },
  ];
}

function buildCompositeCanvasNode(args: {
  node: ComponentsNode;
  displayLabel: string;
  blueprintMetadata?: BlueprintsBlueprint;
  isMissing: boolean;
  position: { x: number; y: number };
  parameters: CompositeProps["parameters"];
}): CanvasNode {
  const { node, displayLabel, blueprintMetadata, isMissing, position, parameters } = args;

  return {
    id: node.id!,
    position,
    data: {
      type: "composite",
      label: displayLabel,
      state: "pending" as const,
      outputChannels: blueprintMetadata?.outputChannels?.map((c) => c.name!) || ["default"],
      composite: {
        iconSlug: CANVAS_BUNDLE_ICON_SLUG,
        iconColor: getColorClass(CANVAS_BUNDLE_COLOR),
        collapsedBackground: getBackgroundColorClass(CANVAS_BUNDLE_COLOR),
        collapsed: node.isCollapsed,
        title: displayLabel,
        description: blueprintMetadata?.description,
        isMissing,
        error: node.errorMessage,
        warning: node.warningMessage,
        paused: !!node.paused,
        parameters,
      },
    },
  };
}

function appendCompositeLastRunItem(
  canvasNode: CanvasNode,
  execution: CanvasesCanvasNodeExecution,
  nodes: ComponentsNode[],
) {
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

function appendCompositeQueueInfo(
  canvasNode: CanvasNode,
  nodeId: string,
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
  nodes: ComponentsNode[],
) {
  const nextInQueueInfo = getNextInQueueInfo(nodeQueueItemsMap, nodeId, nodes);
  if (nextInQueueInfo) {
    (canvasNode.data.composite as CompositeProps).nextInQueue = nextInQueueInfo;
  }
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
        },
        error: "Select a component from the sidebar",
        parameters: [],
      },
    },
  };
}

function buildComponentAdditionalData(args: {
  componentDef?: ComponentsComponent;
  node: ComponentsNode;
  nodes: ComponentsNode[];
  executions: CanvasesCanvasNodeExecution[];
  workflowId: string;
  queryClient: QueryClient;
  organizationId: string;
  currentUser?: { id?: string; email?: string };
  edges?: ComponentsEdge[];
}) {
  const { componentDef, node, nodes, executions, workflowId, queryClient, organizationId, currentUser, edges } = args;

  if (!componentDef) {
    return undefined;
  }

  return getComponentAdditionalDataBuilder(node.component?.name || "")?.buildAdditionalData({
    nodes: nodes.map((n) => buildNodeInfo(n)),
    node: buildNodeInfo(node),
    componentDefinition: buildComponentDefinition(componentDef),
    lastExecutions: executions.map((e) => buildExecutionInfo(e)),
    edges,
    canvasId: workflowId,
    queryClient,
    organizationId,
    currentUser,
  });
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
  };
}

export function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>,
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const displayLabel = getTriggerDisplayLabel(node, triggerMetadata);
  const position = getNodePosition(node);

  try {
    return buildPreparedTriggerCanvasNode({
      node,
      triggerMetadata,
      nodeEventsMap,
      displayLabel,
      position,
    });
  } catch (error) {
    console.error(`[CanvasPage] Failed to prepare trigger node "${node.id}":`, error);
    return buildTriggerFallbackCanvasNode({ node, displayLabel, triggerMetadata });
  }
}

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
  const position = getNodePosition(node);

  const fieldLabelMap = buildCompositeFieldLabelMap(blueprintMetadata);
  const parameters = buildCompositeParameters(node, fieldLabelMap);
  const canvasNode = buildCompositeCanvasNode({
    node,
    displayLabel,
    blueprintMetadata,
    isMissing,
    position,
    parameters,
  });

  if (executions.length > 0) {
    appendCompositeLastRunItem(canvasNode, executions[0], nodes);
  }

  appendCompositeQueueInfo(canvasNode, node.id!, nodeQueueItemsMap, nodes);

  return canvasNode;
}

export function prepareComponentNode(args: PrepareComponentNodeArgs): CanvasNode {
  const {
    nodes,
    node,
    components,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    organizationId,
    currentUser,
    edges,
  } = args;
  const isPlaceholder = !node.component?.name && node.name === "New Component";

  if (isPlaceholder) {
    return buildPlaceholderComponentNode(node);
  }

  return prepareComponentBaseNode({
    nodes,
    node,
    components,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    organizationId: organizationId || "",
    currentUser,
    edges,
  });
}

export function prepareComponentBaseNode(args: PrepareComponentBaseNodeArgs): CanvasNode {
  const {
    nodes,
    node,
    components,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    workflowId,
    queryClient,
    organizationId,
    currentUser,
    edges,
  } = args;
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
    const additionalData = buildComponentAdditionalData({
      componentDef,
      node,
      nodes,
      executions,
      workflowId,
      queryClient,
      organizationId,
      currentUser,
      edges,
    });

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

    const emptyStateProps = resolveComponentEmptyStateProps(componentBaseProps, node);

    return {
      id: node.id!,
      position: getNodePosition(node),
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
