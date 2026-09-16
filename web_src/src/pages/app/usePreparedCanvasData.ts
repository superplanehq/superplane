import { useMemo } from "react";
import type { QueryClient } from "@tanstack/react-query";
import type {
  ActionsAction,
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeQueueItem,
  SuperplaneMeUser,
  TriggersTrigger,
} from "@/api-client";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";
import type { TriggerActionModal } from "./mappers/types";
import { prepareData } from "./workflowPageHelpers";

type PreparedCanvasDataArgs = {
  canvas?: CanvasesCanvas;
  triggers: TriggersTrigger[];
  components: ActionsAction[];
  nodeEventsMap: Record<string, CanvasesCanvasEvent[]>;
  nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>;
  canvasId?: string;
  queryClient: QueryClient;
  user?: SuperplaneMeUser | null;
  canvasMode: "live" | "edit";
  openModal?: (modal: TriggerActionModal) => void;
  organizationId?: string;
  enabled: boolean;
};

export function usePreparedCanvasData({
  canvas,
  triggers,
  components,
  nodeEventsMap,
  nodeExecutionsMap,
  nodeQueueItemsMap,
  canvasId,
  queryClient,
  user,
  canvasMode,
  openModal,
  organizationId,
  enabled,
}: PreparedCanvasDataArgs): { nodes: CanvasNode[]; edges: CanvasEdge[] } {
  return useMemo(() => {
    if (!enabled || !canvas || !canvasId) {
      return { nodes: [], edges: [] };
    }

    return prepareData(
      canvas,
      triggers,
      components,
      nodeEventsMap,
      nodeExecutionsMap,
      nodeQueueItemsMap,
      canvasId,
      queryClient,
      user,
      canvasMode,
      openModal,
      organizationId,
    );
  }, [
    canvas,
    triggers,
    components,
    nodeEventsMap,
    nodeExecutionsMap,
    nodeQueueItemsMap,
    canvasId,
    queryClient,
    user,
    canvasMode,
    openModal,
    organizationId,
    enabled,
  ]);
}
