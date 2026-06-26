import type {
  CanvasesCanvas,
  CanvasesCanvasVersion,
  ActionsAction,
  ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
  TriggersTrigger,
} from "@/api-client";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { buildDraftDiffMap } from "./draftNodeDiff";
import { prepareNode } from "./workflowPageHelpers";

type UseDraftVisualDiffArgs = {
  isViewingDraftVersion: boolean;
  canvas: CanvasesCanvas | null | undefined;
  liveCanvas?: CanvasesCanvas | null;
  latestDraftVersion?: CanvasesCanvasVersion;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  preparedNodes: CanvasNode[];
  preparedEdges: CanvasEdge[];
  allTriggers: TriggersTrigger[];
  allComponents: ActionsAction[];
  canvasId?: string;
  queryClient: QueryClient;
};

const edgeKey = (source: string, target: string, channel: string) => `${source}->${target}::${channel}`;

function applyNodeStatuses(nodes: CanvasNode[], statusMap: Record<string, string> | undefined) {
  if (!statusMap) {
    return nodes;
  }

  return nodes.map((node) => {
    const draftDiffStatus = statusMap[node.id];
    if (!draftDiffStatus) {
      return node;
    }

    return {
      ...node,
      data: {
        ...node.data,
        _draftDiffStatus: draftDiffStatus,
      },
    };
  });
}

function buildEdgeDiffSets(
  preparedEdges: CanvasEdge[],
  liveCanvas?: CanvasesCanvas | null,
): { liveEdgeSet: Set<string>; draftEdgeSet: Set<string>; liveEdges: Array<Record<string, unknown>> } | null {
  const liveEdges = (liveCanvas?.spec?.edges ?? []) as Array<Record<string, unknown>>;
  if (liveEdges.length === 0 && preparedEdges.length === 0) return null;
  const liveEdgeSet = new Set(
    liveEdges.map((e) => edgeKey(String(e.sourceId ?? ""), String(e.targetId ?? ""), String(e.channel ?? "default"))),
  );
  const draftEdgeSet = new Set(preparedEdges.map((e) => edgeKey(e.source, e.target, e.sourceHandle || "default")));
  return { liveEdgeSet, draftEdgeSet, liveEdges };
}

function countEdgeDiffs(
  preparedEdges: CanvasEdge[],
  liveCanvas?: CanvasesCanvas | null,
): { addedEdges: number; removedEdges: number } {
  const sets = buildEdgeDiffSets(preparedEdges, liveCanvas);
  if (!sets) return { addedEdges: 0, removedEdges: 0 };
  let addedEdges = 0;
  let removedEdges = 0;
  for (const e of preparedEdges) {
    const key = edgeKey(e.source, e.target, e.sourceHandle || "default");
    if (!sets.liveEdgeSet.has(key)) addedEdges += 1;
  }
  for (const e of sets.liveEdges) {
    const key = edgeKey(String(e.sourceId ?? ""), String(e.targetId ?? ""), String(e.channel ?? "default"));
    if (!sets.draftEdgeSet.has(key)) removedEdges += 1;
  }
  return { addedEdges, removedEdges };
}

function buildEdgesWithDiff(preparedEdges: CanvasEdge[], liveCanvas?: CanvasesCanvas | null) {
  const sets = buildEdgeDiffSets(preparedEdges, liveCanvas);
  if (!sets) return preparedEdges;

  const styledEdges = preparedEdges.map((edge) => {
    const key = edgeKey(edge.source, edge.target, edge.sourceHandle || "default");
    if (sets.liveEdgeSet.has(key)) return edge;
    return { ...edge, data: { ...edge.data, _draftDiffStatus: "added" } };
  });
  const removedEdges = sets.liveEdges
    .filter((edge) => {
      const key = edgeKey(String(edge.sourceId ?? ""), String(edge.targetId ?? ""), String(edge.channel ?? "default"));
      return !sets.draftEdgeSet.has(key);
    })
    .map((edge) => {
      const source = String(edge.sourceId ?? "");
      const target = String(edge.targetId ?? "");
      const channel = String(edge.channel ?? "default");
      return {
        id: `removed-edge-${source}->${target}::${channel}`,
        source,
        target,
        sourceHandle: channel,
        data: { _draftDiffStatus: "removed" },
      };
    });

  return [...styledEdges, ...removedEdges];
}

export type DraftVisualDiffResult = ReturnType<typeof useDraftVisualDiff>;

