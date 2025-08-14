import { useState, useMemo } from "react";
import { SuperplaneEventSource, SuperplaneEvent } from "@/api-client";
import { useResizableSidebar } from "../hooks/useResizableSidebar";
import { SidebarHeader } from "./SidebarHeader";
import { ResizeHandle } from "./ResizeHandle";
import { MaterialSymbol } from "@/components/MaterialSymbol/material-symbol";
import { EventItem } from "./EventItem";
import { useIntegrations } from "../hooks/useIntegrations";
import { useCanvasStore } from "../store/canvasStore";
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';

const EventSourceImageMap = {
  'webhook': <MaterialSymbol className='-mt-1 -mb-1' name="webhook" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />
}

interface EventSourceSidebarProps {
  selectedEventSource: SuperplaneEventSource & {
    events?: Array<SuperplaneEvent>;
    eventSourceType?: string;
  };
  onClose: () => void;
}

export const EventSourceSidebar = ({ selectedEventSource, onClose }: EventSourceSidebarProps) => {
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(400);
  const [searchQuery, setSearchQuery] = useState('');
  const canvasId = useCanvasStore(state => state.canvasId) || '';

  const { data: canvasIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");

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
  const events = selectedEventSource.events || [];
  const limitedEvents = events.slice(0, 20);

  const filteredEvents = limitedEvents.filter(event =>
    event.id?.toLowerCase().includes(searchQuery.toLowerCase())
  );

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

      <div className="p-4 border-b border-gray-200 dark:border-zinc-700">
        <div className="relative">
          <MaterialSymbol
            name="search"
            size="md"
            className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 dark:text-zinc-500"
          />
          <input
            type="text"
            placeholder="Search events..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100 placeholder-gray-500 dark:placeholder-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="p-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-zinc-100 uppercase tracking-wide">
              Event History ({filteredEvents.length})
            </h3>
          </div>

          <div className="space-y-2">
            {filteredEvents.length > 0 ? (
              filteredEvents.map((event) => (
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
              <div className="text-sm text-gray-500 dark:text-gray-400 italic py-8 text-center">
                {searchQuery ? 'No events match your search' : 'No events received'}
              </div>
            )}
          </div>

          {events.length > 20 && (
            <div className="mt-4 text-center">
              <button className="text-sm text-blue-600 dark:text-blue-400 hover:underline">
                Load more events
              </button>
            </div>
          )}
        </div>
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