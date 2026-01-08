import type { Edge, EdgeChange, Node, NodeChange } from "@xyflow/react";
import { applyEdgeChanges, applyNodeChanges } from "@xyflow/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AiProps, CanvasPageProps } from ".";
import { BreadcrumbItem } from "../../components/Breadcrumbs";

export interface CanvasPageState {
  title: string;
  breadcrumbs: BreadcrumbItem[];

  ai: AiProps;
  nodes: Node[];
  edges: Edge[];

  setNodes: (nodes: Node[] | ((prevNodes: Node[]) => Node[])) => void;
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

export function useCanvasState(props: CanvasPageProps): CanvasPageState {
  const { nodes: initialNodes, edges: initialEdges, startCollapsed } = props;

  const loadedFirstCollapsedNodeIds = useRef(false);
  const [nodes, setNodes] = useState<Node[]>(() => initialNodes ?? []);
  const [edges, setEdges] = useState<Edge[]>(() => initialEdges ?? []);
  const isCollapsed = useMemo<boolean>(() => {
    if (startCollapsed !== undefined) {
      return startCollapsed;
    }

    const isMajorityCollapsed =
      nodes.filter((node) => {
        const nodeType = node.data.type as string;
        const componentData = node.data[nodeType] as { collapsed: boolean };
        return componentData.collapsed;
      }).length >
      nodes.length / 2;
    return isMajorityCollapsed;
  }, [startCollapsed, nodes]);
  const [, setCollapsedNodeIds] = useState<string[]>([]);

  // Sync node data changes from parent (but not collapsed state or selected state)
  useEffect(() => {
    if (!initialNodes) return;

    const newCollapsedNodeIds: string[] = [];
    setNodes((currentNodes) => {
      // Preserve locally-added template and pending connection nodes
      const localOnlyNodes = currentNodes.filter((node) => node.data.isTemplate || node.data.isPendingConnection);

      const syncedNodes = initialNodes.map((newNode) => {
        const existingNode = currentNodes.find((n) => n.id === newNode.id);
        const nodeData = { ...newNode.data };
        const nodeType = nodeData.type as string;

        // Preserve collapsed state from existing node
        if (existingNode && nodeType && nodeData[nodeType]) {
          const existingType = existingNode.data.type as string;
          const existingCollapsed =
            existingType && (existingNode.data[existingType] as { collapsed: boolean })?.collapsed;
          newCollapsedNodeIds.push(existingNode.id);
          nodeData[nodeType] = {
            ...nodeData[nodeType],
            collapsed: existingCollapsed,
          };
        }

        // Preserve selected state from existing node
        return {
          ...newNode,
          data: nodeData,
          selected: existingNode?.selected ?? newNode.selected,
        };
      });

      // Append local-only nodes at the end
      return [...syncedNodes, ...localOnlyNodes];
    });
  }, [initialNodes]);

  useEffect(() => {
    if (initialNodes.length === 0 || loadedFirstCollapsedNodeIds.current) return;

    setCollapsedNodeIds(
      initialNodes
        .filter((node) => {
          const nodeType = node.data.type as string;
          const componentData = node.data[nodeType] as { collapsed: boolean };
          return componentData.collapsed;
        })
        .map((node) => node.id),
    );

    loadedFirstCollapsedNodeIds.current = true;
  }, [initialNodes]);

  useEffect(() => {
    if (!initialEdges) return;

    setEdges((currentEdges) => {
      // Preserve edges connected to template or pending connection nodes
      const localOnlyEdges = currentEdges.filter((edge) => {
        const sourceIsLocal = nodes.some(
          (n) => n.id === edge.source && (n.data.isTemplate || n.data.isPendingConnection),
        );
        const targetIsLocal = nodes.some(
          (n) => n.id === edge.target && (n.data.isTemplate || n.data.isPendingConnection),
        );
        return sourceIsLocal || targetIsLocal;
      });

      // Combine synced edges with local-only edges
      return [...initialEdges, ...localOnlyEdges];
    });
  }, [initialEdges, nodes]);

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
        }),
      );
    }
  }, [startCollapsed, initialNodes]);

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => {
      // Propagate node removals (e.g., via Backspace/Delete) to parent
      const removedNodeIds = changes.filter((change) => change.type === "remove").map((change) => change.id);

      if (removedNodeIds.length > 0 && props.onNodeDelete) {
        removedNodeIds.forEach((id) => props.onNodeDelete?.(id));
      }

      // Check for position changes and notify parent
      changes.forEach((change) => {
        if (change.type === "position" && change.position && change.dragging === false && props.onNodePositionChange) {
          // Only notify when dragging ends (dragging === false)
          props.onNodePositionChange(change.id, change.position);
        }
      });

      setNodes((nds) => applyNodeChanges(changes, nds));
    },
    [props],
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      // Check for edge removals and notify parent
      const removedEdgeIds = changes.filter((change) => change.type === "remove").map((change) => change.id);

      if (removedEdgeIds.length > 0 && props.onEdgeDelete) {
        props.onEdgeDelete(removedEdgeIds);
      }

      setEdges((eds) => applyEdgeChanges(changes, eds));
    },
    [props],
  );

  const toggleCollapse = useCallback(() => {
    const newCollapsed = !isCollapsed;
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
      }),
    );

    if (newCollapsed) {
      setCollapsedNodeIds(nodes.map((node) => node.id));
    } else {
      setCollapsedNodeIds([]);
    }
    return newCollapsed;
  }, [nodes, setNodes, isCollapsed]);

  const toggleNodeCollapse = useCallback(
    (nodeId: string) => {
      let isCurrentlyCollapsed = false;
      setCollapsedNodeIds((prev) => {
        isCurrentlyCollapsed = prev.includes(nodeId);
        return isCurrentlyCollapsed ? prev.filter((id) => id !== nodeId) : [...prev, nodeId];
      });

      setNodes((nds) =>
        nds.map((node) => {
          const nodeData = { ...node.data };
          const nodeType = nodeData.type as string;
          const componentData = nodeData[nodeType] as { collapsed: boolean };

          if (nodeType && nodeData[nodeType]) {
            nodeData[nodeType] = {
              ...nodeData[nodeType],
              collapsed: nodeId === node.id ? !isCurrentlyCollapsed : (componentData.collapsed as boolean),
            };
          }

          return { ...node, data: nodeData };
        }),
      );
    },
    [setNodes],
  );

  const componentSidebar = useComponentSidebarState(props.initialSidebar, props.onSidebarChange);

  // Memoize the default ai object to prevent unnecessary re-renders
  const defaultAi = useMemo<AiProps>(
    () => ({
      enabled: false,
      sidebarOpen: false,
      setSidebarOpen: () => {},
      showNotifications: false,
      notificationMessage: undefined,
      suggestions: {},
      onApply: () => {},
      onDismiss: () => {},
    }),
    [],
  );

  return {
    title: props.title || "Untitled Workflow",
    breadcrumbs: props.breadcrumbs || [{ label: "Workflows" }, { label: props.title || "Untitled Workflow" }],
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

function useComponentSidebarState(
  initial: { isOpen?: boolean; nodeId?: string | null } | undefined,
  onChange?: (isOpen: boolean, selectedNodeId: string | null) => void,
): CanvasPageState["componentSidebar"] {
  const [isOpen, setIsOpen] = useState<boolean>(initial?.isOpen ?? false);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(initial?.nodeId ?? null);
  const lastInitialRef = useRef<{ isOpen: boolean; nodeId: string | null } | null>(null);

  const close = useCallback(() => {
    setIsOpen(false);
    setSelectedNodeId(null);
    onChange?.(false, null);
  }, [onChange]);

  const open = useCallback(
    (nodeId: string) => {
      setSelectedNodeId(nodeId);
      setIsOpen(true);
      onChange?.(true, nodeId);
    },
    [onChange],
  );

  // Keep external listener updated when selection changes while open
  useEffect(() => {
    if (isOpen) {
      onChange?.(true, selectedNodeId);
    }
  }, [isOpen, selectedNodeId, onChange]);

  useEffect(() => {
    if (initial?.isOpen === undefined && initial?.nodeId === undefined) {
      return;
    }

    const nextIsOpen = initial?.isOpen ?? false;
    const nextNodeId = initial?.nodeId ?? null;
    const lastInitial = lastInitialRef.current;

    if (lastInitial && lastInitial.isOpen === nextIsOpen && lastInitial.nodeId === nextNodeId) {
      return;
    }

    lastInitialRef.current = { isOpen: nextIsOpen, nodeId: nextNodeId };
    setIsOpen(nextIsOpen);
    setSelectedNodeId(nextNodeId);
  }, [initial?.isOpen, initial?.nodeId]);

  // Don't memoize the object itself - let it be a new reference each render
  // But the callbacks (open, close) are stable thanks to useCallback
  return {
    isOpen,
    selectedNodeId,
    close,
    open,
  };
}
