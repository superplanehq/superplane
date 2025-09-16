import { Stage } from "../../store/types";
import { RunItem } from './RunItem';
import { useCallback, useMemo, useState, useEffect } from 'react';
import { useOrganizationUsersForCanvas } from '@/hooks/useCanvasData';
import { useInfiniteQuery } from '@tanstack/react-query';
import { superplaneListStageExecutions } from '@/api-client/sdk.gen';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';
import { ControlledTabs, Tab } from '@/components/Tabs/tabs';
import {
  formatDuration,
  getMinApprovedAt,
  getApprovalsNames,
  getDiscardedByName,
  mapExecutionOutputs,
  mapExecutionEventInputs,
  createUserDisplayNames
} from '../../utils/stageEventUtils';

interface ExecutionsTabProps {
  selectedStage: Stage;
  organizationId: string;
  canvasId: string;
  cancelStageExecution: (executionId: string, stageId: string) => Promise<void>;
}

export const ExecutionsTab = ({ selectedStage, organizationId, canvasId, cancelStageExecution }: ExecutionsTabProps) => {
  const { data: orgUsers = [] } = useOrganizationUsersForCanvas(organizationId);
  const [activeFilter, setActiveFilter] = useState('all');

  // Create query key that includes the filter
  const createQueryKey = (filter: string) => [
    'canvas', 'stageExecutions', canvasId, selectedStage.metadata?.id || '', filter
  ];

  // Determine which results to filter by based on active filter
  const getResultsFilter = useCallback((filter: string) => {
    switch (filter) {
      case 'passed':
        return ['RESULT_PASSED' as const];
      case 'failed':
        return ['RESULT_FAILED' as const];
      case 'cancelled':
        return ['RESULT_CANCELLED' as const];
      default:
        return undefined; // No filter for 'all'
    }
  }, []);

  // Fetch executions with server-side filtering
  const {
    data: executionsData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
    isLoading
  } = useInfiniteQuery({
    queryKey: createQueryKey(activeFilter),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListStageExecutions(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, stageIdOrName: selectedStage.metadata!.id! },
          query: {
            results: getResultsFilter(activeFilter),
            limit: 20,
            ...(pageParam && { before: pageParam })
          }
        })
      );
      return {
        executions: response.data?.executions || [],
        totalCount: response.data?.totalCount || 0,
        hasNextPage: response.data?.hasNextPage || false,
        lastTimestamp: response.data?.lastTimestamp
      };
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.hasNextPage ? lastPage.lastTimestamp : undefined,
    staleTime: 30 * 1000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
    enabled: !!canvasId && !!selectedStage.metadata?.id
  });

  const allExecutions = useMemo(() =>
    executionsData?.pages.flatMap(page => page.executions) || [],
    [executionsData?.pages]
  );

  const totalCount = useMemo(() =>
    executionsData?.pages[0]?.totalCount || 0,
    [executionsData?.pages]
  );

  // Refetch when selectedStage.executions changes
  useEffect(() => {
    refetch();
  }, [selectedStage.executions, refetch]);

  const userDisplayNames = useMemo(() => createUserDisplayNames(orgUsers), [orgUsers]);

  const filterTabs: Tab[] = [
    { id: 'all', label: 'All' },
    { id: 'passed', label: 'Passed' },
    { id: 'failed', label: 'Failed' },
    { id: 'cancelled', label: 'Cancelled' }
  ];

  const handleLoadMore = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <div className="p-6">
      <h3 className="font-bold text-left text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">
        Executions ({totalCount})
      </h3>

      <div className="mt-5 mb-6">
        <div className="flex items-center justify-end">
          <div className="flex-shrink-0">
            <ControlledTabs
              tabs={filterTabs}
              activeTab={activeFilter}
              onTabChange={setActiveFilter}
              variant="pills"
              buttonClasses="text-xs"
            />
          </div>
        </div>
      </div>

      <div className="mb-8 space-y-3">
        {isLoading ? (
          <div className="text-center py-8">
            <div className="inline-flex items-center justify-center w-16 h-16 mb-3">
              <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent"></div>
            </div>
            <p className="text-zinc-600 dark:text-zinc-400 text-sm">Loading executions...</p>
          </div>
        ) : allExecutions.length === 0 ? (
          <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
            <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3" aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>history</span>
            <p className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No executions available</p>
          </div>
        ) : (
          <>
            {allExecutions.map((execution) => {
              const sourceEvent = (execution.stageEvent as any)?.triggerEvent;

              return (
                <RunItem
                  key={execution.id!}
                  title={execution.stageEvent?.name || execution.id || 'Execution'}
                  runId={execution.id}
                  inputs={mapExecutionEventInputs(execution)}
                  outputs={mapExecutionOutputs(execution)}
                  state={execution.state || 'STATE_UNKNOWN'}
                  result={execution.result || 'RESULT_UNKNOWN'}
                  timestamp={execution.createdAt || new Date().toISOString()}
                  executionDuration={formatDuration(execution.startedAt || execution.createdAt, execution.finishedAt)}
                  approvedOn={getMinApprovedAt(execution)}
                  approvedBy={getApprovalsNames(execution, userDisplayNames)}
                  queuedOn={execution.stageEvent?.createdAt}
                  discardedOn={execution.stageEvent?.discardedAt}
                  discardedBy={getDiscardedByName(execution, userDisplayNames)}
                  eventId={sourceEvent?.id}
                  sourceEvent={sourceEvent}
                  onCancel={() => cancelStageExecution(execution.id!, selectedStage.metadata!.id!)}
                />
              );
            })}

            {hasNextPage && (
              <div className="flex justify-center pt-4">
                <button
                  onClick={handleLoadMore}
                  disabled={isFetchingNextPage}
                  className="text-blue-600 text-sm hover:text-blue-700 disabled:text-blue-400 underline transition-colors duration-200 disabled:cursor-not-allowed"
                >
                  {isFetchingNextPage ? 'Loading...' : 'Load More'}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};