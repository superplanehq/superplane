import { canvasesListRuns } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { backlogAnalysisRuns, type BacklogAnalysisRun } from "@/pages/factories/lib/backlogAnalysis";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "@/pages/factories/lib/customAutomation";
import { useQueries } from "@tanstack/react-query";

/** Load custom task-entry runs only while the task console is open. */
export function useBacklogCustomRuns(
  organizationId: string,
  workOrderId: string,
  apps: Array<{ id?: string; columnKey?: string; templateId?: string }>,
): BacklogAnalysisRun[] {
  const queries = useQueries({
    queries: apps
      .filter((app) => app.id && app.columnKey === "backlog" && app.templateId === CUSTOM_AUTOMATION_TEMPLATE_ID)
      .map((app) => ({
        queryKey: ["backlog-custom-runs", organizationId, app.id, workOrderId],
        queryFn: async ({ signal }: { signal: AbortSignal }) => {
          const canvasId = app.id!;
          const runs: BacklogAnalysisRun[] = [];
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
            runs.push(
              ...backlogAnalysisRuns(canvasId, response.data?.runs ?? []).filter(
                (entry) => entry.workOrderId === workOrderId,
              ),
            );
            const next = response.data?.lastTimestamp;
            before = response.data?.hasNextPage && next !== before ? next : undefined;
          } while (before);
          return runs;
        },
        enabled: Boolean(organizationId && workOrderId),
        refetchInterval: 5000,
        refetchOnWindowFocus: false,
      })),
  });
  return queries.flatMap((query) => query.data ?? []);
}
