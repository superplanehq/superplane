import type { CanvasesCanvasRun } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { useCanvasWebsocket } from "@/hooks/useCanvasWebsocket";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  analyzingWorkOrderIds,
  backlogAnalysisRuns,
  backlogAnalysisRunsByWorkOrder,
  clearBacklogAnalysisPending,
  findBacklogAnalyzerCanvasId,
  mergeBacklogAnalysisRunSnapshots,
  pendingBacklogAnalysisIds,
  subscribeBacklogAnalysisPending,
  type BacklogAnalysisRun,
  upsertBacklogAnalysisRun,
} from "@/pages/factories/lib/backlogAnalysis";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useSyncExternalStore } from "react";

import { useFactoryAutomations } from "./useFactoryData";
import { useFactoryIntakes } from "./useFactoryIntakeData";

const BACKLOG_ANALYSIS_RUNS_LIMIT = 50;

export function backlogAnalysisRunsKey(organizationId: string, canvasId: string | undefined) {
  return ["backlog-analysis-runs", organizationId, canvasId] as const;
}

/**
 * Runs of the factory Backlog automation, keyed to the task each one
 * analyzes. The board reads it to show that a score is on the way, and the
 * task popup reads it to open the live log of the analysis.
 */
export function useBacklogAnalysisRuns(organizationId: string, canvasId: string | undefined) {
  return useQuery({
    queryKey: backlogAnalysisRunsKey(organizationId, canvasId),
    queryFn: async (): Promise<BacklogAnalysisRun[]> => {
      if (!canvasId) {
        return [];
      }
      const response = await canvasesListRuns(
        withOrganizationHeader({
          organizationId,
          path: { canvasId },
          query: { limit: BACKLOG_ANALYSIS_RUNS_LIMIT },
        }),
      );
      return backlogAnalysisRuns(canvasId, response.data?.runs ?? []);
    },
    enabled: Boolean(organizationId && canvasId),
    refetchOnWindowFocus: false,
    structuralSharing: (current, incoming) =>
      mergeBacklogAnalysisRunSnapshots(
        current as BacklogAnalysisRun[] | undefined,
        incoming as BacklogAnalysisRun[],
        canvasId ?? "",
      ),
  });
}

/**
 * Backlog analysis of one factory: which tasks wait for a score, and
 * the runs of each task. Intake automations can share the Backlog
 * name, so their canvases are excluded.
 */
export function useFactoryBacklogAnalysis(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const { data: intakes = [] } = useFactoryIntakes(organizationId, factoryId);
  const analyzerCanvasId = useMemo(
    () =>
      findBacklogAnalyzerCanvasId(
        apps,
        intakes.flatMap((intake) => (intake.canvasId ? [intake.canvasId] : [])),
      ),
    [apps, intakes],
  );
  const queryKey = useMemo(
    () => backlogAnalysisRunsKey(organizationId, analyzerCanvasId),
    [organizationId, analyzerCanvasId],
  );
  const { data: runs = [] } = useBacklogAnalysisRuns(organizationId, analyzerCanvasId);
  const runsByWorkOrder = useMemo(() => backlogAnalysisRunsByWorkOrder(runs), [runs]);
  const applyRunEvent = useCallback(
    (run: CanvasesCanvasRun) => {
      if (!analyzerCanvasId) {
        return;
      }
      queryClient.setQueryData<BacklogAnalysisRun[]>(queryKey, (current) =>
        upsertBacklogAnalysisRun(current, analyzerCanvasId, run),
      );
    },
    [analyzerCanvasId, queryClient, queryKey],
  );
  const resynchronizeRuns = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey });
  }, [queryClient, queryKey]);

  useCanvasWebsocket({
    canvasId: analyzerCanvasId ?? "",
    organizationId,
    processRuntimeEvents: false,
    enabled: Boolean(organizationId && analyzerCanvasId),
    onRunEvent: applyRunEvent,
    onConnectionOpen: resynchronizeRuns,
  });

  // Once a work order's real run is known (active or finished, i.e. the
  // Confidence score has arrived), the optimistic entry has done its job.
  useEffect(() => {
    for (const workOrderId of runsByWorkOrder.keys()) {
      clearBacklogAnalysisPending(workOrderId);
    }
  }, [runsByWorkOrder]);

  const pendingIds = useSyncExternalStore(subscribeBacklogAnalysisPending, pendingBacklogAnalysisIds);

  return useMemo(() => {
    const analyzingOrderIds = new Set(analyzingWorkOrderIds(runs));
    for (const workOrderId of pendingIds) {
      analyzingOrderIds.add(workOrderId);
    }
    return {
      analyzingOrderIds,
      runsByWorkOrder,
    };
  }, [runs, runsByWorkOrder, pendingIds]);
}

/**
 * Work orders whose Backlog analysis finished: they have at least one run
 * and none still in flight. The first-run analysis screen counts these as
 * scored tickets.
 */
export function useBacklogAnalysisScoredOrderIds(organizationId: string, factoryId: string): ReadonlySet<string> {
  const { analyzingOrderIds, runsByWorkOrder } = useFactoryBacklogAnalysis(organizationId, factoryId);
  return useMemo(() => {
    const scored = new Set<string>();
    for (const workOrderId of runsByWorkOrder.keys()) {
      if (!analyzingOrderIds.has(workOrderId)) {
        scored.add(workOrderId);
      }
    }
    return scored;
  }, [analyzingOrderIds, runsByWorkOrder]);
}
