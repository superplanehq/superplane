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

  return {
    onNodeDragStop,
    onInit,
    reactFlowInstanceRef
  };
};