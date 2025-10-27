import { useMemo } from "react";
import { useParams } from "react-router-dom";
import { useQueries } from "@tanstack/react-query";

import {
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsWorkflow,
  WorkflowsWorkflowEvent,
  WorkflowsWorkflowNodeExecution,
} from "@/api-client";

import { useWorkflow, useTriggers, nodeEventsQueryOptions, nodeExecutionsQueryOptions, nodeQueueItemsQueryOptions } from "@/hooks/useWorkflowData";
import { useBlueprints } from "@/hooks/useBlueprintData";
import { CanvasEdge, CanvasNode, CanvasPage } from "@/ui/CanvasPage";
import { CompositeProps, LastRunState } from "@/ui/composite";
import { getTriggerRenderer } from "./renderers";
import { getColorClass, getBackgroundColorClass } from "@/utils/colors";

export function WorkflowPageV2() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: blueprints = [], isLoading: blueprintsLoading } = useBlueprints(organizationId!);
  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!);

  //
  // Get last event for triggers
  // Memoize to prevent unnecessary re-renders and query recreations
  //
  const triggerNodes = useMemo(
    () => workflow?.nodes?.filter((node) => node.type === 'TYPE_TRIGGER') || [],
    [workflow?.nodes]
  );

  const compositeNodes = useMemo(
    () => workflow?.nodes?.filter((node) => node.type === 'TYPE_BLUEPRINT') || [],
    [workflow?.nodes]
  );

  const nodeEventsMap = useTriggerNodeEvents(workflowId!, triggerNodes);
  const { nodeExecutionsMap, nodeQueueItemsMap } = useCompositeNodeData(workflowId!, compositeNodes);

  const { nodes, edges } = useMemo(
    () => {
      if (!workflow) return { nodes: [], edges: [] };
      return prepareData(workflow, triggers, blueprints, nodeEventsMap, nodeExecutionsMap, nodeQueueItemsMap);
    },
    [workflow, triggers, blueprints, nodeEventsMap, nodeExecutionsMap, nodeQueueItemsMap]
  );

  // Show loading indicator while data is being fetched
  if (workflowLoading || triggersLoading || blueprintsLoading) {
    return null;
  }

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

function useCompositeNodeData(workflowId: string, compositeNodes: ComponentsNode[]) {
  // Fetch last executions for each composite node
  const executionResults = useQueries({
    queries: compositeNodes.map((node) =>
      nodeExecutionsQueryOptions(workflowId, node.id!, { limit: 1 })
    ),
  });

  // Fetch queue items for each composite node
  const queueItemResults = useQueries({
    queries: compositeNodes.map((node) =>
      nodeQueueItemsQueryOptions(workflowId, node.id!)
    ),
  });

  // Build maps of nodeId -> data
  // Memoize to prevent unnecessary re-renders downstream
  const nodeExecutionsMap = useMemo(() => {
    const map: Record<string, any> = {};
    compositeNodes.forEach((node, index) => {
      const result = executionResults[index];
      if (result.data?.executions && result.data.executions.length > 0) {
        map[node.id!] = result.data.executions;
      }
    });
    return map;
  }, [executionResults, compositeNodes]);

  const nodeQueueItemsMap = useMemo(() => {
    const map: Record<string, any> = {};
    compositeNodes.forEach((node, index) => {
      const result = queueItemResults[index];
      if (result.data?.items && result.data.items.length > 0) {
        map[node.id!] = result.data.items;
      }
    });
    return map;
  }, [queueItemResults, compositeNodes]);

  return { nodeExecutionsMap, nodeQueueItemsMap };
}

function prepareData(
  data: WorkflowsWorkflow,
  triggers: TriggersTrigger[],
  blueprints: any[],
  nodeEventsMap: Record<string, any>,
  nodeExecutionsMap: Record<string, any>,
  nodeQueueItemsMap: Record<string, any>
): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const nodes = data?.nodes!.map((node) => prepareNode(node, triggers, blueprints, nodeEventsMap, nodeExecutionsMap, nodeQueueItemsMap));
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

function prepareCompositeNode(
  node: ComponentsNode,
  blueprints: any[],
  nodeExecutionsMap: Record<string, any>,
  nodeQueueItemsMap: Record<string, any>
): CanvasNode {
  const blueprintMetadata = blueprints.find((b) => b.id === node.blueprint?.id);
  const color = blueprintMetadata?.color || "indigo";
  const executions = nodeExecutionsMap[node.id!] || [];
  const queueItems = nodeQueueItemsMap[node.id!] || [];

  let canvasNode: CanvasNode = {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "composite",
      label: node.name!,
      state: "pending" as const,
      composite: {
        iconSlug: blueprintMetadata?.icon || "boxes",
        iconColor: getColorClass(color),
        iconBackground: getBackgroundColorClass(color),
        headerColor: getBackgroundColorClass(color),
        collapsedBackground: getBackgroundColorClass(color),
        title: node.name!,
        description: blueprintMetadata?.description,
        parameters: Object.keys(node.configuration!).map((key) => {
          return {
            icon: "cog",
            items: [`${node.configuration![key]}`],
          };
        })
      },
    },
  };

  if (executions.length > 0) {
    const execution = executions[0];
    (canvasNode.data.composite as CompositeProps).lastRunItem = {
      title: "",
      subtitle: "",
      receivedAt: new Date(execution.createdAt),
      state: getRunItemState(execution),

      //
      // TODO: we should either load child executions from /children endpoint,
      // or return them as part of the execution payload here.
      //
      // TODO: what is ChildEventsInfo.waitingInfos supposed to be???
      //
      childEventsInfo: {},

      //
      // TODO: from the storybook pages, it seems like this comes from the root event.
      // We kind of have this in execution.input, but it's the raw data.
      //
      values: {},
    }
  }

  if (queueItems.length > 0) {
    (canvasNode.data.composite as CompositeProps).nextInQueue = {
      title: "",
      subtitle: "",
      receivedAt: new Date(queueItems[0].createdAt),
    }
  }

  return canvasNode;
}

function getRunItemState(execution: WorkflowsWorkflowNodeExecution): LastRunState {
  if (execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

function prepareNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[],
  blueprints: any[],
  nodeEventsMap: Record<string, any>,
  nodeExecutionsMap: Record<string, any>,
  nodeQueueItemsMap: Record<string, any>
): CanvasNode {
  switch (node.type) {
    case "TYPE_TRIGGER":
      return prepareTriggerNode(node, triggers, nodeEventsMap);
    case "TYPE_BLUEPRINT":
      return prepareCompositeNode(node, blueprints, nodeExecutionsMap, nodeQueueItemsMap);
    default:
      return prepareCompositeNode(node, blueprints, nodeExecutionsMap, nodeQueueItemsMap);
  }
}

function prepareEdge(edge: ComponentsEdge): CanvasEdge {
  const id = `${edge.sourceId!}--${edge.targetId!}--${edge.channel!}`;

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
  };
}
