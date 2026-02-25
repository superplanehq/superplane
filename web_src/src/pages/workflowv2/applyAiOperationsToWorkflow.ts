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
  const defaultHorizontalStep = 460;
  const defaultLaneY = 100;
  const disconnectedFlowVerticalGap = 320;
  const estimatedNodeWidth = 420;
  const estimatedNodeHeight = 180;
  const nodePadding = 40;

  const existingNodes = [...(workflow.spec?.nodes || [])];
  const updatedNodes: ComponentsNode[] = [...existingNodes];
  const updatedEdges: ComponentsEdge[] = [...(workflow.spec?.edges || [])];
  const existingNodeIds = new Set(existingNodes.map((node) => node.id));

  const resolveExistingNodeId = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) return null;
    if (ref.nodeId && existingNodeIds.has(ref.nodeId)) {
      return ref.nodeId;
    }
    if (ref.nodeName) {
      const found = existingNodes.find((node) => node.id === ref.nodeName || node.name === ref.nodeName);
      return found?.id || null;
    }
    return null;
  };

  const addedNodeKeys = new Set(
    operations
      .filter(
        (operation): operation is Extract<AiCanvasOperation, { type: "add_node" }> => operation.type === "add_node",
      )
      .map((operation) => operation.nodeKey)
      .filter((nodeKey): nodeKey is string => !!nodeKey),
  );
  const existingGraphNode = "__existing__";
  const graphEdges = new Map<string, Set<string>>();
  const addGraphEdge = (left: string, right: string) => {
    if (!graphEdges.has(left)) graphEdges.set(left, new Set());
    if (!graphEdges.has(right)) graphEdges.set(right, new Set());
    graphEdges.get(left)?.add(right);
    graphEdges.get(right)?.add(left);
  };
  const toGraphNode = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) return null;
    if (ref.nodeKey && addedNodeKeys.has(ref.nodeKey)) {
      return `key:${ref.nodeKey}`;
    }
    if (resolveExistingNodeId(ref)) {
      return existingGraphNode;
    }
    return null;
  };

  for (const nodeKey of addedNodeKeys) {
    graphEdges.set(`key:${nodeKey}`, new Set());
  }
  for (const operation of operations) {
    if (operation.type !== "connect_nodes") continue;
    const sourceGraphNode = toGraphNode(operation.source);
    const targetGraphNode = toGraphNode(operation.target);
    if (!sourceGraphNode || !targetGraphNode || sourceGraphNode === targetGraphNode) {
      continue;
    }
    addGraphEdge(sourceGraphNode, targetGraphNode);
  }

  const graphNodesConnectedToExisting = new Set<string>();
  if (graphEdges.has(existingGraphNode)) {
    const queue = [existingGraphNode];
    graphNodesConnectedToExisting.add(existingGraphNode);
    while (queue.length > 0) {
      const current = queue.shift();
      if (!current) continue;
      const neighbors = graphEdges.get(current);
      if (!neighbors) continue;
      for (const neighbor of neighbors) {
        if (graphNodesConnectedToExisting.has(neighbor)) continue;
        graphNodesConnectedToExisting.add(neighbor);
        queue.push(neighbor);
      }
    }
  }

  const detachedComponentByNodeKey = new Map<string, number>();
  const detachedVisited = new Set<string>();
  let detachedComponentIndex = 0;
  for (const nodeKey of addedNodeKeys) {
    const graphNode = `key:${nodeKey}`;
    if (graphNodesConnectedToExisting.has(graphNode) || detachedVisited.has(graphNode)) {
      continue;
    }
    const queue = [graphNode];
    while (queue.length > 0) {
      const current = queue.shift();
      if (!current || detachedVisited.has(current)) continue;
      detachedVisited.add(current);
      if (current.startsWith("key:")) {
        detachedComponentByNodeKey.set(current.slice(4), detachedComponentIndex);
      }
      const neighbors = graphEdges.get(current);
      if (!neighbors) continue;
      for (const neighbor of neighbors) {
        if (neighbor !== existingGraphNode && !detachedVisited.has(neighbor)) {
          queue.push(neighbor);
        }
      }
    }
    detachedComponentIndex += 1;
  }

  const existingNodeXs = existingNodes
    .map((node) => node.position?.x)
    .filter((x): x is number => typeof x === "number");
  const startX =
    existingNodeXs.length > 0 ? existingNodeXs.reduce((minX, x) => (x < minX ? x : minX), Number.POSITIVE_INFINITY) : 0;
  const maxYFromExistingNodes = existingNodes
    .map((node) => node.position?.y)
    .filter((y): y is number => typeof y === "number")
    .reduce((maxY, y) => (y > maxY ? y : maxY), Number.NEGATIVE_INFINITY);
  const detachedFlowStartY = Number.isFinite(maxYFromExistingNodes)
    ? maxYFromExistingNodes + disconnectedFlowVerticalGap
    : defaultLaneY;
  const detachedFlowYByComponent = new Map<number, number>();
  const detachedFlowNextXByComponent = new Map<number, number>();
  let nextDetachedFlowLane = 0;

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

      // Treat touching edges as non-overlap so linear chains can sit on the same lane.
      return !(
        bounds.maxX <= nodeBounds.minX ||
        bounds.minX >= nodeBounds.maxX ||
        bounds.maxY <= nodeBounds.minY ||
        bounds.minY >= nodeBounds.maxY
      );
    });
  };

  const findAvailablePosition = (initialPosition: { x: number; y: number }) => {
    if (!overlapsExistingNode(initialPosition)) {
      return initialPosition;
    }

    const horizontalStep = defaultHorizontalStep;
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
            x: Math.round((sourceNode.position.x || 0) + defaultHorizontalStep),
            y: Math.round(sourceNode.position.y || defaultLaneY),
          };
        }

        if (operation.nodeKey) {
          const graphNode = `key:${operation.nodeKey}`;
          const isConnectedToExisting = graphNodesConnectedToExisting.has(graphNode);
          const detachedComponent = detachedComponentByNodeKey.get(operation.nodeKey);
          if (!isConnectedToExisting && detachedComponent !== undefined) {
            if (!detachedFlowYByComponent.has(detachedComponent)) {
              detachedFlowYByComponent.set(
                detachedComponent,
                detachedFlowStartY + nextDetachedFlowLane * disconnectedFlowVerticalGap,
              );
              detachedFlowNextXByComponent.set(detachedComponent, startX);
              nextDetachedFlowLane += 1;
            }

            const laneY = detachedFlowYByComponent.get(detachedComponent) || detachedFlowStartY;
            const laneX = detachedFlowNextXByComponent.get(detachedComponent) || startX;
            detachedFlowNextXByComponent.set(detachedComponent, laneX + defaultHorizontalStep);
            return {
              x: Math.round(laneX),
              y: Math.round(laneY),
            };
          }
        }

        const rightMostNode = updatedNodes
          .filter((node) => !!node.position)
          .reduce<ComponentsNode | null>((best, node) => {
            if (!node.position) return best;
            if (!best?.position) return node;
            const bestX = best.position.x ?? 0;
            const nodeX = node.position.x ?? 0;
            return nodeX > bestX ? node : best;
          }, null);
        if (rightMostNode?.position) {
          return {
            x: Math.round((rightMostNode.position.x || 0) + defaultHorizontalStep),
            y: Math.round(rightMostNode.position.y || defaultLaneY),
          };
        }

        return {
          x: (updatedNodes.length || 0) * defaultHorizontalStep,
          y: defaultLaneY,
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

          let nextY = channel === "default" ? sourceY : targetY;
          const isNearlySameLane = Math.abs(targetY - sourceY) < 80;
          const existingOutgoingEdges = updatedEdges.filter((edge) => edge.sourceId === sourceId);
          const createsDistinctBranch = existingOutgoingEdges.some(
            (edge) => edge.targetId !== targetId || edge.channel !== channel,
          );
          if (createsDistinctBranch && isNearlySameLane) {
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

  // Normalize disconnected components relative to existing canvas content:
  // when new flows are disconnected, place them on new lanes below existing flows
  // and align their start X with the existing flow start.
  if (existingNodeIds.size > 0) {
    const nodeIds = updatedNodes
      .map((node) => node.id)
      .filter((nodeId): nodeId is string => typeof nodeId === "string" && nodeId.length > 0);
    const adjacency = new Map<string, Set<string>>();
    for (const nodeId of nodeIds) {
      adjacency.set(nodeId, new Set());
    }
    for (const edge of updatedEdges) {
      const sourceId = edge.sourceId;
      const targetId = edge.targetId;
      if (!sourceId || !targetId || !adjacency.has(sourceId) || !adjacency.has(targetId)) {
        continue;
      }
      adjacency.get(sourceId)?.add(targetId);
      adjacency.get(targetId)?.add(sourceId);
    }

    const visited = new Set<string>();
    const components: string[][] = [];
    for (const nodeId of nodeIds) {
      if (visited.has(nodeId)) continue;
      const queue = [nodeId];
      const component: string[] = [];
      while (queue.length > 0) {
        const current = queue.shift();
        if (!current || visited.has(current)) continue;
        visited.add(current);
        component.push(current);
        for (const neighbor of adjacency.get(current) || []) {
          if (!visited.has(neighbor)) {
            queue.push(neighbor);
          }
        }
      }
      components.push(component);
    }

    const idToNode = new Map<string, ComponentsNode>();
    for (const node of updatedNodes) {
      if (typeof node.id === "string" && node.id.length > 0) {
        idToNode.set(node.id, node);
      }
    }
    const componentHasExisting = (component: string[]) => component.some((nodeId) => existingNodeIds.has(nodeId));
    const positionedMetrics = (component: string[]) => {
      const positioned = component
        .map((id) => idToNode.get(id))
        .filter((node): node is ComponentsNode & { position: { x?: number; y?: number } } => !!node?.position);
      if (positioned.length === 0) {
        return null;
      }
      const minX = positioned.reduce(
        (value, node) => Math.min(value, node.position.x ?? value),
        Number.POSITIVE_INFINITY,
      );
      const minY = positioned.reduce(
        (value, node) => Math.min(value, node.position.y ?? value),
        Number.POSITIVE_INFINITY,
      );
      const maxY = positioned.reduce(
        (value, node) => Math.max(value, node.position.y ?? value),
        Number.NEGATIVE_INFINITY,
      );
      return { minX, minY, maxY };
    };

    const existingComponents = components.filter(componentHasExisting);
    const detachedComponents = components.filter((component) => !componentHasExisting(component));
    if (existingComponents.length > 0 && detachedComponents.length > 0) {
      const existingMetrics = existingComponents
        .map(positionedMetrics)
        .filter((metric): metric is NonNullable<typeof metric> => !!metric);
      const detachedWithMetrics = detachedComponents
        .map((component) => ({ component, metrics: positionedMetrics(component) }))
        .filter(
          (entry): entry is { component: string[]; metrics: NonNullable<ReturnType<typeof positionedMetrics>> } =>
            !!entry.metrics,
        )
        .sort((left, right) => {
          if (left.metrics.minY !== right.metrics.minY) {
            return left.metrics.minY - right.metrics.minY;
          }
          return left.metrics.minX - right.metrics.minX;
        });

      if (existingMetrics.length > 0) {
        const existingStartX = existingMetrics.reduce(
          (value, metric) => Math.min(value, metric.minX),
          Number.POSITIVE_INFINITY,
        );
        const existingMaxY = existingMetrics.reduce(
          (value, metric) => Math.max(value, metric.maxY),
          Number.NEGATIVE_INFINITY,
        );
        let nextLaneY = existingMaxY + disconnectedFlowVerticalGap;

        for (const { component, metrics } of detachedWithMetrics) {
          const shiftX = existingStartX - metrics.minX;
          const shiftY = nextLaneY - metrics.minY;
          for (const nodeId of component) {
            const node = idToNode.get(nodeId);
            if (!node?.position) continue;
            node.position = {
              x: Math.round((node.position.x || 0) + shiftX),
              y: Math.round((node.position.y || 0) + shiftY),
            };
          }
          nextLaneY += disconnectedFlowVerticalGap;
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
