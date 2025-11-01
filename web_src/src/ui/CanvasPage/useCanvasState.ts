import type { Edge, EdgeChange, Node, NodeChange } from "@xyflow/react";
import { applyEdgeChanges, applyNodeChanges } from "@xyflow/react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AiProps, CanvasPageProps } from ".";
import { BreadcrumbItem } from "../../components/Breadcrumbs";

export interface CanvasPageState {
  title: string;
  breadcrumbs: BreadcrumbItem[];

  ai: AiProps;
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
    selectedNodeId: string | null;
    close: () => void;
    open: (nodeId: string) => void;
  };

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
}

export function useCanvasState(props: CanvasPageProps) : CanvasPageState {
  const { nodes: initialNodes, edges: initialEdges, startCollapsed } = props;

  const [nodes, setNodes] = useState<Node[]>(() => initialNodes ?? []);
  const [edges, setEdges] = useState<Edge[]>(() => initialEdges ?? []);
  const [isCollapsed, setIsCollapsed] = useState<boolean>(startCollapsed ?? false);

  // Apply initial collapsed state to nodes
  useEffect(() => {
    if (startCollapsed !== undefined && initialNodes) {
      setNodes((nds) =>
        nds.map((node) => {
          const nodeData = { ...node.data };
          const nodeType = nodeData.type as string;

          if (nodeType && nodeData[nodeType]) {
            nodeData[nodeType] = {
              ...nodeData[nodeType],
              collapsed: startCollapsed,
            };
          }

          return { ...node, data: nodeData };
        })
      );
    }
  }, [startCollapsed, initialNodes]);

  const onNodesChange = useCallback((changes: NodeChange[]) => {
    // Check for position changes and notify parent
    changes.forEach((change) => {
      if (change.type === 'position' && change.position && change.dragging === false && props.onNodePositionChange) {
        // Only notify when dragging ends (dragging === false)
        props.onNodePositionChange(change.id, change.position);
      }
    });

    setNodes((nds) => applyNodeChanges(changes, nds));
  }, [props]);

  const onEdgesChange = useCallback((changes: EdgeChange[]) => {
    // Check for edge removals and notify parent
    const removedEdgeIds = changes
      .filter((change) => change.type === 'remove')
      .map((change) => (change as any).id);

    if (removedEdgeIds.length > 0 && props.onEdgeDelete) {
      props.onEdgeDelete(removedEdgeIds);
    }

    setEdges((eds) => applyEdgeChanges(changes, eds));
  }, [props]);

  const toggleCollapse = useCallback(() => {
    setIsCollapsed((prev) => {
      const newCollapsed = !prev;
      setNodes((nds) =>
        nds.map((node) => {
          const nodeData = { ...node.data };
          const nodeType = nodeData.type as string;

          if (nodeType && nodeData[nodeType]) {
            nodeData[nodeType] = {
              ...nodeData[nodeType],
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

        const nodeData = { ...node.data };
        const nodeType = nodeData.type as string;

        if (nodeType && nodeData[nodeType]) {
          nodeData[nodeType] = {
            ...nodeData[nodeType],
            collapsed: !nodeData[nodeType].collapsed,
          };
        }

        return { ...node, data: nodeData };
      })
    );
  }, []);

  const componentSidebar = useComponentSidebarState();

  // Memoize the default ai object to prevent unnecessary re-renders
  const defaultAi = useMemo<AiProps>(() => ({
    enabled: false,
    sidebarOpen: false,
    setSidebarOpen: () => {},
    showNotifications: false,
    notificationMessage: undefined,
    suggestions: {},
    onApply: () => {},
    onDismiss: () => {},
  }), []);

  return {
    title: props.title || "Untitled Workflow",
    breadcrumbs: props.breadcrumbs || [
      { label: "Workflows" },
      { label: props.title || "Untitled Workflow" },
    ],
    nodes,
    componentSidebar,
    ai: props.ai || defaultAi,
    edges,
    setNodes,
    setEdges,
    onNodesChange,
    onEdgesChange,
    onNodeExpand: props.onNodeExpand,
    isCollapsed,
    toggleCollapse,
    toggleNodeCollapse,
  };
}

function useComponentSidebarState() : CanvasPageState["componentSidebar"] {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const close = useCallback(() => {
    setIsOpen(false);
    setSelectedNodeId(null);
  }, []);

  const open = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);
    setIsOpen(true);
  }, []);

  // Don't memoize the object itself - let it be a new reference each render
  // But the callbacks (open, close) are stable thanks to useCallback
  return {
    isOpen,
    selectedNodeId,
    close,
    open
  };
}
