import { useCallback } from "react";
import { Edge } from "@xyflow/react";
import { useCanvasStore } from "../store/canvasStore";
import { AllNodeType } from "../types/flow";
import { autoLayoutNodes } from "../utils/flowTransformers";

export const useAutoLayout = () => {
  const updateNodePosition  = useCanvasStore((state) => state.updateNodePosition);
  const setNodes = useCanvasStore((state) => state.setNodes);

  const applyElkAutoLayout = useCallback(async (
    layoutedNodes: AllNodeType[],
    flowEdges: Edge[]
  ) => {
    if (layoutedNodes.length === 0) return;
    try {
      const newNodes = await autoLayoutNodes(layoutedNodes, flowEdges);

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