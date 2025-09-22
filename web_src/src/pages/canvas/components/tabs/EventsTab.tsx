import { Stage } from "../../store/types";
import { EventItem } from '../../../../components/EventItem';
import { RejectionItem } from '../RejectionItem';
import { useCallback, useMemo, useState, useEffect } from 'react';
import { useEventRejections, useStageEvents, useEvents } from '@/hooks/useCanvasData';
import { ControlledTabs, Tab } from '@/components/Tabs/tabs';
import { SuperplaneEvent, SuperplaneEventRejection, SuperplaneStageEvent } from '@/api-client';

interface EventsTabProps {
  selectedStage: Stage;
  organizationId: string;
  canvasId: string;
}

export const EventsTab = ({ selectedStage, canvasId }: EventsTabProps) => {
  const [activeFilter, setActiveFilter] = useState('received');

  // Fetch rejected events (using the new hook)
  const {
    data: rejectedEventsData,
    fetchNextPage: fetchNextRejected,
    hasNextPage: hasNextRejected,
    isFetchingNextPage: isFetchingNextRejected,
    refetch: refetchRejected,
    isLoading: isLoadingRejected
  } = useEventRejections(canvasId, 'TYPE_STAGE', selectedStage.metadata?.id || '');

  // Fetch emitted events (events emitted by this stage)
  const {
    data: emittedEventsData,
    fetchNextPage: fetchNextEmitted,
    hasNextPage: hasNextEmitted,
    isFetchingNextPage: isFetchingNextEmitted,
    refetch: refetchEmitted,
    isLoading: isLoadingEmitted
  } = useEvents(canvasId, 'EVENT_SOURCE_TYPE_STAGE', selectedStage.metadata?.id || '');

  // Fetch received events (stage events received by this stage)
  const {
    data: receivedEventsData,
    fetchNextPage: fetchNextReceived,
    hasNextPage: hasNextReceived,
    isFetchingNextPage: isFetchingNextReceived,
    refetch: refetchReceived,
    isLoading: isLoadingReceived
  } = useStageEvents(canvasId, selectedStage.metadata?.id || '', []);

  // Get all events based on active filter
  const allEvents = useMemo(() => {
    switch (activeFilter) {
      case 'rejected':
        return rejectedEventsData?.pages.flatMap(page => page.rejections) || [];
      case 'emitted':
        return emittedEventsData?.pages.flatMap(page => page.events) || [];
      case 'received':
        return receivedEventsData?.pages.flatMap(page => page.events.map((event: SuperplaneStageEvent) => event.triggerEvent).filter(Boolean)) || [];
      default:
        return [];
    }
  }, [activeFilter, rejectedEventsData?.pages, emittedEventsData?.pages, receivedEventsData?.pages]);

  const totalCount = useMemo(() => {
    switch (activeFilter) {
      case 'rejected':
        return rejectedEventsData?.pages[0]?.totalCount || 0;
      case 'emitted':
        return emittedEventsData?.pages[0]?.totalCount || 0;
      case 'received':
        return receivedEventsData?.pages[0]?.totalCount || 0;
      default:
        return 0;
    }
  }, [activeFilter, rejectedEventsData?.pages, emittedEventsData?.pages, receivedEventsData?.pages]);

  const isLoading = useMemo(() => {
    switch (activeFilter) {
      case 'rejected':
        return isLoadingRejected;
      case 'emitted':
        return isLoadingEmitted;
      case 'received':
        return isLoadingReceived;
      default:
        return false;
    }
  }, [activeFilter, isLoadingRejected, isLoadingEmitted, isLoadingReceived]);

  // Refetch when filter changes
  useEffect(() => {
    switch (activeFilter) {
      case 'rejected':
        refetchRejected();
        break;
      case 'emitted':
        refetchEmitted();
        break;
      case 'received':
        refetchReceived();
        break;
    }
  }, [activeFilter, refetchRejected, refetchEmitted, refetchReceived]);


  const filterTabs: Tab[] = [
    { id: 'received', label: 'Received' },
    { id: 'emitted', label: 'Emitted' },
    { id: 'rejected', label: 'Rejected' }
  ];

  const handleLoadMore = useCallback(() => {
    switch (activeFilter) {
      case 'rejected':
        if (hasNextRejected && !isFetchingNextRejected) {
          fetchNextRejected();
        }
        break;
      case 'emitted':
        if (hasNextEmitted && !isFetchingNextEmitted) {
          fetchNextEmitted();
        }
        break;
      case 'received':
        if (hasNextReceived && !isFetchingNextReceived) {
          fetchNextReceived();
        }
        break;
    }
  }, [activeFilter, hasNextRejected, isFetchingNextRejected, fetchNextRejected, hasNextEmitted, isFetchingNextEmitted, fetchNextEmitted, hasNextReceived, isFetchingNextReceived, fetchNextReceived]);

  const hasNextPage = useMemo(() => {
    switch (activeFilter) {
      case 'rejected':
        return hasNextRejected;
      case 'emitted':
        return hasNextEmitted;
      case 'received':
        return hasNextReceived;
      default:
        return false;
    }
  }, [activeFilter, hasNextRejected, hasNextEmitted, hasNextReceived]);

  const isFetchingNextPage = useMemo(() => {
    switch (activeFilter) {
      case 'rejected':
        return isFetchingNextRejected;
      case 'emitted':
        return isFetchingNextEmitted;
      case 'received':
        return isFetchingNextReceived;
      default:
        return false;
    }
  }, [activeFilter, isFetchingNextRejected, isFetchingNextEmitted, isFetchingNextReceived]);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h3 className="font-bold text-left text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">
          Events ({totalCount})
        </h3>
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

      <div className="mb-8 space-y-3">
        {isLoading ? (
          <div className="text-center py-8">
            <div className="inline-flex items-center justify-center w-16 h-16 mb-3">
              <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent"></div>
            </div>
            <p className="text-zinc-600 dark:text-zinc-400 text-sm">Loading events...</p>
          </div>
        ) : allEvents.length === 0 ? (
          <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
            <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3" aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>event</span>
            <p className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No events available</p>
          </div>
        ) : (
          <>
            {allEvents.map((event, index: number) => {
              const eventId = event?.id || `event-${index}`;

              // Use RejectionItem for rejected events
              if (activeFilter === 'rejected') {
                return (
                  <RejectionItem
                    key={eventId}
                    rejection={event as SuperplaneEventRejection}
                  />
                );
              }

              // Use EventItem for other event types
              const sourceEvent = event as SuperplaneEvent;
              const plainEventPayload = sourceEvent?.raw;
              const plainEventHeaders = sourceEvent?.headers;

              return (
                <EventItem
                  key={eventId}
                  eventId={eventId!}
                  timestamp={sourceEvent?.receivedAt!}
                  state={sourceEvent?.state}
                  stateReason={sourceEvent?.stateReason}
                  stateMessage={sourceEvent?.stateMessage}
                  eventType={sourceEvent?.type}
                  sourceName={sourceEvent?.sourceName}
                  headers={plainEventHeaders}
                  payload={plainEventPayload}
                  showStateLabel={false}
                />
              );
            })}

            {hasNextPage && (
              <div className="flex justify-center pt-4">
                {isFetchingNextPage ? (
                  <div className="inline-flex items-center justify-center">
                    <div className="animate-spin rounded-full h-6 w-6 border-2 border-blue-600 border-t-transparent mr-2"></div>
                    <span className="text-zinc-600 dark:text-zinc-400 text-sm">Loading more...</span>
                  </div>
                ) : (
                  <button
                    onClick={handleLoadMore}
                    className="text-blue-600 text-sm hover:text-blue-700 underline transition-colors duration-200"
                  >
                    Load More
                  </button>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};