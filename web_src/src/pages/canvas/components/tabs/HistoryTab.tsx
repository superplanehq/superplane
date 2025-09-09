import { ExecutionWithEvent, StageWithEventQueue } from "../../store/types";
import { SuperplaneStageEvent, SuperplaneEvent, SuperplaneExecution } from "@/api-client";
import MessageItem from '../MessageItem';
import { RunItem } from './RunItem';
import { useCallback, useMemo, useState, useEffect } from 'react';
import { useOrganizationUsersForCanvas, useStageQueueEvents, useStageEvents } from '@/hooks/useCanvasData';
import { ControlledTabs, Tab } from '@/components/Tabs/tabs';
import {
  formatDuration,
  getMinApprovedAt,
  getApprovalsNames,
  getCancelledByName,
  mapExecutionOutputs,
  mapExecutionEventInputs,
  createUserDisplayNames
} from '../../utils/stageEventUtils';

interface HistoryTabProps {
  selectedStage: StageWithEventQueue;
  organizationId: string;
  canvasId: string;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  cancelStageEvent: (stageEventId: string, stageId: string) => void;
  connectionEventsById?: Record<string, SuperplaneEvent>;
  isFetchingNextConnectedEvents: boolean;
  fetchNextConnectedEvents: () => void;
}

export const HistoryTab = ({ selectedStage, organizationId, canvasId, approveStageEvent, cancelStageEvent, connectionEventsById, fetchNextConnectedEvents }: HistoryTabProps) => {
  // Create a unified timeline by merging executions, stage events, and discarded events
  type TimelineItem = {
    type: 'execution' | 'stage_event' | 'discarded_event';
    timestamp: string;
    data: ExecutionWithEvent | SuperplaneStageEvent | SuperplaneEvent;
  };

  const { data: orgUsers = [] } = useOrganizationUsersForCanvas(organizationId);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeFilter, setActiveFilter] = useState('runs');
  const [timelineLimit, setTimelineLimit] = useState(20);

  const discardedEventsCount = useMemo(() => {
    return Object.values(connectionEventsById || {}).filter(event => event.state === 'STATE_DISCARDED').length || 0;
  }, [connectionEventsById]);

  const {
    data: queueEventsData,
    fetchNextPage: fetchNextQueuePage,
    hasNextPage: hasNextQueuePage,
    isFetchingNextPage: isFetchingNextQueuePage,
    refetch: refetchQueueEvents
  } = useStageQueueEvents(canvasId, selectedStage.metadata!.id!, ['STATE_PENDING', 'STATE_WAITING', 'STATE_PROCESSED'], ['STATE_REASON_APPROVAL', 'STATE_REASON_TIME_WINDOW', "STATE_REASON_CANCELLED", "STATE_REASON_UNKNOWN"], [], undefined, "EXECUTION_FILTER_WITHOUT_EXECUTION");

  const {
    data: runQueueEventsData,
    fetchNextPage: fetchNextRunQueuePage,
    hasNextPage: hasNextRunQueuePage,
    isFetchingNextPage: isFetchingNextRunQueuePage,
    refetch: refetchRunQueueEvents
  } = useStageQueueEvents(canvasId, selectedStage.metadata!.id!, ['STATE_PROCESSED', 'STATE_WAITING', "STATE_PENDING"], ['STATE_REASON_EXECUTION', "STATE_REASON_UNKNOWN"], ["STATE_CANCELLED", "STATE_FINISHED", "STATE_FINISHED"], undefined, "EXECUTION_FILTER_WITH_EXECUTION");

  const {
    data: stagePlainEventsData,
    fetchNextPage: fetchNextStagePlainEventsPage,
    hasNextPage: hasNextStagePlainEventsPage,
    isFetchingNextPage: isFetchingNextStagePlainEventsPage,
    refetch: refetchStagePlainEvents
  } = useStageEvents(canvasId, selectedStage.metadata!.id!);

  const eventsByExecutionId = useMemo(() => {
    const emittedEventsById: Record<string, SuperplaneEvent> = {};

    stagePlainEventsData?.pages.flatMap(page => page.events)?.forEach(event => {
      const execution = event.raw?.execution as SuperplaneExecution;
      if (execution?.id) {
        emittedEventsById[execution.id || ''] = event;
      }
    })

    return emittedEventsById;
  }, [stagePlainEventsData?.pages]);


  const allExecutionsData = useMemo(() => (runQueueEventsData?.pages.flatMap(page => page.events) || [])
    .filter(event => event.execution)
    .flatMap(event => ({ ...event.execution, event }) as ExecutionWithEvent)
    .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()), [runQueueEventsData]);

  const discardedEvents = useMemo(() => {
    return Object.values(connectionEventsById || {}).filter(event => event.state === 'STATE_DISCARDED')
      .filter(plainEvent => (plainEvent?.state === 'STATE_DISCARDED' && plainEvent?.receivedAt && plainEvent.sourceId !== selectedStage.metadata?.id));
  }, [connectionEventsById, selectedStage.metadata?.id]);

  // Refetch queries when selectedStage.events or .queue changes
  // That means there was a new event or a new queue event since
  // those small arrays are updated near real time
  useEffect(() => {
    refetchQueueEvents();
    refetchRunQueueEvents();
    refetchStagePlainEvents();
  }, [selectedStage.events, selectedStage.queue, refetchQueueEvents, refetchRunQueueEvents, refetchStagePlainEvents]);

  const createTimeline = (): TimelineItem[] => {
    const items: TimelineItem[] = [];

    allExecutionsData.forEach(execution => {
      if (execution?.createdAt) {
        items.push({
          type: 'execution',
          timestamp: execution.createdAt,
          data: execution
        });
      }
    });

    const allStageEvents = (queueEventsData?.pages.flatMap(page => page.events) || [])

    allStageEvents.forEach(event => {
      if (event?.createdAt) {
        items.push({
          type: 'stage_event',
          timestamp: event.createdAt,
          data: event
        });
      }
    });

    if (connectionEventsById) {
      discardedEvents.forEach(plainEvent => {
        items.push({
          type: 'discarded_event',
          timestamp: plainEvent.receivedAt!,
          data: plainEvent
        });
      });
    }

    return items.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  };

  const userDisplayNames = useMemo(() => createUserDisplayNames(orgUsers), [orgUsers]);

  const filterTabs: Tab[] = [
    { id: 'all', label: 'All' },
    { id: 'runs', label: 'Runs' },
    { id: 'queue', label: 'Queue' },
    { id: 'events', label: 'Events' }
  ];

  const timeline = createTimeline();

  const filteredTimeline = useMemo(() => {
    let filtered = timeline;

    // Filter by type
    if (activeFilter !== 'all') {
      if (activeFilter === 'runs') {
        filtered = filtered.filter(item => item.type === 'execution');
      } else if (activeFilter === 'queue') {
        filtered = filtered.filter(item => item.type === 'stage_event');
      } else if (activeFilter === 'events') {
        filtered = filtered.filter(item => item.type === 'discarded_event');
      }
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(item => {
        if (item.type === 'execution') {
          const execution = item.data as ExecutionWithEvent;
          return (
            execution.event?.name?.toLowerCase().includes(query) ||
            execution.id?.toLowerCase().includes(query) ||
            execution.state?.toLowerCase().includes(query) ||
            execution.result?.toLowerCase().includes(query)
          );
        } else if (item.type === 'stage_event') {
          const stageEvent = item.data as SuperplaneStageEvent;
          return (
            stageEvent.name?.toLowerCase().includes(query) ||
            stageEvent.id?.toLowerCase().includes(query) ||
            stageEvent.state?.toLowerCase().includes(query)
          );
        } else if (item.type === 'discarded_event') {
          const plainEvent = item.data as SuperplaneEvent;
          return (
            plainEvent.id?.toLowerCase().includes(query) ||
            plainEvent.state?.toLowerCase().includes(query)
          );
        }
        return false;
      });
    }

    return filtered.slice(0, timelineLimit);
  }, [timeline, activeFilter, searchQuery, timelineLimit]);

  const handleLoadMore = useCallback(() => {
    const newLimit = timelineLimit + 20;
    setTimelineLimit(newLimit);

    // Fetch next pages if we need more data
    if (hasNextQueuePage && !isFetchingNextQueuePage) {
      fetchNextQueuePage();
    }
    if (hasNextRunQueuePage && !isFetchingNextRunQueuePage) {
      fetchNextRunQueuePage();
    }

    if (hasNextStagePlainEventsPage && !isFetchingNextStagePlainEventsPage) {
      fetchNextStagePlainEventsPage();
    }

    // Fetch next connected events
    fetchNextConnectedEvents();
  }, [
    timelineLimit,
    hasNextQueuePage,
    isFetchingNextQueuePage,
    isFetchingNextRunQueuePage,
    fetchNextConnectedEvents,
    fetchNextQueuePage,
    fetchNextRunQueuePage,
    fetchNextStagePlainEventsPage,
    isFetchingNextStagePlainEventsPage,
    hasNextRunQueuePage,
    hasNextStagePlainEventsPage,
  ]);

  const totalCount = useMemo(() => {
    const queueEventsTotalCount = queueEventsData?.pages.at(-1)?.totalCount || 0;
    const runQueueEventsTotalCount = runQueueEventsData?.pages.at(-1)?.totalCount || 0;
    const discardedEventsTotalCount = discardedEvents.length;

    if (activeFilter === 'all') {
      return runQueueEventsTotalCount + queueEventsTotalCount + discardedEventsTotalCount;
    }

    if (activeFilter === 'runs') {
      return runQueueEventsTotalCount;
    }

    if (activeFilter === 'queue') {
      return queueEventsTotalCount;
    }

    if (activeFilter === 'events') {
      return discardedEventsTotalCount;
    }

    return 0;
  }, [queueEventsData?.pages, runQueueEventsData?.pages, discardedEvents, activeFilter]);


  const hasMoreItems = useMemo(() => {
    if (activeFilter === 'all') {
      return totalCount > timelineLimit;
    }

    if (activeFilter === 'runs') {
      return hasNextRunQueuePage;
    }

    if (activeFilter === 'queue') {
      return hasNextQueuePage;
    }

    if (activeFilter === 'events') {
      return discardedEventsCount >= timelineLimit;
    }

    return false;
  }, [hasNextQueuePage, hasNextRunQueuePage, activeFilter, timelineLimit, totalCount, discardedEventsCount]);

  const isLoadingMore = isFetchingNextQueuePage || isFetchingNextRunQueuePage || isFetchingNextStagePlainEventsPage;

  // Check if we're still fetching the first page of any of the data sources
  const isLoadingInitial = !queueEventsData || !runQueueEventsData || !stagePlainEventsData;

  return (
    <div className="p-6">
      <h3 className="font-bold text-left text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">
        History ({totalCount} items)
      </h3>

      <div className="mt-5 mb-6">
        <div className="flex items-center gap-2">
          <div className="flex-grow">
            <div className="relative">
              <span className="material-symbols-outlined absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-500 dark:text-zinc-400 text-base pointer-events-none text-base!">
                search
              </span>
              <input
                type="search"
                placeholder="Search history..."
                className="w-full pl-8 pr-4 py-2 text-sm border border-zinc-300 dark:border-zinc-600 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-white placeholder:text-zinc-500 dark:placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>
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
        {isLoadingInitial ? (
          <div className="flex justify-center items-center py-16">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-400"></div>
            <p className="ml-3 text-gray-500">Loading...</p>
          </div>
        ) : filteredTimeline.length === 0 ? (
          <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
            <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3" aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>history</span>
            <p className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No history available</p>
          </div>
        ) : (
          <>
            {filteredTimeline.map((item) => {
              if (item.type === 'execution') {
                const execution = item.data as ExecutionWithEvent;
                const sourceEvent = connectionEventsById?.[execution.event.eventId || ''];
                const emmitedEvent = eventsByExecutionId?.[execution.id || ''];

                return (
                  <RunItem
                    key={execution.id!}
                    title={execution.event.name || execution.id || 'Execution'}
                    runId={execution.id}
                    inputs={mapExecutionEventInputs(execution)}
                    outputs={mapExecutionOutputs(execution)}
                    state={execution.state || 'STATE_UNKNOWN'}
                    result={execution.result || 'RESULT_UNKNOWN'}
                    timestamp={execution.createdAt || new Date().toISOString()}
                    executionDuration={formatDuration(execution.startedAt || execution.createdAt, execution.finishedAt)}
                    approvedOn={getMinApprovedAt(execution)}
                    approvedBy={getApprovalsNames(execution, userDisplayNames)}
                    queuedOn={execution.event.createdAt}
                    cancelledOn={execution.event.cancelledAt}
                    cancelledBy={getCancelledByName(execution, userDisplayNames)}
                    eventId={sourceEvent?.id}
                    sourceEvent={sourceEvent}
                    emmitedEvent={emmitedEvent}
                    onCancel={() => cancelStageEvent(execution.event.id!, selectedStage.metadata!.id!)}
                  />
                );
              }
              if (item.type === 'stage_event') {
                const stageEvent = item.data as SuperplaneStageEvent;
                const sourceEvent = connectionEventsById?.[stageEvent.eventId || ''];
                const plainEventPayload = connectionEventsById?.[stageEvent.eventId || '']?.raw;
                const plainEventHeaders = connectionEventsById?.[stageEvent.eventId || '']?.headers;
                const approvalAndCancelledData = { event: stageEvent } as ExecutionWithEvent;

                return (
                  <MessageItem
                    key={stageEvent.id}
                    event={stageEvent}
                    selectedStage={selectedStage}
                    executionRunning={false}
                    onApprove={stageEvent.state === 'STATE_WAITING' ? (eventId) => approveStageEvent(eventId, selectedStage.metadata!.id!) : undefined}
                    onCancel={(eventId) => cancelStageEvent(eventId, selectedStage.metadata!.id!)}
                    plainEventPayload={plainEventPayload}
                    plainEventHeaders={plainEventHeaders}
                    sourceEvent={sourceEvent}
                    approvedOn={getMinApprovedAt(approvalAndCancelledData)}
                    approvedBy={getApprovalsNames(approvalAndCancelledData, userDisplayNames)}
                    cancelledOn={stageEvent.cancelledAt}
                    cancelledBy={getCancelledByName(approvalAndCancelledData, userDisplayNames)}
                  />
                );
              }
              if (item.type === 'discarded_event') {
                const plainEvent = item.data as SuperplaneEvent;
                // Get payload from the plain event's raw data
                const plainEventPayload = plainEvent.raw;
                const plainEventHeaders = plainEvent.headers

                return (
                  <MessageItem
                    key={plainEvent.id}
                    event={plainEvent}
                    executionRunning={false}
                    plainEventPayload={plainEventPayload}
                    plainEventHeaders={plainEventHeaders}
                  />
                );
              }
              return null;
            })}

            {hasMoreItems && (
              <div className="flex justify-center pt-4">
                <button
                  onClick={handleLoadMore}
                  disabled={isLoadingMore}
                  className="text-blue-600 text-sm hover:text-blue-700 disabled:text-blue-400 underline transition-colors duration-200 disabled:cursor-not-allowed"
                >
                  {isLoadingMore ? 'Loading...' : 'Load More'}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};