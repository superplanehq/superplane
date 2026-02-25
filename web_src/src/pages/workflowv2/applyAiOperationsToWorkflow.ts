import type { CanvasesCanvas, ComponentsEdge, ComponentsNode } from "@/api-client";
import type { AiCanvasOperation, BuildingBlockCategory } from "@/ui/BuildingBlocksSidebar";
import { filterVisibleConfiguration } from "@/utils/components";
import { generateNodeId, generateUniqueNodeName } from "./utils";

type ApplyAiOperationsToWorkflowInput = {
  workflow: CanvasesCanvas;
  operations: AiCanvasOperation[];
  buildingBlocks: BuildingBlockCategory[];
};

export function applyAiOperationsToWorkflow({
  workflow,
  operations,
  buildingBlocks,
}: ApplyAiOperationsToWorkflowInput): CanvasesCanvas {
  const blockLookup = new Map(
    buildingBlocks.flatMap((category) => category.blocks.map((block) => [block.name, block])),
  );
  const createdNodeIdsByKey = new Map<string, string>();
  const minHorizontalGapDefault = 430;
  const minHorizontalGapNamed = 560;
  const minVerticalGap = 220;
  const estimatedNodeWidth = 420;
  const estimatedNodeHeight = 180;
  const nodePadding = 40;

  const updatedNodes: ComponentsNode[] = [...(workflow.spec?.nodes || [])];
  const updatedEdges: ComponentsEdge[] = [...(workflow.spec?.edges || [])];

  const resolveNodeId = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) return null;
    if (ref.nodeKey && createdNodeIdsByKey.has(ref.nodeKey)) {
      return createdNodeIdsByKey.get(ref.nodeKey) || null;
    }
    if (ref.nodeId) return ref.nodeId;
    if (ref.nodeName) {
      const found = updatedNodes.find((node) => node.id === ref.nodeName || node.name === ref.nodeName);
      return found?.id || null;
    }
    return null;
  };

  const overlapsExistingNode = (position: { x: number; y: number }) => {
    const bounds = {
      minX: position.x - nodePadding,
      minY: position.y - nodePadding,
      maxX: position.x + estimatedNodeWidth + nodePadding,
      maxY: position.y + estimatedNodeHeight + nodePadding,
    };

    return updatedNodes.some((node) => {
      if (!node.position) {
        return false;
      }

      const nodeBounds = {
        minX: node.position.x || 0,
        minY: node.position.y || 0,
        maxX: (node.position.x || 0) + estimatedNodeWidth,
        maxY: (node.position.y || 0) + estimatedNodeHeight,
      };

      return !(
        bounds.maxX < nodeBounds.minX ||
        bounds.minX > nodeBounds.maxX ||
        bounds.maxY < nodeBounds.minY ||
        bounds.minY > nodeBounds.maxY
      );
    });
  };

  const findAvailablePosition = (initialPosition: { x: number; y: number }) => {
    if (!overlapsExistingNode(initialPosition)) {
      return initialPosition;
    }

    const horizontalStep = 460;
    const verticalStep = 240;
    const maxSearchRadius = 8;

    for (let radius = 1; radius <= maxSearchRadius; radius += 1) {
      for (let dx = -radius; dx <= radius; dx += 1) {
        for (let dy = -radius; dy <= radius; dy += 1) {
          if (Math.abs(dx) !== radius && Math.abs(dy) !== radius) continue;
          const candidate = {
            x: initialPosition.x + dx * horizontalStep,
            y: initialPosition.y + dy * verticalStep,
          };
          if (!overlapsExistingNode(candidate)) {
            return candidate;
          }
        }
      }
    }

    return {
      x: initialPosition.x + horizontalStep,
      y: initialPosition.y + verticalStep,
    };
  };

  for (const operation of operations) {
    if (operation.type === "add_node") {
      const block = blockLookup.get(operation.blockName);
      if (!block) {
        continue;
      }

      const filteredConfiguration = filterVisibleConfiguration(
        operation.configuration || {},
        block.configuration || [],
      );

      const existingNodeNames = updatedNodes.map((node) => node.name || "").filter(Boolean);
      const uniqueNodeName = generateUniqueNodeName(operation.nodeName || block.name || "node", existingNodeNames);
      const newNodeId = generateNodeId(block.name || "node", uniqueNodeName);

      const initialPosition = (() => {
        if (operation.position) {
          return {
            x: Math.round(operation.position.x),
            y: Math.round(operation.position.y),
          };
        }

        const sourceNodeId = resolveNodeId(operation.source);
        const sourceNode = sourceNodeId ? updatedNodes.find((node) => node.id === sourceNodeId) : undefined;
        if (sourceNode?.position) {
          return {
            x: Math.round((sourceNode.position.x || 0) + 460),
            y: Math.round(sourceNode.position.y || 100),
          };
        }

        return {
          x: (updatedNodes.length || 0) * 460,
          y: 100,
        };
      })();

      const nonOverlappingPosition = findAvailablePosition(initialPosition);

      const newNode: ComponentsNode = {
        id: newNodeId,
        name: uniqueNodeName,
        type:
          block.type === "trigger"
            ? "TYPE_TRIGGER"
            : block.type === "blueprint"
              ? "TYPE_BLUEPRINT"
              : block.name === "annotation"
                ? "TYPE_WIDGET"
                : "TYPE_COMPONENT",
        configuration: filteredConfiguration,
        position: nonOverlappingPosition,
      };

      if (block.name === "annotation") {
        newNode.widget = { name: "annotation" };
        newNode.configuration = { text: "", color: "yellow" };
      } else if (block.type === "component") {
        newNode.component = { name: block.name };
      } else if (block.type === "trigger") {
        newNode.trigger = { name: block.name };
      } else if (block.type === "blueprint") {
        newNode.blueprint = { id: block.id };
      }

      updatedNodes.push(newNode);
      if (operation.nodeKey) {
        createdNodeIdsByKey.set(operation.nodeKey, newNodeId);
      }

      const sourceNodeId = resolveNodeId(operation.source);
      if (sourceNodeId) {
        updatedEdges.push({
          sourceId: sourceNodeId,
          targetId: newNodeId,
          channel: operation.source?.handleId || "default",
        });
      }
      continue;
    }

    if (operation.type === "connect_nodes") {
      const sourceId = resolveNodeId(operation.source);
      const targetId = resolveNodeId(operation.target);
      if (!sourceId || !targetId) {
        continue;
      }
      const channel = operation.source.handleId || "default";

      const sourceIndex = updatedNodes.findIndex((node) => node.id === sourceId);
      const targetIndex = updatedNodes.findIndex((node) => node.id === targetId);
      if (sourceIndex !== -1 && targetIndex !== -1) {
        const minHorizontalGap = channel === "default" ? minHorizontalGapDefault : minHorizontalGapNamed;
        const sourcePos = updatedNodes[sourceIndex].position;
        const targetPos = updatedNodes[targetIndex].position;
        if (sourcePos && targetPos) {
          const sourceX = sourcePos.x ?? 0;
          const sourceY = sourcePos.y ?? 100;
          const targetX = targetPos.x ?? sourceX + minHorizontalGap;
          const targetY = targetPos.y ?? sourceY;

          const nextX = targetX < sourceX + minHorizontalGap ? sourceX + minHorizontalGap : targetX;

          let nextY = targetY;
          const isNearlySameLane = Math.abs(targetY - sourceY) < 80;
          const hasMultipleEdgesFromSource = updatedEdges.some((edge) => edge.sourceId === sourceId);
          if (hasMultipleEdgesFromSource && isNearlySameLane) {
            nextY = sourceY + minVerticalGap;
          }

          if (nextX !== targetX || nextY !== targetY) {
            updatedNodes[targetIndex] = {
              ...updatedNodes[targetIndex],
              position: {
                x: Math.round(nextX),
                y: Math.round(nextY),
              },
            };
          }
        }
      }

      const edgeExists = updatedEdges.some(
        (edge) => edge.sourceId === sourceId && edge.targetId === targetId && edge.channel === channel,
      );
      if (!edgeExists) {
        updatedEdges.push({
          sourceId,
          targetId,
          channel,
        });
      }
      continue;
    }

    if (operation.type === "update_node_config") {
      const targetId = resolveNodeId(operation.target);
      if (!targetId) {
        continue;
      }

      const nodeIndex = updatedNodes.findIndex((node) => node.id === targetId);
      if (nodeIndex === -1) {
        continue;
      }

      const targetNode = updatedNodes[nodeIndex];
      updatedNodes[nodeIndex] = {
        ...targetNode,
        name: operation.nodeName || targetNode.name,
        configuration: {
          ...(targetNode.configuration || {}),
          ...(operation.configuration || {}),
        },
      };
      continue;
    }

    if (operation.type === "delete_node") {
      const targetId = resolveNodeId(operation.target);
      if (!targetId) {
        continue;
      }

      const nodeIndex = updatedNodes.findIndex((node) => node.id === targetId);
      if (nodeIndex === -1) {
        continue;
      }

      updatedNodes.splice(nodeIndex, 1);

      for (let edgeIndex = updatedEdges.length - 1; edgeIndex >= 0; edgeIndex -= 1) {
        const edge = updatedEdges[edgeIndex];
        if (edge.sourceId === targetId || edge.targetId === targetId) {
          updatedEdges.splice(edgeIndex, 1);
        }
      }
    }
  }

  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes: updatedNodes,
      edges: updatedEdges,
    },
  };
}
