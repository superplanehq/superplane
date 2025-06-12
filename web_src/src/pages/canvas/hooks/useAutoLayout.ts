import { useCallback } from "react";
import { Edge } from "@xyflow/react";
import { ElkNode, ElkExtendedEdge } from "elkjs";
import { elk } from "../utils/layoutConfig";
import { DEFAULT_WIDTH, DEFAULT_HEIGHT } from "../utils/constants";
import { useFlowStore } from "../store/flowStore";
import { useCanvasStore } from "../store/canvasStore";
import { AllNodeType } from "../types/flow";

export const useAutoLayout = () => {
  const { setNodes } = useFlowStore();
  const { updateNodePosition } = useCanvasStore();

  const applyElkAutoLayout = useCallback(async (
    layoutedNodes: AllNodeType[],
    flowEdges: Edge[]
  ) => {
    if (layoutedNodes.length === 0) return;

    const elkNodes: ElkNode[] = layoutedNodes.map((node) => ({
      id: node.id,
      width: DEFAULT_WIDTH,
      height: DEFAULT_HEIGHT,
    }));

    const elkEdges: ElkExtendedEdge[] = flowEdges.map((edge) => ({
      id: edge.id,
      sources: [edge.source],
      targets: [edge.target],
    }));

    try {
      const layoutedGraph = await elk.layout({
        id: "root",
        children: elkNodes,
        edges: elkEdges,
      });

      const newNodes = layoutedNodes.map((node) => {
        const elkNode = layoutedGraph.children?.find((n) => n.id === node.id);
        const nodeElement: HTMLDivElement | null = document.querySelector(`[data-id="${node.id}"]`);

        if (elkNode?.x !== undefined && elkNode?.y !== undefined) {
          const newPosition = {
            x: elkNode.x + Math.random() / 1000,
            y: elkNode.y - (nodeElement?.offsetHeight || 0) / 2,
          };

          return {
            ...node,
            position: newPosition,
          };
        }

        return node;
      });

      setNodes(newNodes);

      newNodes.forEach((node) => {
        updateNodePosition(node.id, node.position);
      });
    } catch (error) {
      console.error('ELK auto-layout failed:', error);
    }
  }, [setNodes, updateNodePosition]);

  return { applyElkAutoLayout };
};