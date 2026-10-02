import type {
  CanvasesCanvasRun,
  FactoriesFactoryPullRequest,
  FactoriesListWorkOrderEventsResponse,
  FactoriesWorkOrderCheck,
} from "@/api-client";
import { canvasesDescribeRun, canvasesListRuns, factoriesListWorkOrderEvents } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  getWorkOrderEventsNextPageParam,
  WORK_ORDER_EVENTS_PAGE_LIMIT,
} from "@/pages/factories/lib/workOrderEventsPagination";
import {
  mergeConfidenceCanvases,
  mergeConfidenceRunRefsFromTask,
  mergeConfidenceRunsForPullRequests,
  mergeConfidenceTaskKey,
  mergeMergeConfidenceRunSnapshots,
  upsertMergeConfidenceCanvasRun,
  type MergeConfidenceCanvas,
  type MergeConfidenceLogRun,
  type MergeConfidenceRunRef,
} from "@/pages/factories/lib/mergeConfidenceRuns";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useRef } from "react";

import { useFactoryAutomations } from "./useFactoryData";

const MERGE_CONFIDENCE_RUNS_PAGE_LIMIT = 25;
const MERGE_CONFIDENCE_EVENTS_MAX_PAGES = 20;

export function mergeConfidenceRunsKey(organizationId: string, canvasId: string, taskKey: string) {
  return ["merge-confidence-runs", organizationId, canvasId, taskKey] as const;
}

export function mergeConfidenceDescribedRunKey(organizationId: string, canvasId: string, runId: string) {
  return ["merge-confidence-described-run", organizationId, canvasId, runId] as const;
}

export function mergeConfidenceScoredRunRefsPrefix(organizationId: string, factoryId: string, orderId: string) {
  return ["merge-confidence-scored-run-refs", organizationId, factoryId, orderId] as const;
}

function mergeConfidenceScoredRunRefsKey(
  organizationId: string,
  factoryId: string,
  orderId: string,
  canvasIds: string,
) {
  return [...mergeConfidenceScoredRunRefsPrefix(organizationId, factoryId, orderId), canvasIds] as const;
}

/**
 * Merge confidence runs for this task's pull requests. A later score is
 * another canvas run, so the task log can list each one.
 *
 * The log does not scan the canvas history. It reads the newest page for a
 * score still in progress, and describes runs already recorded on the task.
 */
export function useFactoryMergeConfidenceRuns(
  organizationId: string,
  factoryId: string,
  pullRequests: FactoriesFactoryPullRequest[],
  orderId = "",
  checks?: FactoriesWorkOrderCheck[],
): { runs: MergeConfidenceLogRun[]; canvasIds: string[]; taskKey: string } {
  const queryClient = useQueryClient();
  const idsAtFetchStart = useRef(new Map<string, ReadonlySet<string>>());
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const canvases = useMemo(() => mergeConfidenceCanvases(apps), [apps]);
  const taskKey = useMemo(() => mergeConfidenceTaskKey(orderId, pullRequests), [orderId, pullRequests]);
  const canvasIdSet = useMemo(() => new Set(canvases.map((canvas) => canvas.id)), [canvases]);
  const canvasIdKey = useMemo(() => canvases.map((canvas) => canvas.id).join(","), [canvases]);
  const { data: eventRefs = [] } = useQuery({
    queryKey: mergeConfidenceScoredRunRefsKey(organizationId, factoryId, orderId, canvasIdKey),
    queryFn: () => listMergeConfidenceScoredRunRefs(organizationId, factoryId, orderId, canvasIdSet),
    enabled: Boolean(organizationId && factoryId && orderId && taskKey && canvasIdKey),
    refetchOnWindowFocus: false,
  });
  const checkRefs = useMemo(() => mergeConfidenceRunRefsFromTask(canvasIdSet, { checks }), [canvasIdSet, checks]);
  const knownRefs = useMemo(() => dedupeRunRefs([...checkRefs, ...eventRefs]), [checkRefs, eventRefs]);
  const listQueries = useQueries({
    queries: taskKey
      ? canvases.map((canvas) => {
          const queryKey = [...mergeConfidenceRunsKey(organizationId, canvas.id, taskKey), "newest"] as const;
          return {
            queryKey,
            queryFn: () => {
              const current = queryClient.getQueryData<CanvasesCanvasRun[]>(queryKey);
              idsAtFetchStart.current.set(
                canvas.id,
                new Set((current ?? []).flatMap((run) => (run.id ? [run.id] : []))),
              );
              return listNewestTaskRuns(organizationId, canvas, pullRequests);
            },
            enabled: Boolean(organizationId && canvas.id),
            refetchOnWindowFocus: false,
            structuralSharing: (current: unknown, incoming: unknown) =>
              mergeMergeConfidenceRunSnapshots(
                current as CanvasesCanvasRun[] | undefined,
                incoming as CanvasesCanvasRun[],
                idsAtFetchStart.current.get(canvas.id),
              ),
          };
        })
      : [],
  });
  const describeQueries = useQueries({
    queries: taskKey
      ? knownRefs.map((ref) => ({
          queryKey: mergeConfidenceDescribedRunKey(organizationId, ref.canvasId, ref.runId),
          queryFn: () => describeMergeConfidenceRun(organizationId, ref.canvasId, ref.runId),
          enabled: Boolean(organizationId && ref.canvasId && ref.runId),
          refetchOnWindowFocus: false,
        }))
      : [],
  });
  const runs = useMemo(() => {
    const describedByCanvas = new Map<string, CanvasesCanvasRun[]>();
    knownRefs.forEach((ref, index) => {
      const run = describeQueries[index]?.data;
      if (!run?.id) {
        return;
      }
      const current = describedByCanvas.get(ref.canvasId) ?? [];
      current.push(run);
      describedByCanvas.set(ref.canvasId, current);
    });
    return canvases.flatMap((canvas, index) =>
      mergeConfidenceRunsForPullRequests(
        canvas,
        mergeListedAndDescribed(listQueries[index]?.data, describedByCanvas.get(canvas.id)),
        pullRequests,
      ),
    );
  }, [canvases, describeQueries, knownRefs, listQueries, pullRequests]);
  return {
    runs,
    canvasIds: taskKey ? canvases.map((canvas) => canvas.id) : [],
    taskKey,
  };
}

