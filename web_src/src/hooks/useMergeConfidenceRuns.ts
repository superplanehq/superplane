import type { CanvasesCanvasRun, FactoriesFactoryPullRequest } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  mergeConfidenceCanvases,
  mergeConfidenceRunsForPullRequests,
  mergeMergeConfidenceRunSnapshots,
  type MergeConfidenceLogRun,
} from "@/pages/factories/lib/mergeConfidenceRuns";
import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";

import { useFactoryAutomations } from "./useFactoryData";

const MERGE_CONFIDENCE_RUNS_LIMIT = 50;

export function mergeConfidenceRunsKey(organizationId: string, canvasId: string) {
  return ["merge-confidence-runs", organizationId, canvasId] as const;
}

/**
 * Recent Merge confidence runs for the task pull requests. A later score
 * is another canvas run, so the task log can list each one.
 */
export function useFactoryMergeConfidenceRuns(
  organizationId: string,
  factoryId: string,
  pullRequests: FactoriesFactoryPullRequest[],
): { runs: MergeConfidenceLogRun[]; canvasIds: string[] } {
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const canvases = useMemo(() => mergeConfidenceCanvases(apps), [apps]);
  const queries = useQueries({
    queries: canvases.map((canvas) => ({
      queryKey: mergeConfidenceRunsKey(organizationId, canvas.id),
      queryFn: () => listMergeConfidenceRuns(organizationId, canvas.id),
      enabled: Boolean(organizationId && canvas.id),
      refetchOnWindowFocus: false,
      structuralSharing: (current: unknown, incoming: unknown) =>
        mergeMergeConfidenceRunSnapshots(current as CanvasesCanvasRun[] | undefined, incoming as CanvasesCanvasRun[]),
    })),
  });
  const runs = useMemo(
    () =>
      canvases.flatMap((canvas, index) =>
        mergeConfidenceRunsForPullRequests(canvas, queries[index]?.data ?? [], pullRequests),
      ),
    [canvases, pullRequests, queries],
  );
  return {
    runs,
    canvasIds: canvases.map((canvas) => canvas.id),
  };
}

async function listMergeConfidenceRuns(organizationId: string, canvasId: string): Promise<CanvasesCanvasRun[]> {
  const response = await canvasesListRuns(
    withOrganizationHeader({
      organizationId,
      path: { canvasId },
      query: { limit: MERGE_CONFIDENCE_RUNS_LIMIT },
    }),
  );
  return response.data?.runs ?? [];
}
