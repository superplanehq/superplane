import { useParams } from "react-router-dom";

import {
  ComponentsEdge,
  ComponentsNode,
  TriggersTrigger,
  WorkflowsWorkflow,
} from "@/api-client";

import { useWorkflow, useTriggers } from "@/hooks/useWorkflowData";
import { CanvasEdge, CanvasNode, CanvasPage } from "@/ui/CanvasPage";
import { getTriggerRenderer } from "./renderers";

export function WorkflowPageV2() {
  const data = usePageData();
  const { data: triggers = [] } = useTriggers();

  const { nodes, edges } = prepareData(data!, triggers);

  return <CanvasPage nodes={nodes} edges={edges} />;
}

function usePageData() {
  const { organizationId, workflowId } = useParams<{
    organizationId: string;
    workflowId: string;
  }>();

  return useWorkflow(organizationId!, workflowId!).data;
}

function prepareData(data: WorkflowsWorkflow, triggers: TriggersTrigger[]): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
} {
  const nodes = data?.nodes!.map((node) => prepareNode(node, triggers));
  const edges = data?.edges!.map(prepareEdge);

  return { nodes, edges };
}

function prepareTriggerNode(
  node: ComponentsNode,
  triggers: TriggersTrigger[]
): CanvasNode {
  const triggerMetadata = triggers.find((t) => t.name === node.trigger?.name);
  const renderer = getTriggerRenderer(node.trigger?.name || "");

  return {
    id: node.id!,
    position: { x: node.position?.x!, y: node.position?.y! },
    data: {
      type: "trigger",
      label: node.name!,
      state: "pending" as const,
      trigger: renderer.getTriggerProps(node, triggerMetadata!),
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

function prepareNode(node: ComponentsNode, triggers: TriggersTrigger[]): CanvasNode {
  const nodeType = getNodeType(node);

  switch (nodeType) {
    case "trigger":
      return prepareTriggerNode(node, triggers);
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
