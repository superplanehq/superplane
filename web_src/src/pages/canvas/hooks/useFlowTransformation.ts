import { useCallback } from "react";
import { useFlowStore } from "../store/flowStore";
import { useCanvasStore } from "../store/canvasStore";
import { AllNodeType } from "../types/flow";
import { Edge } from "@xyflow/react";

export const useFlowTransformation = () => {
  const { setNodes, setEdges } = useFlowStore();
  const { updateNodePosition } = useCanvasStore();

  const updateNodesAndEdges = useCallback((
    layoutedNodes: AllNodeType[],
    flowEdges: Edge[]
  ) => {
    setEdges(flowEdges);
    
    const updatedNodes = layoutedNodes.map((node) => {
      const existingNode = document.querySelector(`[data-id="${node.id}"]`);
      if (existingNode) {
        return node;
      } else {
        return node;
      }
    });

    setNodes(updatedNodes);

    updatedNodes.forEach((node) => {
      if (node.position) {
        updateNodePosition(node.id, node.position);
      }
    });
  }, [setNodes, setEdges, updateNodePosition]);

  return { updateNodesAndEdges };
};