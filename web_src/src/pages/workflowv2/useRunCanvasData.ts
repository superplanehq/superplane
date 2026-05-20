import { useMemo } from "react";
import type { QueryClient } from "@tanstack/react-query";
import type {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneActionsAction,
  SuperplaneMeUser,
  TriggersTrigger,
} from "@/api-client";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";
import { hydrateRunExecution, prepareData } from "./workflowPageHelpers";
import { stripCanvasNodeSetupWarningsForRunsView } from "./lib/node-integrations";

type UseRunCanvasDataParams = {
  isRunsMode: boolean;
  selectedRun: CanvasesCanvasRun | null;
  selectedRunCanvas?: CanvasesCanvas | null;
  canvasLoading: boolean;
  triggersLoading: boolean;
  componentsLoading: boolean;
  isSelectedRunVersionLoading: boolean;
  allTriggers: TriggersTrigger[];
  allComponents: SuperplaneActionsAction[];
  canvasId?: string;
  queryClient: QueryClient;
  me?: SuperplaneMeUser | null;
  visibleNodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  selectedRunFullExecutions: CanvasesCanvasNodeExecution[] | undefined;
};

export function useRunCanvasData({
  isRunsMode,
  selectedRun,
  selectedRunCanvas,
  canvasLoading,
  triggersLoading,
  componentsLoading,
  isSelectedRunVersionLoading,
  allTriggers,
  allComponents,
  canvasId,
  queryClient,
  me,
  visibleNodeExecutionsMap,
  selectedRunFullExecutions,
}: UseRunCanvasDataParams): {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  participantNodeIds: string[];
} | null {
  return useMemo(() => {
    if (
      !isRunsMode ||
      !selectedRunCanvas ||
      canvasLoading ||
      triggersLoading ||
      componentsLoading ||
      isSelectedRunVersionLoading
    ) {
      return null;
    }
    if (!selectedRun) {
      return { nodes: [], edges: [], participantNodeIds: [] };
    }
    const runNodeIds = new Set<string>();
    const nodeEventsMap: Record<string, CanvasesCanvasEvent[]> = {};
    const nodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]> = {};
    if (selectedRun.rootEvent?.nodeId) {
      runNodeIds.add(selectedRun.rootEvent.nodeId);
      nodeEventsMap[selectedRun.rootEvent.nodeId] = [selectedRun.rootEvent as CanvasesCanvasEvent];
    }
    for (const execution of selectedRun.executions || []) {
      if (!execution.nodeId) continue;
      runNodeIds.add(execution.nodeId);
      if (!nodeExecutionsMap[execution.nodeId]) {
        nodeExecutionsMap[execution.nodeId] = [];
      }

      nodeExecutionsMap[execution.nodeId].push(
        hydrateRunExecution(
          execution,
          selectedRunFullExecutions,
          visibleNodeExecutionsMap[execution.nodeId],
          selectedRun.rootEvent,
        ),
      );
    }
    const prepared = prepareData(
      selectedRunCanvas,
      allTriggers,
      allComponents,
      nodeEventsMap,
      nodeExecutionsMap,
      {},
      canvasId!,
      queryClient,
      me,
      "live",
    );
    return {
      nodes: stripCanvasNodeSetupWarningsForRunsView(prepared.nodes),
      edges: prepared.edges,
      participantNodeIds: Array.from(runNodeIds),
    };
  }, [
    isRunsMode,
    selectedRun,
    selectedRunCanvas,
    canvasLoading,
    triggersLoading,
    componentsLoading,
    isSelectedRunVersionLoading,
    allTriggers,
    allComponents,
    canvasId,
    queryClient,
    me,
    visibleNodeExecutionsMap,
    selectedRunFullExecutions,
  ]);
}
