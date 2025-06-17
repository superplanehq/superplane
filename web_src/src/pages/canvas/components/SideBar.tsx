import { useMemo, useState } from "react";
import { StageWithEventQueue } from "../store/types";
import { SuperplaneExecution } from "@/api-client";

import { useResizableSidebar } from "../hooks/useResizableSidebar";

import { SidebarHeader } from "./SidebarHeader";
import { SidebarTabs } from "./SidebarTabs";
import { ResizeHandle } from "./ResizeHandle";
import { GeneralTab } from "./tabs/GeneralTab";
import { HistoryTab } from "./tabs/HistoryTab";
import { QueueTab } from "./tabs/QueueTab";
import { SettingsTab } from "./tabs/SettingsTab";

interface SidebarProps {
  selectedStage: StageWithEventQueue;
  onClose: () => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
}

export const Sidebar = ({ selectedStage, onClose, approveStageEvent }: SidebarProps) => {
  const [activeTab, setActiveTab] = useState('general');
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(600);

  // Sidebar tab definitions - memoized to prevent unnecessary re-renders
  const tabs = useMemo(() => [
    { key: 'general', label: 'General' },
    { key: 'history', label: 'History' },
    { key: 'queue', label: 'Queue' },
    { key: 'settings', label: 'Settings' },
  ], []);

  const allExecutions = useMemo(() =>
    selectedStage.queue
      ?.flatMap(event => event.execution as SuperplaneExecution)
      .filter(execution => execution)
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

  const processedEvents = useMemo(() =>
    selectedStage.queue?.filter(event => event.state === 'STATE_PROCESSED') || [],
    [selectedStage.queue]
  );

  // Render the appropriate content based on the active tab
  const renderTabContent = () => {
    switch (activeTab) {
      case 'general':
        return (
          <GeneralTab
            selectedStage={selectedStage}
            pendingEvents={pendingEvents}
            waitingEvents={waitingEvents}
            processedEvents={processedEvents}
            allExecutions={allExecutions}
            approveStageEvent={approveStageEvent}
            executionRunning={executionRunning}
          />
        );

      case 'history':
        return <HistoryTab allExecutions={allExecutions} />;

      case 'queue':
        return (
          <QueueTab
            selectedStage={selectedStage}
            pendingEvents={pendingEvents}
            waitingEvents={waitingEvents}
            processedEvents={processedEvents}
            approveStageEvent={approveStageEvent}
            executionRunning={executionRunning}
          />
        );

      case 'settings':
        return <SettingsTab selectedStage={selectedStage} />;

      default:
        return null;
    }
  };

  return (
    <aside
      ref={sidebarRef}
      className={`fixed top-0 right-0 h-screen z-10 bg-white flex flex-col ${
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