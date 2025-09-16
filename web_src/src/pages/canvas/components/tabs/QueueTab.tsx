import { Stage } from "../../store/types";
import MessageItem from '../MessageItem';
import { useCallback, useMemo, useState, useEffect } from 'react';
import { useOrganizationUsersForCanvas } from '@/hooks/useCanvasData';
import { useInfiniteQuery } from '@tanstack/react-query';
import { superplaneListStageEvents } from '@/api-client/sdk.gen';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';
import { ControlledTabs, Tab } from '@/components/Tabs/tabs';
import {
  getMinApprovedAt,
  getApprovalsNames,
  getDiscardedByName,
  createUserDisplayNames
} from '../../utils/stageEventUtils';

interface QueueTabProps {
  selectedStage: Stage;
  organizationId: string;
  canvasId: string;
  approveStageEvent: (stageEventId: string, stageId: string) => Promise<void>;
  discardStageEvent: (stageEventId: string, stageId: string) => Promise<void>;
}

export const QueueTab = ({ selectedStage, organizationId, canvasId, approveStageEvent, discardStageEvent }: QueueTabProps) => {
  const { data: orgUsers = [] } = useOrganizationUsersForCanvas(organizationId);
  const [activeFilter, setActiveFilter] = useState('all');

  // Create query key that includes the filter
  const createQueryKey = (filter: string) => [
    'canvas', 'stageEvents', canvasId, selectedStage.metadata?.id || '', filter
  ];

  // Determine which states to filter by based on active filter
  const getStatesFilter = useCallback((filter: string) => {
    switch (filter) {
      case 'pending':
        return ['STATE_PENDING' as const];
      case 'waiting':
        return ['STATE_WAITING' as const];
      case 'discarded':
        return ['STATE_DISCARDED' as const];
      default:
        return ['STATE_PENDING' as const, 'STATE_WAITING' as const, 'STATE_DISCARDED' as const]; // All queue states
    }
  }, []);

  // Fetch stage events with server-side filtering
  const {
    data: eventsData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
    isLoading
  } = useInfiniteQuery({
    queryKey: createQueryKey(activeFilter),
    queryFn: async ({ pageParam }) => {
      const response = await superplaneListStageEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId, stageIdOrName: selectedStage.metadata!.id! },
          query: {
            states: getStatesFilter(activeFilter),
            limit: 20,
            ...(pageParam && { before: pageParam })
          }
        })
      );
      return {
        events: response.data?.events || [],
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

  const allEvents = useMemo(() =>
    eventsData?.pages.flatMap(page => page.events) || [],
    [eventsData?.pages]
  );

  const totalCount = useMemo(() =>
    eventsData?.pages[0]?.totalCount || 0,
    [eventsData?.pages]
  );

  // Refetch when selectedStage.queue changes
  useEffect(() => {
    refetch();
  }, [selectedStage.queue, refetch]);

  const userDisplayNames = useMemo(() => createUserDisplayNames(orgUsers), [orgUsers]);

  const filterTabs: Tab[] = [
    { id: 'all', label: 'All' },
    { id: 'pending', label: 'Pending' },
    { id: 'waiting', label: 'Waiting' },
    { id: 'discarded', label: 'Discarded' }
  ];

  const handleLoadMore = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <div className="p-6">
      <h3 className="font-bold text-left text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">
        Queue ({totalCount})
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
            <p className="text-zinc-600 dark:text-zinc-400 text-sm">Loading queue...</p>
          </div>
        ) : allEvents.length === 0 ? (
          <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
            <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3" aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>queue</span>
            <p className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No queue items available</p>
          </div>
        ) : (
          <>
            {allEvents.map((stageEvent) => {
              const sourceEvent = stageEvent.triggerEvent;
              const plainEventPayload = sourceEvent?.raw;
              const plainEventHeaders = sourceEvent?.headers;
              const approvalAndCancelledData = { event: stageEvent } as any;

              return (
                <MessageItem
                  key={stageEvent.id}
                  event={stageEvent}
                  selectedStage={selectedStage}
                  executionRunning={false}
                  onApprove={stageEvent.state === 'STATE_WAITING' ? (eventId) => approveStageEvent(eventId, selectedStage.metadata!.id!) : undefined}
                  onCancel={(eventId) => discardStageEvent(eventId, selectedStage.metadata!.id!)}
                  plainEventPayload={plainEventPayload}
                  plainEventHeaders={plainEventHeaders}
                  sourceEvent={sourceEvent}
                  approvedOn={getMinApprovedAt(approvalAndCancelledData)}
                  approvedBy={getApprovalsNames(approvalAndCancelledData, userDisplayNames)}
                  discardedOn={stageEvent.discardedAt}
                  discardedBy={getDiscardedByName(approvalAndCancelledData, userDisplayNames)}
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