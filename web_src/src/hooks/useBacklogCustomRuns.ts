import { canvasesDescribeRun, canvasesListRuns, type CanvasesCanvasRun } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { backlogAnalysisRuns, type BacklogAnalysisRun } from "@/pages/factories/lib/backlogAnalysis";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "@/pages/factories/lib/customAutomation";
import { isActiveCanvasRun } from "@/pages/factories/lib/workOrderPullRequest";
import { useQueries, useQueryClient } from "@tanstack/react-query";

type CustomRunHistory = { runs: BacklogAnalysisRun[]; latestRunId?: string };

async function loadRecentRuns(
  organizationId: string,
  canvasId: string,
  signal: AbortSignal,
  cachedLatestRunId?: string,
) {
  const recentRuns: CanvasesCanvasRun[] = [];
  let latestRunId: string | undefined;
  let before: string | undefined;
  do {
    const response = await canvasesListRuns(
      withOrganizationHeader({
        organizationId,
        path: { canvasId },
        query: { limit: 200, ...(before ? { before } : {}) },
        signal,
        throwOnError: true,
      }),
    );
    const page = response.data?.runs ?? [];
    if (!before) latestRunId = page[0]?.id;
    recentRuns.push(...page);
    if (cachedLatestRunId && page.some((run) => run.id === cachedLatestRunId)) break;
    const next = response.data?.lastTimestamp;
    before = response.data?.hasNextPage && next !== before ? next : undefined;
  } while (before);
  return { recentRuns, latestRunId };
}

async function refreshHistory(
  organizationId: string,
  canvasId: string,
  workOrderId: string,
  signal: AbortSignal,
  cached?: CustomRunHistory,
): Promise<CustomRunHistory> {
  const { recentRuns, latestRunId } = await loadRecentRuns(organizationId, canvasId, signal, cached?.latestRunId);
  // Older completed runs are immutable. Only task-linked active runs need details.
  const activeRuns = (cached?.runs ?? []).filter(
    (entry) =>
      (isActiveCanvasRun(entry.run) || entry.run.state === "STATE_CANCELLING") &&
      !recentRuns.some((run) => run.id === entry.run.id),
  );
  const updatedRuns = await Promise.all(
    activeRuns.map(async (entry) => {
      const response = await canvasesDescribeRun(
        withOrganizationHeader({
          organizationId,
          path: { canvasId, runId: entry.run.id! },
          signal,
          throwOnError: true,
        }),
      );
      return response.data?.run ?? entry.run;
    }),
  );
  const byId = new Map((cached?.runs ?? []).map((entry) => [entry.run.id, entry.run]));
  for (const entry of backlogAnalysisRuns(canvasId, [...recentRuns, ...updatedRuns])) {
    if (entry.workOrderId === workOrderId) byId.set(entry.run.id, entry.run);
  }
  return { runs: backlogAnalysisRuns(canvasId, [...byId.values()]), latestRunId };
}

/** Load custom task-entry runs only while the task console is open. */
export function useBacklogCustomRuns(
  organizationId: string,
  workOrderId: string,
  apps: Array<{ id?: string; columnKey?: string; templateId?: string }>,
) {
  const queryClient = useQueryClient();
  const queries = useQueries({
    queries: apps
      .filter((app) => app.id && app.columnKey === "backlog" && app.templateId === CUSTOM_AUTOMATION_TEMPLATE_ID)
      .map((app) => ({
        queryKey: ["backlog-custom-runs", organizationId, app.id, workOrderId],
        queryFn: ({ signal }: { signal: AbortSignal }) =>
          refreshHistory(
            organizationId,
            app.id!,
            workOrderId,
            signal,
            queryClient.getQueryData<CustomRunHistory>(["backlog-custom-runs", organizationId, app.id, workOrderId]),
          ),
        enabled: Boolean(organizationId && workOrderId),
        refetchInterval: 5000,
        refetchOnWindowFocus: false,
        retry: false,
      })),
  });
  return {
    runs: queries.flatMap((query) => query.data?.runs ?? []),
    isError: queries.some((query) => query.isError),
    isFetching: queries.some((query) => query.isFetching),
    retry: () => {
      for (const query of queries) void query.refetch();
    },
  };
}
