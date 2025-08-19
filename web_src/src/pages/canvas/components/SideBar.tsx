import { useMemo, useState } from "react";
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
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(400);

  // Sidebar tab definitions - memoized to prevent unnecessary re-renders
  const tabs = useMemo(() => [
    { key: 'activity', label: 'Activity' },
    { key: 'history', label: 'History' },
    { key: 'settings', label: 'Settings' },
  ], []);

  const allExecutions = useMemo(() =>
    selectedStage.queue
      ?.filter(event => event.execution)
      .flatMap(event => ({ ...event.execution, event }) as ExecutionWithEvent)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [selectedStage.queue]
  );

  const executionRunning = useMemo(() =>
    allExecutions.some(execution => execution.state === 'STATE_STARTED'),
    [allExecutions]
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
        return <HistoryTab allExecutions={allExecutions} />;

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