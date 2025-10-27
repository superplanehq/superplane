import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQueries } from "@tanstack/react-query";

import {
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsWorkflow,
  WorkflowsWorkflowEvent,
} from "@/api-client";

import { useWorkflow, useTriggers, nodeEventsQueryOptions } from "@/hooks/useWorkflowData";
import { CanvasEdge, CanvasNode, CanvasPage } from "@/ui/CanvasPage";
import { getTriggerRenderer } from "./renderers";

export function WorkflowPageV2() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  const { data: triggers = [] } = useTriggers();
  const { data: workflow } = useWorkflow(organizationId!, workflowId!);

  //
  // Get last event for triggers
  // Memoize to prevent unnecessary re-renders and query recreations
  //
  const triggerNodes = useMemo(
    () => workflow?.nodes?.filter((node) => node.type === 'TYPE_TRIGGER') || [],
    [workflow?.nodes]
  );

  const nodeEventsMap = useTriggerNodeEvents(workflowId!, triggerNodes);

  const { nodes, edges } = useMemo(
    () => {
      if (!workflow) return { nodes: [], edges: [] };
      return prepareData(workflow, triggers, nodeEventsMap);
    },
    [workflow, triggers, nodeEventsMap]
  );

  if (!workflow) {
    return null;
  }

  return <CanvasPage title={workflow.name!} nodes={nodes} edges={edges} />;
}

function useTriggerNodeEvents(workflowId: string, triggerNodes: ComponentsNode[]) {
  const results = useQueries({
    queries: triggerNodes.map((node) =>
      nodeEventsQueryOptions(workflowId, node.id!, { limit: 1 })
    ),
  });

  // Build a map of nodeId -> last event
  // Memoize to prevent unnecessary re-renders downstream
  const eventsMap = useMemo(() => {
    const map: Record<string, WorkflowsWorkflowEvent> = {};
    triggerNodes.forEach((node, index) => {
      const result = results[index];
      if (result.data?.events && result.data.events.length > 0) {
        map[node.id!] = result.data.events[0];
      }
    });
    return map;
  }, [results, triggerNodes]);

  return eventsMap;
}

function prepareData(
  data: WorkflowsWorkflow,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, any>
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const nodes = data?.nodes!.map((node) => prepareNode(node, triggers, nodeEventsMap));
  const edges = data?.edges!.map(prepareEdge);

  return { nodes, edges };
}

function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, WorkflowsWorkflowEvent>
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const renderer = getTriggerRenderer(node.trigger?.name || "");
  const lastEvent = nodeEventsMap[node.id!];

  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "trigger",
      label: node.name!,
      state: "pending" as const,
      trigger: renderer.getTriggerProps(node, triggerMetadata!, lastEvent),
    },
  };
}

function prepareCompositeNode(node: ComponentsNode): CanvasNode {
  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "composite",
      label: node.name!,
      state: "pending" as const,
    },
  };
}

function prepareNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  nodeEventsMap: Record<string, any>
): CanvasNode {
  const nodeType = getNodeType(node);

  switch (nodeType) {
    case "trigger":
      return prepareTriggerNode(node, triggers, nodeEventsMap);
    case "composite":
      return prepareCompositeNode(node);
    default:
      return prepareCompositeNode(node);
  }
}

function getNodeType(node: ComponentsNode): string {
  switch (node.type) {
    case 'TYPE_TRIGGER':
      return "trigger";
  }

  return "composite";
}

function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}--${edge.targetId!}--${edge.channel!}`;

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
  };
}
