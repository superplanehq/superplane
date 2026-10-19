import { useEffect, useMemo, useState } from "react";
import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useCanvasVersion, useDescribeRun, useEventExecutions, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { useCanvasRuntimeWebsocket } from "@/hooks/useCanvasWebsocket";
import { useFactoryWorkOrders } from "@/hooks/useFactoryData";
import { AutomationRunHistory } from "@/ui/AutomationRunHistory/AutomationRunHistory";
import { AutomationRunDetails } from "@/ui/AutomationRunHistory/AutomationRunDetails";
import { RunnerLogsTerminal } from "@/ui/Runs/RunInspectorRunnerLogs";
import { LiveLogStateNotice } from "@/ui/CanvasPage/RunnerLiveLogDialog/LiveLogStateNotice";
import { runDuration, toAutomationRun, toRunActivities } from "./runHistoryData";

const emptyOrders: FactoriesWorkOrder[] = [];

interface ImplementRunHistoryProps {
  canvasId: string;
  organizationId: string;
  factoryId: string;
  selectedRunId: string | null;
  onSelectRun: (id: string | null) => void;
}

export function ImplementRunHistory(props: ImplementRunHistoryProps) {
  const { canvasId, organizationId, factoryId, selectedRunId, onSelectRun } = props;
  useCanvasRuntimeWebsocket(canvasId, organizationId, true);
  const runsQuery = useInfiniteCanvasRuns(canvasId);
  const tasksQuery = useFactoryWorkOrders(organizationId, factoryId, { includeArchived: true });
  const runs = useMemo(
    () => runsQuery.data?.pages.flatMap((page) => page?.runs ?? []).filter((run) => run.id) ?? [],
    [runsQuery.data],
  );
  const orders = tasksQuery.data ?? emptyOrders;
  const mappedRuns = useMemo(() => runs.map((run) => toAutomationRun(run, orders, Date.now())), [runs, orders]);
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, []);
  return (
    <div className="min-h-0 flex-1 overflow-auto p-6" data-testid="implement-run-history">
      {tasksQuery.isError && (
        <DataError message="Task links could not be loaded." onRetry={() => void tasksQuery.refetch()} />
      )}
      {selectedRunId ? (
        <ImplementRunDetails
          key={selectedRunId}
          {...props}
          orders={orders}
          now={now}
          tasksLoading={tasksQuery.isLoading}
        />
      ) : (
        <>
          {runsQuery.hasNextPage && (
            <p className="mb-3 text-xs text-muted-foreground">Search and filters apply to loaded runs.</p>
          )}
          <AutomationRunHistory
            runs={mappedRuns.map((run, index) => ({ ...run, duration: runDuration(runs[index], now) }))}
            onSelectRun={(run) => onSelectRun(run.id)}
            loading={runsQuery.isLoading}
            tasksLoading={tasksQuery.isLoading}
            error={runsQuery.isError && !runsQuery.isFetchNextPageError ? "Could not load run history." : undefined}
            onRetry={() => void runsQuery.refetch()}
            onLoadMore={
              runsQuery.hasNextPage && !runsQuery.isFetchNextPageError && !runsQuery.isRefetching
                ? () => void runsQuery.fetchNextPage()
                : undefined
            }
            loadingMore={runsQuery.isFetchingNextPage}
          />
          {runsQuery.isFetchNextPageError && (
            <DataError message="More runs could not be loaded." onRetry={() => void runsQuery.fetchNextPage()} />
          )}
        </>
      )}
    </div>
  );
}

function ImplementRunDetails(
  props: ImplementRunHistoryProps & { orders: FactoriesWorkOrder[]; now: number; tasksLoading: boolean },
) {
  const { canvasId, organizationId, selectedRunId, onSelectRun } = props;
  const runQuery = useDescribeRun(canvasId, selectedRunId);
  const run = runQuery.data?.run;
  const executionsQuery = useEventExecutions(canvasId, run?.rootEvent?.id ?? null);
  const versionQuery = useCanvasVersion(organizationId, canvasId, run?.versionId ?? "");
  if (runQuery.isError || (!run && !runQuery.isLoading))
    return (
      <>
        <Button variant="ghost" onClick={() => onSelectRun(null)}>
          Run history
        </Button>
        <DataError message="Run details could not be loaded." onRetry={() => void runQuery.refetch()} />
      </>
    );
  if (!run || executionsQuery.isLoading || versionQuery.isLoading)
    return (
      <>
        <Button variant="ghost" onClick={() => onSelectRun(null)}>
          Run history
        </Button>
        <p role="status" className="py-8 text-sm text-muted-foreground">
          Loading run details…
        </p>
      </>
    );
  return (
    <LoadedImplementRunDetails {...props} run={run} executionsQuery={executionsQuery} versionQuery={versionQuery} />
  );
}

function LoadedImplementRunDetails({
  canvasId,
  organizationId,
  onSelectRun,
  orders,
  now,
  run,
  executionsQuery,
  versionQuery,
  tasksLoading,
}: ImplementRunHistoryProps & {
  orders: FactoriesWorkOrder[];
  now: number;
  tasksLoading: boolean;
  run: CanvasesCanvasRun;
  executionsQuery: ReturnType<typeof useEventExecutions>;
  versionQuery: ReturnType<typeof useCanvasVersion>;
}) {
  const executions = executionsQuery.data?.executions ?? [];
  const nodes = versionQuery.data?.spec?.nodes ?? [];
  const mappedRun = useMemo(() => toAutomationRun(run, orders, Date.now()), [run, orders]);
  const details = {
    ...mappedRun,
    duration: runDuration(run, now),
    activities: toRunActivities(run, executions, nodes),
  };
  return (
    <>
      {executionsQuery.isError && (
        <DataError message="Execution details could not be loaded." onRetry={() => void executionsQuery.refetch()} />
      )}
      {versionQuery.isError && (
        <DataError
          message="Historical step names could not be loaded. Step IDs are shown."
          onRetry={() => void versionQuery.refetch()}
        />
      )}
      <AutomationRunDetails
        run={details}
        tasksLoading={tasksLoading}
        onBack={() => onSelectRun(null)}
        onRetryLogs={() => void executionsQuery.refetch()}
        renderLogs={(activity) => {
          const execution = executions.find((item) => item.id === activity.id && executionBelongsToRun(item, run));
          if (execution?.id && execution.metadata?.runner_broker_task_id)
            return (
              <RunnerLogsTerminal key={execution.id} execution={execution} session={{ organizationId, canvasId }} />
            );
          if (executionsQuery.isError)
            return <p className="py-8 text-sm text-muted-foreground">Load execution details to check for logs.</p>;
          if (executionsQuery.isFetching) return <LiveLogStateNotice state="loading" />;
          return <p className="py-8 text-sm text-muted-foreground">No logs are available for this activity.</p>;
        }}
      />
    </>
  );
}

function executionBelongsToRun(execution: { id?: string; runId?: string }, run: CanvasesCanvasRun) {
  return execution.runId ? execution.runId === run.id : run.executions?.some((ref) => ref.id === execution.id);
}

function DataError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div role="alert" className="mb-4 flex items-center gap-3 text-sm">
      <span>{message}</span>
      <Button variant="outline" size="sm" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
