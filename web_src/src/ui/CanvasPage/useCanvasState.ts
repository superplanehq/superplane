import type { Edge, EdgeChange, Node, NodeChange, NodePositionChange } from "@xyflow/react";
import { applyEdgeChanges, applyNodeChanges } from "@xyflow/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CanvasPageProps } from ".";

function areEdgeListsReferentiallyEqual(currentEdges: Edge[], nextEdges: Edge[]): boolean {
  if (currentEdges.length !== nextEdges.length) {
    return false;
  }

  return currentEdges.every((edge, index) => edge === nextEdges[index]);
}

function arePositionsEqual(
  left: { x: number; y: number } | undefined,
  right: { x: number; y: number } | undefined,
): boolean {
  return left?.x === right?.x && left?.y === right?.y;
}

function getRoundedPosition(position: { x: number; y: number }): { x: number; y: number } {
  return {
    x: Math.round(position.x),
    y: Math.round(position.y),
  };
}

type PendingNodePosition = {
  localPosition: { x: number; y: number };
  savedPosition: { x: number; y: number };
};

export interface CanvasPageState {
  nodes: Node[];
  edges: Edge[];

  setNodes: (nodes: Node[] | ((prevNodes: Node[]) => Node[])) => void;
  setEdges: (edges: Edge[]) => void;

  onNodesChange: (changes: NodeChange[]) => void;
  onEdgesChange: (changes: EdgeChange[]) => void;

  toggleNodeCollapse: (nodeId: string) => void;

  componentSidebar: {
    isOpen: boolean;
    selectedNodeId: string | null;
    close: () => void;
    open: (nodeId: string) => void;
    clearSelection: () => void;
  };
}

