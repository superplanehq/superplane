import { useCallback, useRef } from "react";
import { Node, ReactFlowInstance, OnInit } from "@xyflow/react";
import { useCanvasStore } from "../store/canvasStore";
import { AllNodeType, EdgeType } from "../types/flow";

export const useFlowHandlers = () => {
  const { updateNodePosition } = useCanvasStore();
  const reactFlowInstanceRef = useRef<ReactFlowInstance<AllNodeType, EdgeType> | null>(null);

  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: Node) => {
      updateNodePosition(node.id, node.position);
    },
    [updateNodePosition]
  );

  const onInit: OnInit<AllNodeType, EdgeType> = useCallback((instance) => {
    reactFlowInstanceRef.current = instance;
    instance.fitView();
  }, []);

  const fitViewToNode = useCallback((nodeId: string) => {
    if (reactFlowInstanceRef.current) {
      const nodes = reactFlowInstanceRef.current.getNodes();
      const targetNode = nodes.find(node => node.id === nodeId);
      
      if (targetNode) {
        reactFlowInstanceRef.current.fitView({
          nodes: [targetNode],
          duration: 800,
          padding: 0.3
        });
      }
    }
  }, []);

  return {
    onNodeDragStop,
    onInit,
    reactFlowInstanceRef,
    fitViewToNode
  };
};