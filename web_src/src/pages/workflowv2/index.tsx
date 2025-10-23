import { useParams } from "react-router-dom";
import {
  ComponentsEdge,
  ComponentsNode,
  WorkflowsWorkflow,
} from "../../api-client";
import { useWorkflow } from "../../hooks/useWorkflowData";
import { CanvasPage } from "../../ui/CanvasPage";

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
  nodes: CanvasPage.Node[];
  edges: CanvasPage.Edge[];
} {
  const nodes = data?.nodes!.map(prepareNode);
  const edges = data?.edges!.map(prepareEdge);

  return { nodes, edges };
}

function prepareNode(node: ComponentsNode): CanvasPage.Node {
  return {
    id: node.id!,
    position: { x: -140, y: -80 },
    data: {
      label: node.name!,
    },
  };
}

function prepareEdge(edge: ComponentsEdge): CanvasPage.Edge {
  return {
    // id: edge.id!,
    // source: edge.sourceNodeId!,
    // target: edge.targetNodeId!,
  };
}