export function useCanvasState(props: CanvasPageProps): CanvasPageState {
  const { nodes: initialNodes, edges: initialEdges, startCollapsed } = props;

  const [nodes, setNodes] = useState<Node[]>(() => initialNodes ?? []);
  const [edges, setEdges] = useState<Edge[]>(() => initialEdges ?? []);
  const pendingNodePositionsRef = useRef<Map<string, PendingNodePosition>>(new Map());
  const localOnlyNodeIdsKey = useMemo(
    () =>
      nodes
        .filter((node) => node.data.isTemplate || node.data.isPendingConnection)
        .map((node) => node.id)
        .sort()
        .join("\0"),
    [nodes],
  );

  // Sync node data changes from parent (but not collapsed state or selected state)
  useEffect(() => {
    if (!initialNodes) return;

    setNodes((currentNodes) => {
      // Preserve locally-added template and pending connection nodes
      const localOnlyNodes = currentNodes.filter((node) => node.data.isTemplate || node.data.isPendingConnection);
      const syncedNodeIds = new Set<string>();

      const syncedNodes = initialNodes.map((newNode) => {
        syncedNodeIds.add(newNode.id);
        const existingNode = currentNodes.find((n) => n.id === newNode.id);
        const nodeData = { ...newNode.data };
        const nodeType = nodeData.type as string;

        // Preserve collapsed state from existing node
        if (existingNode && nodeType && nodeData[nodeType]) {
          const existingType = existingNode.data.type as string;
          const existingCollapsed =
            existingType && (existingNode.data[existingType] as { collapsed: boolean })?.collapsed;
          nodeData[nodeType] = {
            ...nodeData[nodeType],
            collapsed: existingCollapsed,
          };
        }

        const pendingPosition = pendingNodePositionsRef.current.get(newNode.id);
        let position = (existingNode?.dragging && existingNode.position) || newNode.position;
        if (pendingPosition) {
          if (arePositionsEqual(newNode.position, pendingPosition.savedPosition)) {
            pendingNodePositionsRef.current.delete(newNode.id);
          } else {
            position = pendingPosition.localPosition;
          }
        }

        // Preserve selected state and position of actively dragged nodes
        return {
          ...newNode,
          data: nodeData,
          selected: existingNode?.selected ?? newNode.selected,
          position,
          dragging: existingNode?.dragging,
        };
      });

      for (const nodeId of pendingNodePositionsRef.current.keys()) {
        if (!syncedNodeIds.has(nodeId)) {
          pendingNodePositionsRef.current.delete(nodeId);
        }
      }

      // Append local-only nodes at the end
      return [...syncedNodes, ...localOnlyNodes];
    });
  }, [initialNodes]);

  useEffect(() => {
    if (!initialEdges) return;

    const localOnlyNodeIds = new Set(localOnlyNodeIdsKey ? localOnlyNodeIdsKey.split("\0") : []);

    setEdges((currentEdges) => {
      // Preserve edges connected to template or pending connection nodes
      const localOnlyEdges = currentEdges.filter((edge) => {
        return localOnlyNodeIds.has(edge.source) || localOnlyNodeIds.has(edge.target);
      });

      // Combine synced edges with local-only edges
      const nextEdges = [...initialEdges, ...localOnlyEdges];
      if (areEdgeListsReferentiallyEqual(currentEdges, nextEdges)) {
        return currentEdges;
      }

      return nextEdges;
    });
  }, [initialEdges, localOnlyNodeIdsKey]);

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
      // Collect all position changes that ended (dragging === false)
      const positionChanges = changes.filter(
        (change): change is NodePositionChange & { position: { x: number; y: number } } =>
          change.type === "position" &&
          change.position !== undefined &&
          change.dragging === false &&
          typeof change.position.x === "number" &&
          typeof change.position.y === "number",
      );

      if (positionChanges.length > 0) {
        positionChanges.forEach((change) => {
          pendingNodePositionsRef.current.set(change.id, {
            localPosition: change.position,
            savedPosition: getRoundedPosition(change.position),
          });
        });

        // If batch update is supported, use it for multiple nodes
        if (positionChanges.length > 1 && props.onNodesPositionChange) {
          const updates = positionChanges.map((change) => ({
            nodeId: change.id,
            position: change.position,
          }));
          props.onNodesPositionChange(updates);
        } else if (props.onNodePositionChange) {
          // Fall back to individual updates for single node or when batch is not supported
          positionChanges.forEach((change) => {
            props.onNodePositionChange!(change.id, change.position);
          });
        }
      }

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

  const toggleNodeCollapse = useCallback(
    (nodeId: string) => {
      setNodes((nds) =>
        nds.map((node) => {
          if (node.id !== nodeId) {
            return node;
          }

          const nodeData = { ...node.data };
          const nodeType = nodeData.type as string;
          if (nodeType && nodeData[nodeType]) {
            const componentData = nodeData[nodeType] as { collapsed?: boolean };
            nodeData[nodeType] = {
              ...nodeData[nodeType],
              collapsed: !componentData.collapsed,
            };
          }

          return { ...node, data: nodeData };
        }),
      );
    },
    [setNodes],
  );

  const componentSidebar = useComponentSidebarState(props.initialSidebar, props.onSidebarChange);

  return {
    nodes,
    componentSidebar,
    edges,
    setNodes,
    setEdges,
    onNodesChange,
    onEdgesChange,
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

  // Keep the latest onChange in a ref so the callbacks and the sync effect below
  // don't depend on its identity. `onChange` typically wraps react-router's
  // `setSearchParams`, whose identity changes on every URL update — depending on
  // it would otherwise re-run the sync effect on unrelated URL changes and
  // re-push stale sidebar params back into the URL (e.g. after exiting edit
  // mode, or clobbering the `branch` param when entering it).
  const onChangeRef = useRef(onChange);
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  const close = useCallback(() => {
    setIsOpen(false);
    setSelectedNodeId(null);
    onChangeRef.current?.(false, null);
  }, []);

  const open = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);
    setIsOpen(true);
    onChangeRef.current?.(true, nodeId);
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedNodeId(null);
    onChangeRef.current?.(true, null);
  }, []);

  // Keep external listener updated when the selection genuinely changes while
  // open. `open`/`close` already notify `onChange` for user-driven changes, so
  // this effect only needs to cover internal selection changes that bypass them.
  //
  // It must NOT fire on mount or when echoing an externally-applied `initial`
  // value. On mount the URL already reflects the sidebar state, and the
  // component remounts whenever the selected version changes (its `key`
  // includes the version id). Re-emitting here on remount would navigate from a
  // stale location snapshot and clobber concurrent URL updates — e.g. switching
  // drafts while a node is open would revert the freshly-set `branch` param.
  const hasSyncedOnceRef = useRef(false);
  useEffect(() => {
    if (!hasSyncedOnceRef.current) {
      hasSyncedOnceRef.current = true;
      return;
    }

    if (!isOpen) {
      return;
    }

    // Don't echo a selection that originated from an external (URL) sync.
    const lastInitial = lastInitialRef.current;
    if (lastInitial && lastInitial.isOpen === isOpen && lastInitial.nodeId === selectedNodeId) {
      return;
    }

    onChangeRef.current?.(true, selectedNodeId);
  }, [isOpen, selectedNodeId]);

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
    clearSelection,
  };
}
