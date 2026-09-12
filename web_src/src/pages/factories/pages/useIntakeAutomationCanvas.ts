import type { SuperplaneComponentsNode } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";
import { usePreparedCanvasGraph } from "@/pages/app/usePreparedCanvasGraph";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";

export interface IntakeAutomationGraph {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  factoryId?: string;
  specNodes?: SuperplaneComponentsNode[];
  organizationId?: string;
}

interface IntakeAutomationCanvasResult {
  graph: IntakeAutomationGraph;
  name?: string;
  isLoading: boolean;
  isError: boolean;
  refetch: () => Promise<unknown>;
}

/**
 * Loads the graph of the app that backs an intake source, prepared the same way
 * as on the canvas editor page so both views show the same nodes and edges.
 */
export function useIntakeAutomationCanvas(
  organizationId: string | undefined,
  appId: string | undefined,
): IntakeAutomationCanvasResult {
  const enabled = Boolean(organizationId && appId);
  const query = useCanvas(organizationId ?? "", appId ?? "", { enabled });
  const prepared = usePreparedCanvasGraph(query.data, organizationId);

  return {
    graph: {
      nodes: prepared.nodes,
      edges: prepared.edges,
      factoryId: query.data?.metadata?.factoryId,
      specNodes: query.data?.spec?.nodes ?? [],
      organizationId,
    },
    name: query.data?.metadata?.name,
    isLoading: enabled && (query.isPending || prepared.isLoading),
    isError: query.isError,
    refetch: query.refetch,
  };
}