export function useDraftVisualDiff({
  isViewingDraftVersion,
  canvas,
  liveCanvas,
  latestDraftVersion,
  selectedCanvasVersion,
  preparedNodes,
  preparedEdges,
  allTriggers,
  allComponents,
  canvasId,
  queryClient,
}: UseDraftVisualDiffArgs) {
  const { visualDiffEnabled: enabled, toggleVisualDiff } = useVisualDiffToggle();
  const { showDeletedNodes, toggleShowDeletedNodes, showEdgeDiff, toggleShowEdgeDiff } = useDiffSubToggles();
  const liveVersionForDiff = useMemo(
    () => (liveCanvas?.spec ? ({ spec: liveCanvas.spec } as CanvasesCanvasVersion) : undefined),
    [liveCanvas?.spec],
  );
  const draftDiffResult = useMemo(() => {
    if (!isViewingDraftVersion || !canvas?.spec) {
      return undefined;
    }

    const draftVersionForDiff = {
      ...(latestDraftVersion || selectedCanvasVersion || {}),
      spec: canvas.spec,
    } as CanvasesCanvasVersion;

    return buildDraftDiffMap(liveVersionForDiff, draftVersionForDiff);
  }, [canvas?.spec, isViewingDraftVersion, latestDraftVersion, liveVersionForDiff, selectedCanvasVersion]);

  const nodes = useMemo(() => {
    const nodesWithStatuses = enabled ? applyNodeStatuses(preparedNodes, draftDiffResult?.statusMap) : preparedNodes;
    if (
      !enabled ||
      !showDeletedNodes ||
      !draftDiffResult?.removedNodes.length ||
      !liveCanvas?.spec?.nodes ||
      !canvasId
    ) {
      return nodesWithStatuses;
    }

    const liveNodes = liveCanvas.spec.nodes as ComponentsNode[];
    const liveEdges = liveCanvas.spec.edges as ComponentsEdge[] | undefined;
    const ghostNodes = draftDiffResult.removedNodes.map((removedNode) => {
      const prepared = prepareNode(
        liveNodes,
        removedNode as unknown as ComponentsNode,
        allTriggers,
        allComponents,
        {},
        {},
        {},
        canvasId,
        queryClient,
        undefined,
        liveEdges,
        "edit",
      );

      return {
        ...prepared,
        draggable: false,
        selectable: false,
        data: {
          ...prepared.data,
          _draftDiffStatus: "removed",
        },
      };
    });

    return [...nodesWithStatuses, ...ghostNodes];
  }, [
    allComponents,
    allTriggers,
    canvasId,
    draftDiffResult?.removedNodes,
    draftDiffResult?.statusMap,
    enabled,
    liveCanvas?.spec?.nodes,
    preparedNodes,
    queryClient,
    showDeletedNodes,
  ]);

  const edges = useMemo(() => {
    if (!enabled || !isViewingDraftVersion || !showEdgeDiff) {
      return preparedEdges;
    }

    return buildEdgesWithDiff(preparedEdges, liveCanvas);
  }, [enabled, isViewingDraftVersion, showEdgeDiff, liveCanvas, preparedEdges]);

  const diffCounts = useMemo(() => {
    let added = 0;
    let updated = 0;
    let removed = 0;

    if (draftDiffResult?.statusMap) {
      const values = Object.values(draftDiffResult.statusMap);
      added = values.filter((s) => s === "added").length;
      updated = values.filter((s) => s === "updated").length;
      removed = draftDiffResult.removedNodes?.length || 0;
    }

    const { addedEdges, removedEdges } = countEdgeDiffs(preparedEdges, liveCanvas);
    added += addedEdges;
    removed += removedEdges;

    return { added, updated, removed };
  }, [draftDiffResult, preparedEdges, liveCanvas]);

  return {
    nodes,
    edges,
    diffCounts,
    visualDiffEnabled: enabled,
    toggleVisualDiff,
    diffToggles: { showDeletedNodes, toggleShowDeletedNodes, showEdgeDiff, toggleShowEdgeDiff },
  };
}

const STORAGE_KEY = "visual-diff-enabled";

export function useVisualDiffToggle() {
  const [enabled, setEnabled] = useState(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored === null ? true : stored === "true";
  });

  const toggle = useCallback(() => {
    setEnabled((prev) => {
      const next = !prev;
      localStorage.setItem(STORAGE_KEY, String(next));
      return next;
    });
  }, []);

  return { visualDiffEnabled: enabled, toggleVisualDiff: toggle };
}

const SHOW_DELETED_NODES_KEY = "visual-diff-show-deleted-nodes";
const SHOW_EDGE_DIFF_KEY = "visual-diff-show-edges";

function useLocalStorageToggle(key: string, defaultValue = true) {
  const [value, setValue] = useState(() => {
    const stored = localStorage.getItem(key);
    return stored === null ? defaultValue : stored === "true";
  });
  const toggle = useCallback(() => {
    setValue((prev) => {
      const next = !prev;
      localStorage.setItem(key, String(next));
      return next;
    });
  }, [key]);
  return [value, toggle] as const;
}

function useDiffSubToggles() {
  const [showDeletedNodes, toggleShowDeletedNodes] = useLocalStorageToggle(SHOW_DELETED_NODES_KEY);
  const [showEdgeDiff, toggleShowEdgeDiff] = useLocalStorageToggle(SHOW_EDGE_DIFF_KEY);
  return { showDeletedNodes, toggleShowDeletedNodes, showEdgeDiff, toggleShowEdgeDiff };
}

export { useCanvasConsoleVersionDiff, type CanvasConsoleVersionDiffResult } from "./useCanvasConsoleVersionDiff";
