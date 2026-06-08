import type {
  CanvasesCanvas,
  SuperplaneActionsAction,
  SuperplaneComponentsEdge as ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { DefaultLayoutEngine } from "@/lib/layout";

import type { CanvasChangesetChange } from "./canvas-changeset-types";

export async function applyCanvasChangesetLocally(
  workflow: CanvasesCanvas,
  operations: CanvasChangesetChange[],
  components: SuperplaneActionsAction[],
): Promise<CanvasesCanvas> {
  let nodes = [...(workflow.spec?.nodes ?? [])];
  let edges = [...(workflow.spec?.edges ?? [])];

  for (const operation of operations) {
    switch (operation.type) {
      case "ADD_NODE":
        if (operation.node) {
          nodes.push(operation.node as ComponentsNode);
        }
        break;
      case "UPDATE_NODE":
        if (operation.node?.id) {
          nodes = nodes.map((node) => (node.id === operation.node?.id ? { ...node, ...operation.node } : node));
        }
        break;
      case "DELETE_NODE":
        if (operation.nodeId) {
          nodes = nodes.filter((node) => node.id !== operation.nodeId);
          edges = edges.filter((edge) => edge.sourceId !== operation.nodeId && edge.targetId !== operation.nodeId);
        }
        break;
      case "ADD_EDGE":
        if (operation.edge) {
          edges.push(operation.edge as ComponentsEdge);
        }
        break;
      case "DELETE_EDGE":
        if (operation.edgeId) {
          edges = edges.filter((edge, index) => {
            const edgeKey = `${edge.sourceId}:${edge.targetId}:${edge.channel ?? ""}`;
            return edgeKey !== operation.edgeId && String(index) !== operation.edgeId;
          });
        }
        break;
      default:
        break;
    }
  }

  let nextWorkflow: CanvasesCanvas = {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes,
      edges,
    },
  };

  const autoLayoutNodeIds = operations
    .filter((operation) => operation.type === "ADD_NODE")
    .map((operation) => operation.node?.id)
    .filter((id): id is string => Boolean(id));

  if (autoLayoutNodeIds.length > 0) {
    nextWorkflow = await DefaultLayoutEngine.apply(nextWorkflow, {
      scope: "connected-component",
      nodeIds: autoLayoutNodeIds,
      components,
    });
  }

  return nextWorkflow;
}
