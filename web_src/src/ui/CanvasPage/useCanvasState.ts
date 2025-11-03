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

  const [loaded, setLoaded] = useState(false);
  const [nodes, setNodes] = useState<Node[]>(() => initialNodes ?? []);
  const [edges, setEdges] = useState<Edge[]>(() => initialEdges ?? []);
  const [isCollapsed, setIsCollapsed] = useState<boolean>(startCollapsed ?? false);
  const [collapsedNodeIds, setCollapsedNodeIds] = useState<string[]>([]);

  // Sync nodes from props, but preserve collapsed state from collapsedNodeIds
  useEffect(() => {
    if (!initialNodes) return;
    const newCollapsedNodeIds: string[] = [];

    setNodes(initialNodes.map((node) => {
      const nodeData = { ...node.data };
      const nodeType = nodeData.type as string;
      
      if (nodeType && nodeData[nodeType]) {

        const isCollapsed = loaded ? collapsedNodeIds.includes(node.id) : (nodeData[nodeType] as {collapsed: boolean}).collapsed;
        nodeData[nodeType] = {
          ...nodeData[nodeType],
          collapsed: isCollapsed,
        };

        if (!loaded && isCollapsed) {
          newCollapsedNodeIds.push(node.id);
        }
      }

      return { ...node, data: nodeData };
    }));

    if (!loaded) {
      setCollapsedNodeIds(newCollapsedNodeIds);
      setLoaded(true);
    }
  }, [collapsedNodeIds, loaded]); // Only depend on collapsedNodeIds, not initialNodes

  // Sync node data changes from parent (but not collapsed state)
  useEffect(() => {
    if (!initialNodes) return;

    setNodes(currentNodes => {
      return initialNodes.map((newNode) => {
        const existingNode = currentNodes.find(n => n.id === newNode.id);
        const nodeData = { ...newNode.data };
        const nodeType = nodeData.type as string;

        // Preserve collapsed state from existing node
        if (existingNode && nodeType && nodeData[nodeType]) {
          const existingType = existingNode.data.type as string;
          const existingCollapsed = existingType && (existingNode.data[existingType] as any)?.collapsed;

          nodeData[nodeType] = {
            ...nodeData[nodeType],
            collapsed: existingCollapsed,
          };
        }

        return { ...newNode, data: nodeData };
      });
    });
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
      if (newCollapsed) {
        setCollapsedNodeIds(nodes.map((node) => node.id));
      } else {
        setCollapsedNodeIds([]);
      }
      return newCollapsed;
    });
    
  }, [nodes]);

  const toggleNodeCollapse = useCallback((nodeId: string) => {
    setCollapsedNodeIds((prev) => {
      const isCurrentlyCollapsed = prev.includes(nodeId);

      return isCurrentlyCollapsed
        ? prev.filter((id) => id !== nodeId)
        : [...prev, nodeId];
    });
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
