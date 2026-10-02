import type { CanvasesCanvasRun, CanvasesListRunsResponse, FactoriesFactoryPullRequest } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  mergeConfidenceCanvases,
  mergeConfidenceRunsForPullRequests,
  mergeMergeConfidenceRunSnapshots,
  type MergeConfidenceLogRun,
} from "@/pages/factories/lib/mergeConfidenceRuns";
import { useQueries, useQueryClient } from "@tanstack/react-query";
import { useMemo, useRef } from "react";

import { useFactoryAutomations } from "./useFactoryData";

const MERGE_CONFIDENCE_RUNS_PAGE_LIMIT = 25;
const MERGE_CONFIDENCE_RUNS_MAX_PAGES = 100;

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
  const queryClient = useQueryClient();
  const idsAtFetchStart = useRef(new Map<string, ReadonlySet<string>>());
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const canvases = useMemo(() => mergeConfidenceCanvases(apps), [apps]);
  const queries = useQueries({
    queries: canvases.map((canvas) => ({
      queryKey: mergeConfidenceRunsKey(organizationId, canvas.id),
      queryFn: () => {
        const current = queryClient.getQueryData<CanvasesCanvasRun[]>(
          mergeConfidenceRunsKey(organizationId, canvas.id),
        );
        idsAtFetchStart.current.set(canvas.id, new Set((current ?? []).flatMap((run) => (run.id ? [run.id] : []))));
        return listMergeConfidenceRuns(organizationId, canvas.id);
      },
      enabled: Boolean(organizationId && canvas.id),
      refetchOnWindowFocus: false,
      structuralSharing: (current: unknown, incoming: unknown) =>
        mergeMergeConfidenceRunSnapshots(
          current as CanvasesCanvasRun[] | undefined,
          incoming as CanvasesCanvasRun[],
          idsAtFetchStart.current.get(canvas.id),
        ),
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
  const runs: CanvasesCanvasRun[] = [];
  const seen = new Set<string>();
  let before: string | undefined;
  let loadedCount = 0;

  for (let page = 0; page < MERGE_CONFIDENCE_RUNS_MAX_PAGES; page += 1) {
    const response = await canvasesListRuns(
      withOrganizationHeader({
        organizationId,
        path: { canvasId },
        query: {
          limit: MERGE_CONFIDENCE_RUNS_PAGE_LIMIT,
          ...(before ? { before } : {}),
        },
      }),
    );
    const data = response.data;
    const pageRuns = data?.runs ?? [];
    for (const run of pageRuns) {
      if (run.id && seen.has(run.id)) {
        continue;
      }
      if (run.id) {
        seen.add(run.id);
      }
      runs.push(run);
    }
    loadedCount += pageRuns.length;

    const nextBefore = nextRunListCursor({ before, pageRuns, loadedCount, response: data });
    if (!nextBefore) {
      break;
    }
    before = nextBefore;
  }

  return runs;
}

function nextRunListCursor(options: {
  before?: string;
  pageRuns: CanvasesCanvasRun[];
  loadedCount: number;
  response: CanvasesListRunsResponse | undefined;
}): string | undefined {
  const { before, pageRuns, loadedCount, response } = options;
  if (pageRuns.length === 0 || !response?.lastTimestamp || response.lastTimestamp === before) {
    return undefined;
  }

  const totalCount = response.totalCount;
  if (typeof totalCount === "number" && totalCount > 0 && loadedCount >= totalCount) {
    return undefined;
  }
  if (response.hasNextPage === false) {
    return undefined;
  }

  return response.lastTimestamp;
}
