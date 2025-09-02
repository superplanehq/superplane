import { useState, useMemo, useEffect } from "react";
import { SuperplaneEventSource, SuperplaneEvent } from "@/api-client";
import { useResizableSidebar } from "../hooks/useResizableSidebar";
import { SidebarHeader } from "./SidebarHeader";
import { ResizeHandle } from "./ResizeHandle";
import { MaterialSymbol } from "@/components/MaterialSymbol/material-symbol";
import { EventItem } from "./EventItem";
import { useIntegrations } from "../hooks/useIntegrations";
import { useCanvasStore } from "../store/canvasStore";
import { useEventSourceEvents } from "@/hooks/useCanvasData";
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';
import { SidebarTabs } from './SidebarTabs';

const EventSourceImageMap = {
  'webhook': <MaterialSymbol className='-mt-1 -mb-1' name="webhook" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />
}

type TabType = 'history' | 'settings';

interface EventSourceSidebarProps {
  selectedEventSource: SuperplaneEventSource & {
    events?: Array<SuperplaneEvent>;
    eventSourceType?: string;
  };
  onClose: () => void;
}

export const EventSourceSidebar = ({ selectedEventSource, onClose }: EventSourceSidebarProps) => {
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(450);
  const [activeTab, setActiveTab] = useState<TabType>('history');
  const canvasId = useCanvasStore(state => state.canvasId) || '';

  const { data: canvasIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");

  const {
    data: eventsData,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading: eventsLoading,
    refetch: refetchEvents
  } = useEventSourceEvents(canvasId, selectedEventSource.metadata?.id || '');

  // Flatten all pages into a single array
  const allEvents = useMemo(() => {
    return eventsData?.pages.flatMap(page => page.events) || [];
  }, [eventsData]);

  useEffect(() => {
    if ((selectedEventSource?.events?.length || 0) > 0 && allEvents.length > 0) {
      const latestBulkEvent = selectedEventSource?.events?.[0];
      const latestQueryEvent = allEvents[0];

      // If bulk has a newer event or there are pending events, refetch the query
      if (latestBulkEvent?.receivedAt && latestQueryEvent?.receivedAt) {
        const bulkTime = new Date(latestBulkEvent.receivedAt);
        const queryTime = new Date(latestQueryEvent.receivedAt);
        const hasPendingEvents = allEvents.some(event => event.state === 'STATE_PENDING');
        
        if (bulkTime > queryTime || hasPendingEvents) {
          refetchEvents();
        }
      }
    }
  }, [selectedEventSource?.events, allEvents, refetchEvents]);

  const eventSourceType = useMemo(() => {
    if (selectedEventSource.eventSourceType)
      return selectedEventSource.eventSourceType;

    const integrationName = selectedEventSource.spec?.integration?.name;
    const integration = canvasIntegrations.find(integration => integration.metadata?.name === integrationName);
    if (integration && integration.spec?.type) {
      return integration.spec?.type;
    }
    return "webhook";
  }, [canvasIntegrations, selectedEventSource.eventSourceType, selectedEventSource.spec?.integration?.name]);


  return (
    <aside
      ref={sidebarRef}
      className={`fixed top-[2.6rem] right-0 z-10 bg-white dark:bg-zinc-900 flex flex-col w-[250px] ${isDragging.current ? '' : 'transition-all duration-200'
        }`}
      style={{
        width: width,
        height: 'calc(100vh - 3rem)',
        boxShadow: 'rgba(0,0,0,0.07) -2px 0 12px',
      }}
    >
      <SidebarHeader
        image={EventSourceImageMap[eventSourceType as keyof typeof EventSourceImageMap]}
        stageName={selectedEventSource.metadata?.name || ''}
        onClose={onClose}
      />

      {/* Tab Navigation */}
      <SidebarTabs
        tabs={[
          { key: 'history', label: 'History' },
          { key: 'settings', label: 'Settings' }
        ]}
        activeTab={activeTab}
        onTabChange={(tab) => setActiveTab(tab as TabType)}
      />

      <div className="flex-1 overflow-y-auto">
        {activeTab === 'history' && (
          <div className="p-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-zinc-100 uppercase tracking-wide">
                Event History ({allEvents.length})
              </h3>
            </div>

            <div className="space-y-2">
              {eventsLoading && allEvents.length === 0 ? (
                <div className="text-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                  <p className="text-sm text-gray-500 mt-2">Loading events...</p>
                </div>
              ) : allEvents.length > 0 ? (
                allEvents.map((event) => (
                  <EventItem
                    key={event.id}
                    eventId={event.id!}
                    timestamp={event.receivedAt!}
                    state={event.state}
                    eventType={event.type}
                    sourceName={event.sourceName}
                    headers={event.headers}
                    payload={event.raw}
                  />
                ))
              ) : (
                <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
                  <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3 " aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>inbox</span>
                  <p data-slot="text" className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No events received</p>
                </div>
              )}
            </div>

            {hasNextPage && (
              <div className="mt-4 text-center">
                <button
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="text-sm text-blue-600 dark:text-blue-400 hover:underline disabled:text-gray-400 disabled:cursor-not-allowed"
                >
                  {isFetchingNextPage ? 'Loading...' : 'Load more events'}
                </button>
              </div>
            )}
          </div>
        )}

        {activeTab === 'settings' && (
          <div className="p-4 text-left">
            <div className="space-y-6">
              <div className="space-y-4">
                {eventSourceType !== 'webhook' ? (
                  <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-4 bg-white dark:bg-zinc-900">
                    <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide mb-3">
                      {eventSourceType.charAt(0).toUpperCase() + eventSourceType.slice(1)} Configuration
                    </div>
                    <div className="space-y-3">
                      <div>
                        <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Integration</div>
                        <div className="text-sm text-gray-900 dark:text-zinc-200">{selectedEventSource.spec?.integration?.name || `${eventSourceType.charAt(0).toUpperCase() + eventSourceType.slice(1)} Integration`}</div>
                      </div>
                      <div>
                        <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Project</div>
                        <div className="text-sm text-gray-900 dark:text-zinc-200">{selectedEventSource.metadata?.name || 'Unknown Project'}</div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-4 bg-white dark:bg-zinc-900">
                    <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide mb-3">
                      Integration Configuration
                    </div>
                    <div className="space-y-3">
                      <div>
                        <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Integration</div>
                        <div className="text-sm text-gray-900 dark:text-zinc-200">Direct Webhook</div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>

      <ResizeHandle
        onMouseDown={handleMouseDown}
        onMouseEnter={() => {
          if (!isDragging.current && sidebarRef.current)
            sidebarRef.current.style.cursor = 'ew-resize';
        }}
        onMouseLeave={() => {
          if (!isDragging.current && sidebarRef.current)
            sidebarRef.current.style.cursor = 'default';
        }}
      />
    </aside>
  );
};