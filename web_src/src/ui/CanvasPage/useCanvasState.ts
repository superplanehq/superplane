import type { Edge, EdgeChange, Node, NodeChange } from "@xyflow/react";
import { applyEdgeChanges, applyNodeChanges } from "@xyflow/react";
import { useCallback, useEffect, useState } from "react";
import { CanvasPageProps, OnApproveFn, OnRejectFn } from ".";
import { BreadcrumbItem } from "../../components/Breadcrumbs";

export interface CanvasPageState {
  title: string;
  breadcrumbs: BreadcrumbItem[];

  nodes: Node[];
  edges: Edge[];

  setNodes: (nodes: Node[]) => void;
  setEdges: (edges: Edge[]) => void;

  onNodesChange: (changes: NodeChange[]) => void;
  onEdgesChange: (changes: EdgeChange[]) => void;

  isCollapsed: boolean;
  toggleCollapse: () => void;
  toggleNodeCollapse: (nodeId: string) => void;

  componentSidebar: {
    isOpen: boolean;
    close: () => void;
    open: () => void;
  };

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
  onApprove?: OnApproveFn;
  onReject?: OnRejectFn;
}

export function useCanvasState(props: CanvasPageProps) : CanvasPageState {
  const { nodes: initialNodes, edges: initialEdges, startCollapsed } = props;

  const [nodes, setNodes] = useState<Node[]>(() => initialNodes ?? []);
  const [edges, setEdges] = useState<Edge[]>(() => initialEdges ?? []);
  const [isCollapsed, setIsCollapsed] = useState<boolean>(startCollapsed ?? false);

  useEffect(() => {
    if (initialNodes) setNodes(initialNodes);
  }, [initialNodes]);

  useEffect(() => {
    if (initialEdges) setEdges(initialEdges);
  }, [initialEdges]);

  // Apply initial collapsed state to nodes
  useEffect(() => {
    if (startCollapsed !== undefined && initialNodes) {
      setNodes((nds) =>
        nds.map((node) => {
          const nodeData = { ...node.data };

          if (nodeData.type === "composite" && nodeData.composite) {
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: startCollapsed,
            };
          }

          if (nodeData.type === "approval" && nodeData.approval) {
            nodeData.approval = {
              ...nodeData.approval,
              collapsed: startCollapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: startCollapsed,
            };
          }

          return { ...node, data: nodeData };
        })
      );
    }
  }, [startCollapsed, initialNodes]);

  const onNodesChange = useCallback((changes: NodeChange[]) => {
    setNodes((nds) => applyNodeChanges(changes, nds));
  }, []);

  const onEdgesChange = useCallback((changes: EdgeChange[]) => {
    setEdges((eds) => applyEdgeChanges(changes, eds));
  }, []);

  const toggleCollapse = useCallback(() => {
    setIsCollapsed((prev) => {
      const newCollapsed = !prev;
      setNodes((nds) =>
        nds.map((node) => {
          const nodeData = { ...node.data };

          if (nodeData.type === "composite" && nodeData.composite) {
            nodeData.composite = {
              ...nodeData.composite,
              collapsed: newCollapsed,
            };
          }

          if (nodeData.type === "approval" && nodeData.approval) {
            nodeData.approval = {
              ...nodeData.approval,
              collapsed: newCollapsed,
            };
          }

          if (nodeData.type === "trigger" && nodeData.trigger) {
            nodeData.trigger = {
              ...nodeData.trigger,
              collapsed: newCollapsed,
            };
          }

          return { ...node, data: nodeData };
        })
      );
      return newCollapsed;
    });
  }, []);

  const toggleNodeCollapse = useCallback((nodeId: string) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id !== nodeId) return node;

        const nodeData = { ...node.data } as any;

        if (nodeData.type === "composite" && nodeData.composite) {
          nodeData.composite = {
            ...nodeData.composite,
            collapsed: !nodeData.composite.collapsed,
          };
        }

        if (nodeData.type === "approval" && nodeData.approval) {
          nodeData.approval = {
            ...nodeData.approval,
            collapsed: !nodeData.approval.collapsed,
          };
        }

        if (nodeData.type === "trigger" && nodeData.trigger) {
          nodeData.trigger = {
            ...nodeData.trigger,
            collapsed: !nodeData.trigger.collapsed,
          };
        }

        return { ...node, data: nodeData };
      })
    );
  }, []);

  const componentSidebar = useComponentSidebarState();

  return { 
    title: props.title || "Untitled Workflow", 
    breadcrumbs: props.breadcrumbs || [
      { label: "Workflows" },
      { label: props.title || "Untitled Workflow" },
    ],
    nodes, 
    componentSidebar,
    edges, 
    setNodes, 
    setEdges, 
    onNodesChange, 
    onEdgesChange, 
    onNodeExpand: props.onNodeExpand,
    isCollapsed, 
    toggleCollapse, 
    toggleNodeCollapse,
    onApprove: props.onApprove,
    onReject: props.onReject,
  };
}

function useComponentSidebarState() : CanvasPageState["componentSidebar"] {
  const [isOpen, setIsOpen] = useState<boolean>(false);

  const close = () => setIsOpen(false);
  const open = () => setIsOpen(true);

  return {
    isOpen,
    close,
    open
  };
}
