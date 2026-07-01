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
type OrderedRunExecution =
  | { kind: "identified"; id: string }
  | { kind: "anonymous"; execution: CanvasesCanvasNodeExecution };

export function mergeRunExecutionsForCanvas(
  runExecutions: RunExecutionRef[] = [],
  fullExecutions: CanvasesCanvasNodeExecution[] = [],
): CanvasesCanvasNodeExecution[] {
  const identifiedExecutions = new Map<string, CanvasesCanvasNodeExecution>();
  const orderedExecutions: OrderedRunExecution[] = [];

  const appendExecution = (execution: RunExecutionRef | CanvasesCanvasNodeExecution): void => {
    if (!execution.id) {
      orderedExecutions.push({ kind: "anonymous", execution });
      return;
    }

    if (!identifiedExecutions.has(execution.id)) {
      identifiedExecutions.set(execution.id, execution);
      orderedExecutions.push({ kind: "identified", id: execution.id });
    }
  };

  runExecutions.forEach(appendExecution);

  for (const execution of fullExecutions) {
    if (!execution.id) {
      orderedExecutions.push({ kind: "anonymous", execution });
      continue;
    }

    const existing = identifiedExecutions.get(execution.id);
    if (existing) {
      identifiedExecutions.set(execution.id, { ...existing, ...execution });
      continue;
    }

    identifiedExecutions.set(execution.id, execution);
    orderedExecutions.push({ kind: "identified", id: execution.id });
  }

  return orderedExecutions.map((execution) => {
    if (execution.kind === "anonymous") return execution.execution;
    return identifiedExecutions.get(execution.id)!;
  });
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
