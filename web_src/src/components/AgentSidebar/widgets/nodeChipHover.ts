import type { CanvasesCanvas, SuperplaneComponentsNode } from "@/api-client";
import type { MetadataItem } from "@/ui/metadataList";
import { getComponentBaseMapper, getTriggerRenderer } from "@/pages/app/mappers";
import { buildComponentDefinition, buildNodeInfo } from "@/pages/app/utils";

type CanvasEdge = NonNullable<NonNullable<CanvasesCanvas["spec"]>["edges"]>[number];

export type NodeNeighbor = {
  id: string;
  label: string;
  direction: "upstream" | "downstream";
};

export type NodeNeighborList = {
  items: NodeNeighbor[];
  overflow: number;
};

const CONFIG_SUMMARIZERS: Record<string, (config: Record<string, unknown>) => string> = {
  http: (c) => `${c.method || "GET"} ${c.url || ""}`.trim(),
  ssh: (c) => `${c.username || "root"}@${c.host || ""}`.trim(),
  if: (c) => String(c.expression || "").trim(),
  filter: (c) => String(c.expression || "").trim(),
  wait: (c) => {
    const value = c.duration || c.waitFor || "";
    return value ? `Wait: ${value}` : "";
  },
  webhook: (c) => `Auth: ${c.authentication || "none"}`,
  schedule: (c) => {
    const cron = c.cron || "";
    return cron ? `Cron: ${cron}` : "";
  },
  approval: (c) => String(c.message || "Approval required").trim(),
};

export function getConfigSummary(component?: string, config?: Record<string, unknown>): string | null {
  if (!component || !config) return null;
  const summarizer = CONFIG_SUMMARIZERS[component];
  if (!summarizer) return null;
  const summary = summarizer(config);
  return summary || null;
}

function isTriggerNode(node: SuperplaneComponentsNode): boolean {
  return node.type === "TYPE_TRIGGER";
}

function fallbackMetadataItems(component: string | undefined, configuration: unknown): MetadataItem[] {
  const summary = getConfigSummary(
    component,
    configuration && typeof configuration === "object" ? (configuration as Record<string, unknown>) : undefined,
  );
  if (!summary) return [];
  return [{ icon: "info", label: summary }];
}

/**
 * Same secondary details the canvas node card shows (repo, channel, URL, …),
 * via the component/trigger mapper registry. Falls back to built-in one-line
 * summarizers when the mapper returns no metadata.
 */
export function getNodeHoverMetadataItems(node: SuperplaneComponentsNode, component?: string): MetadataItem[] {
  const componentName = component || node.component || "";
  const nodeInfo = buildNodeInfo(node);
  const definition = buildComponentDefinition({ name: componentName });

  try {
    if (isTriggerNode(node)) {
      const props = getTriggerRenderer(componentName).getTriggerProps({
        node: nodeInfo,
        definition,
        lastEvent: undefined,
        canvasMode: "live",
      });
      if (props.metadata?.length) {
        return props.metadata;
      }
    } else {
      const props = getComponentBaseMapper(componentName).props({
        nodes: [nodeInfo],
        node: nodeInfo,
        componentDefinition: definition,
        lastExecutions: [],
        currentUser: undefined,
        actions: {
          invokeNodeExecutionHook: async () => undefined,
        },
        canvasMode: "live",
      });
      if (props.metadata?.length) {
        return props.metadata;
      }
    }
  } catch (error) {
    // Hover cards must not crash on mapper failures; fall through to summarizers.
    if (import.meta.env.DEV) {
      console.warn(`Failed to resolve hover metadata for ${componentName}`, error);
    }
  }

  return fallbackMetadataItems(componentName, node.configuration);
}

function nodeLabel(nodesById: Map<string, SuperplaneComponentsNode>, id: string): string {
  const node = nodesById.get(id);
  return node?.name || id;
}

/** Upstream neighbors first, then downstream. Caps visible items and reports overflow. */
export function listNodeNeighbors(
  nodeId: string | undefined,
  edges: CanvasEdge[] | undefined,
  nodes: SuperplaneComponentsNode[] | undefined,
  maxVisible = 4,
): NodeNeighborList {
  if (!nodeId || !edges?.length) {
    return { items: [], overflow: 0 };
  }

  const nodesById = new Map((nodes ?? []).filter((node) => node.id).map((node) => [node.id!, node]));
  const upstream: NodeNeighbor[] = [];
  const downstream: NodeNeighbor[] = [];
  const seenUpstream = new Set<string>();
  const seenDownstream = new Set<string>();

  for (const edge of edges) {
    if (edge.targetId === nodeId && edge.sourceId && !seenUpstream.has(edge.sourceId)) {
      seenUpstream.add(edge.sourceId);
      upstream.push({
        id: edge.sourceId,
        label: nodeLabel(nodesById, edge.sourceId),
        direction: "upstream",
      });
    }
    if (edge.sourceId === nodeId && edge.targetId && !seenDownstream.has(edge.targetId)) {
      seenDownstream.add(edge.targetId);
      downstream.push({
        id: edge.targetId,
        label: nodeLabel(nodesById, edge.targetId),
        direction: "downstream",
      });
    }
  }

  const all = [...upstream, ...downstream];
  if (all.length <= maxVisible) {
    return { items: all, overflow: 0 };
  }

  return {
    items: all.slice(0, maxVisible),
    overflow: all.length - maxVisible,
  };
}
