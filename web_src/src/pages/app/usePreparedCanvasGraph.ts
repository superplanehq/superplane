import { useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

import type { ActionsAction, CanvasesCanvas, TriggersTrigger } from "@/api-client";
import { useTriggers } from "@/hooks/useCanvasData";
import { useComponents } from "@/hooks/useComponentData";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useMe } from "@/hooks/useMe";
import { actionsFromCapabilities, triggersFromCapabilities } from "@/lib/capabilities";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";

import { prepareData } from "./workflowPageHelpers";

interface PreparedCanvasGraph {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  isLoading: boolean;
}

const EMPTY_NODE_MAP = {};

/**
 * Builds the nodes and edges of a canvas the same way the live canvas does,
 * without run data. Use it to show a canvas outside its page, for example in a
 * settings popup, so both views stay identical.
 */
export function usePreparedCanvasGraph(
  canvas: CanvasesCanvas | undefined,
  organizationId: string | undefined,
): PreparedCanvasGraph {
  const queryClient = useQueryClient();
  const { data: me } = useMe();
  const { data: triggers = [], isLoading: triggersLoading } = useTriggers();
  const { data: components = [], isLoading: componentsLoading } = useComponents(organizationId ?? "");
  const { data: integrations = [], isLoading: integrationsLoading } = useAvailableIntegrations();

  const catalog = useMemo(() => {
    const allTriggers: TriggersTrigger[] = [...triggers];
    const allComponents: ActionsAction[] = [...components];
    for (const integration of integrations) {
      if (!integration.capabilities) {
        continue;
      }
      allTriggers.push(...triggersFromCapabilities(integration.capabilities));
      allComponents.push(...actionsFromCapabilities(integration.capabilities));
    }
    return { triggers: allTriggers, components: allComponents };
  }, [components, integrations, triggers]);

  const isLoading = triggersLoading || componentsLoading || integrationsLoading;

  return useMemo(() => {
    if (!canvas || isLoading) {
      return { nodes: [], edges: [], isLoading };
    }

    const { nodes, edges } = prepareData(
      canvas,
      catalog.triggers,
      catalog.components,
      EMPTY_NODE_MAP,
      EMPTY_NODE_MAP,
      EMPTY_NODE_MAP,
      canvas.metadata?.id ?? "",
      queryClient,
      me,
      "live",
    );

    return { nodes, edges, isLoading: false };
  }, [canvas, catalog, isLoading, me, queryClient]);
}
