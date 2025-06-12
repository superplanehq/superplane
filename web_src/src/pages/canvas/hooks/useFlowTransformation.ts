import { useCallback } from "react";
import { useCanvasStore } from "../store/canvasStore";
import { AllNodeType } from "../types/flow";

export const useFlowTransformation = () => {
  const { updateNodePosition, setNodes } = useCanvasStore();

  const updateNodesAndEdges = useCallback((
    layoutedNodes: AllNodeType[],
  ) => {
    
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
  }, [setNodes, updateNodePosition]);

  return { updateNodesAndEdges };
};