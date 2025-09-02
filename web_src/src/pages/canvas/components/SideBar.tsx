import { useMemo, useState, useEffect, useCallback } from "react";
import { useParams } from 'react-router-dom';
import { ExecutionWithEvent, StageWithEventQueue } from "../store/types";

import { useResizableSidebar } from "../hooks/useResizableSidebar";
import { useStageEvents, useStageQueueEvents, useConnectedSourcesEvents } from "@/hooks/useCanvasData";

import { SidebarHeader } from "./SidebarHeader";
import { SidebarTabs } from "./SidebarTabs";
import { ResizeHandle } from "./ResizeHandle";
import { ActivityTab } from "./tabs/ActivityTab";
import { HistoryTab } from "./tabs/HistoryTab";
import { SettingsTab } from "./tabs/SettingsTab";
import { MaterialSymbol } from "@/components/MaterialSymbol/material-symbol";
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';
import { SuperplaneConnectionType, SuperplaneEvent, SuperplaneExecution } from "@/api-client";
import { useCanvasStore } from "../store/canvasStore";

const StageImageMap = {
  'http': <MaterialSymbol className='w-6 h-5 -mt-2' name="rocket_launch" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-8 h-8 object-contain p-1 rounded dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />
}

interface SidebarProps {
  selectedStage: StageWithEventQueue;
  onClose: () => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
}

