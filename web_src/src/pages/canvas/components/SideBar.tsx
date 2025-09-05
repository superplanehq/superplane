import { useEffect, useMemo, useState } from "react";
import { useParams } from 'react-router-dom';
import { ExecutionWithEvent, StageWithEventQueue } from "../store/types";

import { useResizableSidebar } from "../hooks/useResizableSidebar";

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
import { useConnectedSourcesEvents } from "@/hooks/useCanvasData";

const StageImageMap = {
  'http': <MaterialSymbol className='w-6 h-5 -mt-2' name="rocket_launch" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-8 h-8 object-contain p-1 rounded dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />
}

interface SidebarProps {
  selectedStage: StageWithEventQueue;
  onClose: () => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  cancelStageEvent: (stageEventId: string, stageId: string) => void;
}

export const Sidebar = ({ selectedStage, onClose, approveStageEvent, cancelStageEvent }: SidebarProps) => {
  const [activeTab, setActiveTab] = useState('activity');
  const { organizationId, canvasId } = useParams<{ organizationId: string, canvasId: string }>();
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(450);
  const { connectionGroups, stages, eventSources } = useCanvasStore();

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

  const partialConnectionEvents = useMemo(() => {
    const plainEventsById: SuperplaneEvent[] = [];

    const connectedEventSourceNames = new Set<string>();
    const connectedStageNames = new Set<string>();
    const connectedConnectionGroupNames = new Set<string>();

    allConnections.forEach(connection => {
      if (connection.type === 'TYPE_EVENT_SOURCE') {
        connectedEventSourceNames.add(connection.name);
      } else if (connection.type === 'TYPE_STAGE') {
        connectedStageNames.add(connection.name);
      } else if (connection.type === 'TYPE_CONNECTION_GROUP') {
        connectedConnectionGroupNames.add(connection.name);
      }
    });

    eventSources.forEach(eventSource => {
      if (connectedEventSourceNames.has(eventSource.metadata?.name || '')) {
        eventSource?.events?.forEach(event => {
          plainEventsById.push(event);
        });
      }
    });

    stages.forEach(stage => {
      if (connectedStageNames.has(stage.metadata?.name || '')) {
        stage?.events?.forEach(event => {
          plainEventsById.push(event);
        });
      }
    });

    connectionGroups.forEach(connectionGroup => {
      if (connectedConnectionGroupNames.has(connectionGroup.metadata?.name || '')) {
        connectionGroup?.events?.forEach(event => {
          plainEventsById.push(event);
        });
      }
    });

    return plainEventsById;
  }, [allConnections, eventSources, stages, connectionGroups]);


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


  const {
    data: connectedEventsData,
    isFetchingNextPage: isFetchingNextConnectedEvents,
    refetch: refetchConnectedEvents,
    fetchNextPage: fetchNextConnectedEvents,
  } = useConnectedSourcesEvents(canvasId || '', connectedSources, 20);

  const connectionEventsById = useMemo(() => {
    const eventsById: Record<string, SuperplaneEvent> = {};

    connectedEventsData?.pages.forEach(page => {
      page.events.forEach(event => {
        eventsById[event.id || ''] = event;
      });
    });

    return eventsById;
  }, [connectedEventsData]);

  const eventsByExecutionId = useMemo(() => {
    const emittedEventsById: Record<string, SuperplaneEvent> = {};

    selectedStage.events?.forEach(event => {
      const execution = event.raw?.execution as SuperplaneExecution;
      if (execution?.id) {
        emittedEventsById[execution.id || ''] = event;
      }
    })

    return emittedEventsById;
  }, [selectedStage.events]);

  const partialExecutions = useMemo(() =>
    selectedStage.queue
      ?.filter(event => event.execution)
      .flatMap(event => ({ ...event.execution, event }) as ExecutionWithEvent)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [selectedStage.queue]
  );

  const executionRunning = useMemo(() =>
    partialExecutions.some(execution => execution.state === 'STATE_STARTED'),
    [partialExecutions]
  );

  // Filter events by their state
  const pendingEvents = useMemo(() =>
    selectedStage.queue?.filter(event => event.state === 'STATE_PENDING' && !event.execution) || [],
    [selectedStage.queue]
  );

  const waitingEvents = useMemo(() =>
    selectedStage.queue?.filter(event => event.state === 'STATE_WAITING' && !event.execution) || [],
    [selectedStage.queue]
  );

  // Every time partialConnectionEvents changes, refetch connected events
  useEffect(() => {
    refetchConnectedEvents();
  }, [refetchConnectedEvents, partialConnectionEvents]);

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
            partialExecutions={partialExecutions}
            approveStageEvent={approveStageEvent}
            executionRunning={executionRunning}
            organizationId={organizationId!}
            connectionEventsById={connectionEventsById}
            eventsByExecutionId={eventsByExecutionId}
            cancelStageEvent={cancelStageEvent}
          />
        );

      case 'history':
        return <HistoryTab
          canvasId={canvasId!}
          approveStageEvent={approveStageEvent}
          organizationId={organizationId!}
          selectedStage={selectedStage}
          connectionEventsById={connectionEventsById}
          isFetchingNextConnectedEvents={isFetchingNextConnectedEvents}
          fetchNextConnectedEvents={fetchNextConnectedEvents}
          cancelStageEvent={cancelStageEvent}
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