import { useCallback, useEffect, useState } from "react";
import type { Edge, EdgeChange, Node, NodeChange } from "reactflow";
import { applyEdgeChanges, applyNodeChanges } from "reactflow";

export function useCanvasState({ nodes: initialNodes, edges: initialEdges }: { nodes?: Node[]; edges?: Edge[] }) {
  const [nodes, setNodes] = useState<Node[]>(() => initialNodes ?? []);
  const [edges, setEdges] = useState<Edge[]>(() => initialEdges ?? []);

  useEffect(() => {
    if (initialNodes) setNodes(initialNodes);
  }, [initialNodes]);

  useEffect(() => {
    if (initialEdges) setEdges(initialEdges);
  }, [initialEdges]);

  const onNodesChange = useCallback((changes: NodeChange[]) => {
    setNodes((nds) => applyNodeChanges(changes, nds));
  }, []);

  const onEdgesChange = useCallback((changes: EdgeChange[]) => {
    setEdges((eds) => applyEdgeChanges(changes, eds));
  }, []);

  return { nodes, edges, setNodes, setEdges, onNodesChange, onEdgesChange } as const;
}
