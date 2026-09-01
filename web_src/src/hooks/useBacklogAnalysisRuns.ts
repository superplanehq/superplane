import type { FactoriesWorkOrder } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  analyzingWorkOrderIds,
  backlogAnalysisRuns,
  backlogAnalysisRunsByWorkOrder,
  clearBacklogAnalysisPending,
  findBacklogAnalyzerCanvasId,
  hasActiveBacklogAnalysisRun,
  pendingBacklogAnalysisIds,
  subscribeBacklogAnalysisPending,
  type BacklogAnalysisRun,
} from "@/pages/factories/lib/backlogAnalysis";
import { getWorkOrderDisplayStatus } from "@/pages/factories/lib/workOrderProgress";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState, useSyncExternalStore } from "react";

import { useFactoryApps, useFactoryWorkOrders } from "./useFactoryData";
import { useFactoryIntakes } from "./useFactoryIntakeData";

const BACKLOG_ANALYSIS_RUNS_LIMIT = 50;

/** Analysis is short. Poll only while a run is in flight. */
const BACKLOG_ANALYSIS_POLL_MS = 4000;

/**
 * Bound on how long a fresh draft (no run yet, likely created through the
 * API) keeps the runs query polling. Wide enough to catch the run once the
 * factory creates it asynchronously; bounded so a draft that is never
 * analyzed does not poll forever.
 */
const RECENT_DRAFT_ANALYSIS_MS = 120_000;

/**
 * Runs of the factory Backlog automation, keyed to the task each one
 * analyzes. The board reads it to show that a score is on the way, and the
 * task popup reads it to open the live log of the analysis.
 */
export function useBacklogAnalysisRuns(organizationId: string, canvasId: string | undefined, keepPolling = false) {
  const pendingIds = useSyncExternalStore(subscribeBacklogAnalysisPending, pendingBacklogAnalysisIds);

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
      hasActiveBacklogAnalysisRun(query.state.data ?? []) || pendingIds.size > 0 || keepPolling
        ? BACKLOG_ANALYSIS_POLL_MS
        : false,
  });
}

/**
 * Backlog analysis of one factory: which tasks wait for a score, and
 * the runs of each task. Intake automations can share the Backlog
 * name, so their canvases are excluded.
 */
export function useFactoryBacklogAnalysis(organizationId: string, factoryId: string) {
  const { data: apps = [] } = useFactoryApps(organizationId, factoryId);
  const { data: intakes = [] } = useFactoryIntakes(organizationId, factoryId);
  const { data: orders = [] } = useFactoryWorkOrders(organizationId, factoryId);
  const analyzerCanvasId = useMemo(
    () =>
      findBacklogAnalyzerCanvasId(
        apps,
        intakes.flatMap((intake) => (intake.canvasId ? [intake.canvasId] : [])),
      ),
    [apps, intakes],
  );

  // Whether the runs query should keep polling for a recent draft that has
  // no run yet (typically an order created through the API). Derived after
  // each fetch from that fetch's own data, so it lags one render behind a
  // fresh draft appearing — acceptable since the window is two minutes wide.
  const [keepPolling, setKeepPolling] = useState(false);
  const { data: runs = [] } = useBacklogAnalysisRuns(organizationId, analyzerCanvasId, keepPolling);
  const runsByWorkOrder = useMemo(() => backlogAnalysisRunsByWorkOrder(runs), [runs]);

  // Once a work order's real run is known (active or finished, i.e. the
  // Confidence score has arrived), the optimistic entry has done its job.
  useEffect(() => {
    for (const workOrderId of runsByWorkOrder.keys()) {
      clearBacklogAnalysisPending(workOrderId);
    }
  }, [runsByWorkOrder]);

  useEffect(() => {
    setKeepPolling(hasRecentUnanalyzedDraft(orders, runsByWorkOrder));
  }, [orders, runsByWorkOrder]);

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
 * Whether some draft has no Backlog run yet but was created recently
 * enough that one might still be created asynchronously (typically an
 * order created through the API rather than the UI). Used only to widen
 * the poll window; the indicator itself waits for the real run.
 */
function hasRecentUnanalyzedDraft(
  orders: FactoriesWorkOrder[],
  runsByWorkOrder: Map<string, BacklogAnalysisRun[]>,
  now = Date.now(),
): boolean {
  return orders.some((order) => {
    if (!order.id || getWorkOrderDisplayStatus(order) !== "draft" || runsByWorkOrder.has(order.id)) {
      return false;
    }
    const createdAtMs = Date.parse(order.createdAt ?? "");
    return Number.isFinite(createdAtMs) && now - createdAtMs <= RECENT_DRAFT_ANALYSIS_MS;
  });
}