export const Sidebar = ({ selectedStage, onClose, approveStageEvent }: SidebarProps) => {
  const [activeTab, setActiveTab] = useState('activity');
  const { organizationId, canvasId } = useParams<{ organizationId: string, canvasId: string }>();
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(450);
  const { connectionGroups, stages, eventSources } = useCanvasStore();

  // State for managing limit per source for connected events
  const [connectedEventsLimitPerSource, setConnectedEventsLimitPerSource] = useState(20);

  // Use the new infinite query hooks for stage events
  const {
    data: stageEventsData,
    fetchNextPage: fetchNextStageEvents,
    hasNextPage: hasNextStageEvents,
    isFetchingNextPage: isFetchingNextStageEvents,
    isLoading: stageEventsLoading,
    refetch: refetchStageEvents
  } = useStageEvents(canvasId || '', selectedStage.metadata?.id || '');

  const {
    data: stageQueueData,
    fetchNextPage: fetchNextQueueEvents,
    hasNextPage: hasNextQueueEvents,
    isFetchingNextPage: isFetchingNextQueueEvents,
    isLoading: queueEventsLoading,
    refetch: refetchQueueEvents
  } = useStageQueueEvents(canvasId || '', selectedStage.metadata?.id || '');

  // Flatten all pages into single arrays
  const allStageEvents = useMemo(() => {
    return stageEventsData?.pages.flatMap(page => page.events) || [];
  }, [stageEventsData]);

  const allQueueEvents = useMemo(() => {
    return stageQueueData?.pages.flatMap(page => page.events) || [];
  }, [stageQueueData]);

  // Monitor for changes in selectedStage.events and selectedStage.queue and refetch if needed
  useEffect(() => {
    const bulkStageEvents = selectedStage.events || [];
    const bulkQueueEvents = selectedStage.queue || [];

    // Check stage events
    if (bulkStageEvents.length > 0 && allStageEvents.length > 0) {
      const latestBulkEvent = bulkStageEvents[0];
      const latestQueryEvent = allStageEvents[0];
      const hasPendingEvents = allStageEvents.some(event => event.state === 'STATE_PENDING');

      if (latestBulkEvent?.receivedAt && latestQueryEvent?.receivedAt) {
        const bulkTime = new Date(latestBulkEvent.receivedAt);
        const queryTime = new Date(latestQueryEvent.receivedAt);
        if (bulkTime > queryTime || hasPendingEvents) {
          refetchStageEvents();
        }
      }
    }

    // Check queue events
    if (bulkQueueEvents.length > 0 && allQueueEvents.length > 0) {
      const latestBulkEvent = bulkQueueEvents[0];
      const latestQueryEvent = allQueueEvents[0];
      const hasPendingEvents = allQueueEvents.some(event =>
        event.state === 'STATE_PENDING' || event.state === 'STATE_WAITING'
      );

      if (latestBulkEvent?.createdAt && latestQueryEvent?.createdAt) {
        const bulkTime = new Date(latestBulkEvent.createdAt);
        const queryTime = new Date(latestQueryEvent.createdAt);
        if (bulkTime > queryTime || hasPendingEvents) {
          refetchQueueEvents();
        }
      }
    }
  }, [selectedStage.events, selectedStage.queue, allStageEvents, allQueueEvents, refetchStageEvents, refetchQueueEvents]);

  // Sidebar tab definitions - memoized to prevent unnecessary re-renders
  const tabs = useMemo(() => [
    { key: 'activity', label: 'Activity' },
    { key: 'history', label: 'History' },
    { key: 'settings', label: 'Settings' },
  ], []);

  const allConnections: { type: SuperplaneConnectionType; name: string }[] = useMemo(() =>
    selectedStage.spec?.connections
      ?.map(connection => ({ type: connection.type as SuperplaneConnectionType, name: connection.name as string })) || [],
    [selectedStage.spec?.connections]
  );

  // Get connected source IDs
  const connectedSources = useMemo(() => {
    const eventSourceIds: string[] = [];
    const stageIds: string[] = [];
    const connectionGroupIds: string[] = [];

    allConnections.forEach(connection => {
      if (connection.type === 'TYPE_EVENT_SOURCE') {
        const eventSource = eventSources.find(es => es.metadata?.name === connection.name);
        if (eventSource?.metadata?.id) {
          eventSourceIds.push(eventSource.metadata.id);
        }
      } else if (connection.type === 'TYPE_STAGE') {
        const stage = stages.find(s => s.metadata?.name === connection.name);
        if (stage?.metadata?.id) {
          stageIds.push(stage.metadata.id);
        }
      } else if (connection.type === 'TYPE_CONNECTION_GROUP') {
        const connectionGroup = connectionGroups.find(cg => cg.metadata?.name === connection.name);
        if (connectionGroup?.metadata?.id) {
          connectionGroupIds.push(connectionGroup.metadata.id);
        }
      }
    });

    return { eventSourceIds, stageIds, connectionGroupIds };
  }, [allConnections, eventSources, stages, connectionGroups]);

  // Use the new hook for connected sources events
  const {
    data: connectedEventsData,
    isFetchingNextPage: isFetchingNextConnectedEvents,
    isLoading: connectedEventsLoading,
    refetch: refetchConnectedEvents
  } = useConnectedSourcesEvents(canvasId || '', connectedSources, connectedEventsLimitPerSource);

  // Flatten all connected events into a single array
  const allConnectedEvents = useMemo(() => {
    return connectedEventsData?.pages.flatMap(page => page.events) || [];
  }, [connectedEventsData]);

  const connectionEventsById = useMemo(() => {
    const plainEventsById: Record<string, SuperplaneEvent> = {};

    // Use the paginated connected events data (20 events per source)
    allConnectedEvents.forEach(event => {
      if (event?.id) {
        plainEventsById[event.id] = event;
      }
    });

    return plainEventsById;
  }, [allConnectedEvents]);

  // Monitor connected events for changes and refetch if needed
  useEffect(() => {
    // Check if any bulk data has newer events than our query data
    const allBulkEvents = [
      ...eventSources.flatMap(es => es.events || []),
      ...stages.flatMap(s => s.events || []),
      ...connectionGroups.flatMap(cg => cg.events || [])
    ];

    if (allBulkEvents.length > 0 && allConnectedEvents.length > 0) {
      const latestBulkEvent = allBulkEvents[0];
      const latestConnectedEvent = allConnectedEvents[0];
      const hasPendingEvents = allConnectedEvents.some(event => event.state === 'STATE_PENDING');

      if (latestBulkEvent?.receivedAt && latestConnectedEvent?.receivedAt) {
        const bulkTime = new Date(latestBulkEvent.receivedAt);
        const connectedTime = new Date(latestConnectedEvent.receivedAt);
        if (bulkTime > connectedTime || hasPendingEvents) {
          refetchConnectedEvents();
        }
      }
    }
  }, [eventSources, stages, connectionGroups, allConnectedEvents, refetchConnectedEvents]);

  // Function to load more connected events by increasing limit per source
  const loadMoreConnectedEvents = useCallback(() => {
    setConnectedEventsLimitPerSource(prev => prev + 20);
  }, []);

  // Check if we can load more connected events (if any connected source might have more data)
  const canLoadMoreConnectedEvents = useMemo(() => {
    return (connectedSources.eventSourceIds.length > 0 ||
      connectedSources.stageIds.length > 0 ||
      connectedSources.connectionGroupIds.length > 0) &&
      connectedEventsLimitPerSource < 100; // Cap at 100 per source to prevent excessive requests
  }, [connectedSources, connectedEventsLimitPerSource]);


  const eventsByExecutionId = useMemo(() => {
    const emittedEventsById: Record<string, SuperplaneEvent> = {};

    allStageEvents?.forEach(event => {
      const execution = event.raw?.execution as SuperplaneExecution;
      if (execution?.id) {
        emittedEventsById[execution.id || ''] = event;
      }
    })

    return emittedEventsById;
  }, [allStageEvents]);

  const allExecutions = useMemo(() =>
    allQueueEvents
      ?.filter(event => event.execution)
      .flatMap(event => ({ ...event.execution, event }) as ExecutionWithEvent)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [allQueueEvents]
  );

  const executionRunning = useMemo(() =>
    allExecutions.some(execution => execution.state === 'STATE_STARTED'),
    [allExecutions]
  );

  // Filter events by their state
  const pendingEvents = useMemo(() =>
    allQueueEvents?.filter(event => event.state === 'STATE_PENDING' && !event.execution) || [],
    [allQueueEvents]
  );

  const waitingEvents = useMemo(() =>
    allQueueEvents?.filter(event => event.state === 'STATE_WAITING' && !event.execution) || [],
    [allQueueEvents]
  );

  // Render the appropriate content based on the active tab
  const renderTabContent = () => {
    switch (activeTab) {
      case 'activity':
        return (
          <ActivityTab
            onChangeTab={setActiveTab}
            selectedStage={selectedStage}
            pendingEvents={pendingEvents}
            waitingEvents={waitingEvents}
            allExecutions={allExecutions}
            approveStageEvent={approveStageEvent}
            executionRunning={executionRunning}
            organizationId={organizationId!}
            connectionEventsById={connectionEventsById}
            eventsByExecutionId={eventsByExecutionId}
          />
        );

      case 'history':
        return <HistoryTab
          approveStageEvent={approveStageEvent}
          allExecutions={allExecutions}
          organizationId={organizationId!}
          selectedStage={selectedStage}
          allStageEvents={allQueueEvents || []}
          connectionEventsById={connectionEventsById}
          eventsByExecutionId={eventsByExecutionId}
          hasNextQueueEvents={hasNextQueueEvents}
          fetchNextQueueEvents={fetchNextQueueEvents}
          isFetchingNextQueueEvents={isFetchingNextQueueEvents}
          hasNextStageEvents={hasNextStageEvents}
          fetchNextStageEvents={fetchNextStageEvents}
          isFetchingNextStageEvents={isFetchingNextStageEvents}
          queueEventsLoading={queueEventsLoading}
          stageEventsLoading={stageEventsLoading}
          hasNextConnectedEvents={canLoadMoreConnectedEvents}
          fetchNextConnectedEvents={loadMoreConnectedEvents}
          isFetchingNextConnectedEvents={isFetchingNextConnectedEvents}
          connectedEventsLoading={connectedEventsLoading}
        />;

      case 'settings':
        return <SettingsTab selectedStage={selectedStage} />;

      default:
        return null;
    }
  };

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
      {/* Sidebar Header */}
      <SidebarHeader image={StageImageMap[(selectedStage.spec?.executor?.type || 'http') as keyof typeof StageImageMap]} stageName={selectedStage.metadata!.name || ''} onClose={onClose} />

      {/* Sidebar Tabs */}
      <SidebarTabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} />

      {/* Sidebar Content */}
      <div className="flex-1 overflow-y-auto">
        {renderTabContent()}
      </div>

      {/* Resize Handle */}
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