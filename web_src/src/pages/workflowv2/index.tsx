import { useParams } from "react-router-dom";

import {
  ComponentsEdge,
  ComponentsNode,
  WorkflowsWorkflow,
} from "../../api-client";

import { useWorkflow } from "../../hooks/useWorkflowData";
import { CanvasPage } from "../../ui/CanvasPage";
import type { Edge as ReactFlowEdge, Node as ReactFlowNode } from "@xyflow/react";

export function WorkflowPageV2() {
  const data = usePageData();

  const { nodes, edges } = prepareData(data!);

  return <CanvasPage nodes={nodes} edges={edges} />;
}

function usePageData() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  return useWorkflow(organizationId!, workflowId!).data;
}

function prepareData(data: WorkflowsWorkflow): {
  nodes: ReactFlowNode[];
  edges: ReactFlowEdge[];
} {
  const nodes = data?.nodes!.map(prepareNode);
  const edges = data?.edges!.map(prepareEdge);

  return { nodes, edges };
}

function prepareNode(node: ComponentsNode): ReactFlowNode {
  return {
    id: node.id!,
    position: { x: -140, y: -80 },
    data: {
      type: "composite",
      label: node.name!,
    },
  };
}

function prepareEdge(edge: ComponentsEdge): ReactFlowEdge {
  const id = `${edge.sourceId!}--${edge.targetId!}--${edge.channel!}`;
  console.log("Preparing edge with id:", id);

  return {
    id: id,
    source: edge.sourceId!,
    target: edge.targetId!,
  };
}
