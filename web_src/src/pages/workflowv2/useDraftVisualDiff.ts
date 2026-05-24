import type {
  CanvasesCanvas,
  CanvasesCanvasVersion,
  SuperplaneActionsAction,
  SuperplaneComponentsEdge as ComponentsEdge,
  SuperplaneComponentsNode as ComponentsNode,
  TriggersTrigger,
} from "@/api-client";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { buildDraftDiffMap } from "./draftNodeDiff";
import { prepareNode } from "./workflowPageHelpers";

type UseDraftVisualDiffArgs = {
  enabled: boolean;
  isViewingDraftVersion: boolean;
  canvas: CanvasesCanvas | null | undefined;
  liveCanvasVersion?: CanvasesCanvasVersion;
  latestDraftVersion?: CanvasesCanvasVersion;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  preparedNodes: CanvasNode[];
  preparedEdges: CanvasEdge[];
  allTriggers: TriggersTrigger[];
  allComponents: SuperplaneActionsAction[];
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

function buildEdgesWithDiff(preparedEdges: CanvasEdge[], liveCanvasVersion?: CanvasesCanvasVersion) {
  if (!liveCanvasVersion?.spec?.edges) {
    return preparedEdges;
  }

  const liveEdges = liveCanvasVersion.spec.edges as Array<Record<string, unknown>>;
  const liveEdgeSet = new Set(
    liveEdges.map((edge) =>
      edgeKey(String(edge.sourceId ?? ""), String(edge.targetId ?? ""), String(edge.channel ?? "default")),
    ),
  );
  const draftEdgeSet = new Set(
    preparedEdges.map((edge) => edgeKey(edge.source, edge.target, edge.sourceHandle || "default")),
  );
  const styledEdges = preparedEdges.map((edge) => {
    const key = edgeKey(edge.source, edge.target, edge.sourceHandle || "default");
    if (liveEdgeSet.has(key)) {
      return edge;
    }

    return {
      ...edge,
      data: {
        ...edge.data,
        _draftDiffStatus: "added",
      },
    };
  });
  const removedEdges = liveEdges
    .filter((edge) => {
      const key = edgeKey(String(edge.sourceId ?? ""), String(edge.targetId ?? ""), String(edge.channel ?? "default"));
      return !draftEdgeSet.has(key);
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
        data: {
          _draftDiffStatus: "removed",
        },
      };
    });

  return [...styledEdges, ...removedEdges];
}

export function useDraftVisualDiff({
  enabled,
  isViewingDraftVersion,
  canvas,
  liveCanvasVersion,
  latestDraftVersion,
  selectedCanvasVersion,
  preparedNodes,
  preparedEdges,
  allTriggers,
  allComponents,
  canvasId,
  queryClient,
}: UseDraftVisualDiffArgs) {
  const draftDiffResult = useMemo(() => {
    if (!enabled || !isViewingDraftVersion || !canvas?.spec) {
      return undefined;
    }

    const draftVersionForDiff = {
      ...(latestDraftVersion || selectedCanvasVersion || {}),
      spec: canvas.spec,
    } as CanvasesCanvasVersion;

    return buildDraftDiffMap(liveCanvasVersion, draftVersionForDiff);
  }, [enabled, canvas?.spec, isViewingDraftVersion, latestDraftVersion, liveCanvasVersion, selectedCanvasVersion]);

  const nodes = useMemo(() => {
    const nodesWithStatuses = applyNodeStatuses(preparedNodes, draftDiffResult?.statusMap);
    if (!draftDiffResult?.removedNodes.length || !liveCanvasVersion?.spec?.nodes || !canvasId) {
      return nodesWithStatuses;
    }

    const liveNodes = liveCanvasVersion.spec.nodes as ComponentsNode[];
    const liveEdges = liveCanvasVersion.spec.edges as ComponentsEdge[] | undefined;
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
    liveCanvasVersion?.spec,
    preparedNodes,
    queryClient,
  ]);

  const edges = useMemo(() => {
    if (!enabled || !isViewingDraftVersion) {
      return preparedEdges;
    }

    return buildEdgesWithDiff(preparedEdges, liveCanvasVersion);
  }, [enabled, isViewingDraftVersion, liveCanvasVersion, preparedEdges]);

  const diffCounts = useMemo(() => {
    if (!isViewingDraftVersion || !canvas?.spec) return { added: 0, updated: 0, removed: 0 };

    let added = 0;
    let updated = 0;
    let removed = 0;

    // Node diffs
    const draftVersionForDiff = {
      ...(latestDraftVersion || selectedCanvasVersion || {}),
      spec: canvas.spec,
    } as CanvasesCanvasVersion;
    const diffResult = buildDraftDiffMap(liveCanvasVersion, draftVersionForDiff);
    if (diffResult?.statusMap) {
      const values = Object.values(diffResult.statusMap);
      added = values.filter((s) => s === "added").length;
      updated = values.filter((s) => s === "updated").length;
      removed = diffResult.removedNodes?.length || 0;
    }

    // Edge diffs
    if (liveCanvasVersion?.spec?.edges && preparedEdges.length > 0) {
      const liveEdges = liveCanvasVersion.spec.edges as Array<Record<string, unknown>>;
      const liveEdgeSet = new Set(
        liveEdges.map((e) => edgeKey(String(e.sourceId ?? ""), String(e.targetId ?? ""), String(e.channel ?? "default"))),
      );
      const draftEdgeSet = new Set(
        preparedEdges.map((e) => edgeKey(e.source, e.target, e.sourceHandle || "default")),
      );
      for (const e of preparedEdges) {
        const key = edgeKey(e.source, e.target, e.sourceHandle || "default");
        if (!liveEdgeSet.has(key)) added += 1;
      }
      for (const e of liveEdges) {
        const key = edgeKey(String(e.sourceId ?? ""), String(e.targetId ?? ""), String(e.channel ?? "default"));
        if (!draftEdgeSet.has(key)) removed += 1;
      }
    }
    return { added, updated, removed };
  }, [isViewingDraftVersion, canvas?.spec, liveCanvasVersion, latestDraftVersion, selectedCanvasVersion, preparedEdges]);

  return { nodes, edges, diffCounts };
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
