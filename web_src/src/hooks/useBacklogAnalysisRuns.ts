import { canvasesListRuns } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  analyzingWorkOrderIds,
  backlogAnalysisRuns,
  backlogAnalysisRunsByWorkOrder,
  findBacklogAnalyzerCanvasId,
  hasActiveBacklogAnalysisRun,
  type BacklogAnalysisRun,
} from "@/pages/factories/lib/backlogAnalysis";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { useFactoryApps } from "./useFactoryData";
import { useFactoryIntakes } from "./useFactoryIntakeData";

const BACKLOG_ANALYSIS_RUNS_LIMIT = 50;

/** Analysis is short. Poll only while a run is in flight. */
const BACKLOG_ANALYSIS_POLL_MS = 4000;

/**
 * Runs of the factory Backlog automation, keyed to the work order each one
 * analyzes. The board reads it to show that a score is on the way, and the
 * work order popup reads it to open the live log of the analysis.
 */
export function useBacklogAnalysisRuns(organizationId: string, canvasId: string | undefined) {
  return useQuery({
    queryKey: ["backlog-analysis-runs", organizationId, canvasId],
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
    refetchInterval: (query) =>
      hasActiveBacklogAnalysisRun(query.state.data ?? []) ? BACKLOG_ANALYSIS_POLL_MS : false,
  });
}

/**
 * Backlog analysis of one factory: which work orders wait for a score, and
 * the runs of each work order. Intake automations can share the Backlog
 * name, so their canvases are excluded.
 */
export function useFactoryBacklogAnalysis(organizationId: string, factoryId: string) {
  const { data: apps = [] } = useFactoryApps(organizationId, factoryId);
  const { data: intakes = [] } = useFactoryIntakes(organizationId, factoryId);
  const analyzerCanvasId = useMemo(
    () =>
      findBacklogAnalyzerCanvasId(
        apps,
        intakes.flatMap((intake) => (intake.canvasId ? [intake.canvasId] : [])),
      ),
    [apps, intakes],
  );
  const { data: runs = [] } = useBacklogAnalysisRuns(organizationId, analyzerCanvasId);

  return useMemo(
    () => ({
      analyzingOrderIds: analyzingWorkOrderIds(runs),
      runsByWorkOrder: backlogAnalysisRunsByWorkOrder(runs),
    }),
    [runs],
  );
}
