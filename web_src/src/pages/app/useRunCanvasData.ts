import { useMemo } from "react";
import type { QueryClient } from "@tanstack/react-query";
import type {
  CanvasesCanvas,
  CanvasesCanvasEvent,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  ActionsAction,
  SuperplaneMeUser,
  TriggersTrigger,
} from "@/api-client";
import type { CanvasEdge, CanvasNode } from "@/ui/CanvasPage";
import { hydrateRunExecution, prepareData } from "./workflowPageHelpers";
import { stripCanvasNodeSetupWarningsForRunsView } from "./lib/node-integrations";

type UseRunCanvasDataParams = {
  isRunInspectionMode: boolean;
  selectedRun: CanvasesCanvasRun | null;
  selectedRunCanvas?: CanvasesCanvas | null;
  canvasLoading: boolean;
  triggersLoading: boolean;
  componentsLoading: boolean;
  isSelectedRunVersionLoading: boolean;
  allTriggers: TriggersTrigger[];
  allComponents: ActionsAction[];
  canvasId?: string;
  queryClient: QueryClient;
  me?: SuperplaneMeUser | null;
  visibleNodeExecutionsMap: Record<string, CanvasesCanvasNodeExecution[]>;
  selectedRunFullExecutions: CanvasesCanvasNodeExecution[] | undefined;
};

type UseRunCanvasPresentationParams = {
  isRunInspectionMode: boolean;
  selectedRun: CanvasesCanvasRun | null;
  runCanvasData: RunCanvasData | null;
  liveNodes: CanvasNode[];
  liveEdges: CanvasEdge[];
  isSelectedRunLoading: boolean;
  isSelectedRunVersionLoading: boolean;
  isSelectedRunExecutionsLoading: boolean;
};

export type RunCanvasData = {
  nodes: CanvasNode[];
  edges: CanvasEdge[];
  participantNodeIds: string[];
};

type RunExecutionRef = NonNullable<CanvasesCanvasRun["executions"]>[number];

export function mergeRunExecutionsForCanvas(
  runExecutions: RunExecutionRef[] = [],
  fullExecutions: CanvasesCanvasNodeExecution[] = [],
): CanvasesCanvasNodeExecution[] {
  const merged = new Map<string, CanvasesCanvasNodeExecution>();
  const executionsWithoutId: CanvasesCanvasNodeExecution[] = [];

  const appendExecution = (execution: RunExecutionRef | CanvasesCanvasNodeExecution) => {
    if (!execution.id) {
      executionsWithoutId.push(execution);
      return;
    }

    if (!merged.has(execution.id)) {
      merged.set(execution.id, execution);
    }
  };

  runExecutions.forEach(appendExecution);

  for (const execution of fullExecutions) {
    if (!execution.id) {
      executionsWithoutId.push(execution);
      continue;
    }

    const existing = merged.get(execution.id);
    merged.set(execution.id, existing ? { ...existing, ...execution } : execution);
  }

  return [...merged.values(), ...executionsWithoutId];
}

export function getRunCanvasFitKey({
  isRunInspectionMode,
  selectedRunId,
  runCanvasData,
  runCanvasLoading,
}: {
  isRunInspectionMode: boolean;
  selectedRunId: string | null;
  runCanvasData: RunCanvasData | null;
  runCanvasLoading: boolean;
}): string | null {
  if (!isRunInspectionMode || !selectedRunId || !runCanvasData || runCanvasLoading) return null;
  return `${selectedRunId}|${runCanvasData.participantNodeIds.slice().sort().join("|")}`;
}

export function useRunCanvasData({
  isRunInspectionMode,
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
}: UseRunCanvasDataParams): RunCanvasData | null {
  return useMemo(() => {
    if (
      !isRunInspectionMode ||
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
    const runExecutions = mergeRunExecutionsForCanvas(selectedRun.executions, selectedRunFullExecutions);
    if (selectedRun.rootEvent?.nodeId) {
      runNodeIds.add(selectedRun.rootEvent.nodeId);
      nodeEventsMap[selectedRun.rootEvent.nodeId] = [selectedRun.rootEvent as CanvasesCanvasEvent];
    }
    for (const execution of runExecutions) {
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
    isRunInspectionMode,
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

export function useRunCanvasPresentation({
  isRunInspectionMode,
  selectedRun,
  runCanvasData,
  liveNodes,
  liveEdges,
  isSelectedRunLoading,
  isSelectedRunVersionLoading,
  isSelectedRunExecutionsLoading,
}: UseRunCanvasPresentationParams) {
  if (!isRunInspectionMode) {
    return {
      nodes: liveNodes,
      edges: liveEdges,
      runCanvasLoading: false,
    };
  }

  return {
    nodes: runCanvasData?.nodes ?? [],
    edges: runCanvasData?.edges ?? [],
    runCanvasLoading:
      isSelectedRunLoading || (!!selectedRun && (isSelectedRunVersionLoading || isSelectedRunExecutionsLoading)),
  };
}
