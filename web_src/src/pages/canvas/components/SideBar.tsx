import { useMemo, useState } from "react";
import { ExecutionWithEvent, StageWithEventQueue } from "../store/types";

import { useResizableSidebar } from "../hooks/useResizableSidebar";

import { SidebarHeader } from "./SidebarHeader";
import { SidebarTabs } from "./SidebarTabs";
import { ResizeHandle } from "./ResizeHandle";
import { ActivityTab } from "./tabs/ActivityTab";
import { HistoryTab } from "./tabs/HistoryTab";
import { SettingsTab } from "./tabs/SettingsTab";

interface SidebarProps {
  selectedStage: StageWithEventQueue;
  onClose: () => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
}

export const Sidebar = ({ selectedStage, onClose, approveStageEvent }: SidebarProps) => {
  const [activeTab, setActiveTab] = useState('activity');
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(600);

  // Sidebar tab definitions - memoized to prevent unnecessary re-renders
  const tabs = useMemo(() => [
    { key: 'activity', label: 'Activity' },
    { key: 'history', label: 'History' },
    { key: 'settings', label: 'Settings' },
  ], []);

  const allExecutions = useMemo(() =>
    selectedStage.queue
      ?.filter(event => event.execution)
      .flatMap(event => ({...event.execution, event}) as ExecutionWithEvent)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [selectedStage.queue]
  );

  const executionRunning = useMemo(() =>
    allExecutions.some(execution => execution.state === 'STATE_STARTED'),
    [allExecutions]
  );

  // Filter events by their state
  const pendingEvents = useMemo(() =>
    selectedStage.queue?.filter(event => event.state === 'STATE_PENDING') || [],
    [selectedStage.queue]
  );

  const waitingEvents = useMemo(() =>
    selectedStage.queue?.filter(event => event.state === 'STATE_WAITING') || [],
    [selectedStage.queue]
  );

  // const processedEvents = useMemo(() =>
  //   selectedStage.queue?.filter(event => event.state === 'STATE_PROCESSED') || [],
  //   [selectedStage.queue]
  // );

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
          />
        );

      case 'history':
        return <HistoryTab allExecutions={allExecutions} selectedStage={selectedStage} />;

      case 'settings':
        return <SettingsTab selectedStage={selectedStage} />;

      default:
        return null;
    }
  };

  return (
    <aside
      ref={sidebarRef}
      className={`fixed top-12 right-0 h-screen z-10 bg-white flex flex-col ${
        isDragging.current ? '' : 'transition-all duration-200'
      }`}
      style={{
        width: width,
        minWidth: 300,
        maxWidth: 800,
        boxShadow: 'rgba(0,0,0,0.07) -2px 0 12px',
      }}
    >
      {/* Sidebar Header */}
      <SidebarHeader stageName={selectedStage.metadata!.name || ''} onClose={onClose} />

      {/* Sidebar Tabs */}
      <SidebarTabs tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} />

      {/* Sidebar Content */}
      <div className="flex-1 overflow-y-auto bg-gray-50">
        {renderTabContent()}
      </div>

      {/* Resize Handle */}
      <ResizeHandle
        isDragging={isDragging.current}
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