async function describeMergeConfidenceRun(
  organizationId: string,
  canvasId: string,
  runId: string,
): Promise<CanvasesCanvasRun> {
  const response = await canvasesDescribeRun(
    withOrganizationHeader({
      organizationId,
      path: { canvasId, runId },
    }),
  );
  const run = response.data?.run;
  if (!run?.id) {
    throw new Error("Merge confidence run not found");
  }
  return run;
}

function mergeListedAndDescribed(
  listed: CanvasesCanvasRun[] | undefined,
  described: CanvasesCanvasRun[] | undefined,
): CanvasesCanvasRun[] {
  return (described ?? []).reduce((current, run) => upsertMergeConfidenceCanvasRun(current, run), listed ?? []);
}

async function listNewestTaskRuns(
  organizationId: string,
  canvas: MergeConfidenceCanvas,
  pullRequests: FactoriesFactoryPullRequest[],
): Promise<CanvasesCanvasRun[]> {
  const response = await canvasesListRuns(
    withOrganizationHeader({
      organizationId,
      path: { canvasId: canvas.id },
      query: { limit: MERGE_CONFIDENCE_RUNS_PAGE_LIMIT },
    }),
  );
  return mergeConfidenceRunsForPullRequests(canvas, response.data?.runs ?? [], pullRequests).map((entry) => entry.run);
}

async function listMergeConfidenceScoredRunRefs(
  organizationId: string,
  factoryId: string,
  orderId: string,
  canvasIds: ReadonlySet<string>,
): Promise<MergeConfidenceRunRef[]> {
  const pages: Array<FactoriesListWorkOrderEventsResponse | undefined> = [];
  const refs: MergeConfidenceRunRef[] = [];
  let before: string | undefined;

  for (let page = 0; page < MERGE_CONFIDENCE_EVENTS_MAX_PAGES; page += 1) {
    const response = await factoriesListWorkOrderEvents(
      withOrganizationHeader({
        organizationId,
        path: { factoryId, orderId },
        query: {
          limit: WORK_ORDER_EVENTS_PAGE_LIMIT,
          ...(before ? { before } : {}),
        },
      }),
    );
    const data = response.data;
    pages.push(data);
    refs.push(...mergeConfidenceRunRefsFromTask(canvasIds, { events: data?.events }));
    const nextBefore = getWorkOrderEventsNextPageParam(data, pages);
    if (!nextBefore || nextBefore === before) {
      break;
    }
    before = nextBefore;
  }

  return dedupeRunRefs(refs);
}

function dedupeRunRefs(refs: MergeConfidenceRunRef[]): MergeConfidenceRunRef[] {
  const seen = new Set<string>();
  return refs.filter((ref) => {
    const key = `${ref.canvasId}:${ref.runId}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}
