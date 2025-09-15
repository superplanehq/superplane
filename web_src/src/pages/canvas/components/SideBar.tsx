import { useMemo, useState } from "react";
import { useParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { Stage } from "../store/types";

import { useResizableSidebar } from "../hooks/useResizableSidebar";
import { useStageExecutions, useStageQueueEvents, canvasKeys } from "@/hooks/useCanvasData";

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
  selectedStage: Stage;
  onClose: () => void;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  discardStageEvent: (stageEventId: string, stageId: string) => Promise<void>;
  cancelStageExecution: (executionId: string, stageId: string) => Promise<void>;
}

export const Sidebar = ({ selectedStage, onClose, approveStageEvent, discardStageEvent, cancelStageExecution }: SidebarProps) => {
  const [activeTab, setActiveTab] = useState('activity');
  const { organizationId, canvasId } = useParams<{ organizationId: string, canvasId: string }>();
  const { width, isDragging, sidebarRef, handleMouseDown } = useResizableSidebar(600);
  const queryClient = useQueryClient();

  // Wrapper function to handle approval with query invalidation
  const handleApproveStageEvent = async (stageEventId: string, stageId: string) => {
    try {
      await approveStageEvent(stageEventId, stageId);
      
      // Invalidate queries to refresh both stage events and executions data
      await queryClient.invalidateQueries({
        queryKey: canvasKeys.stageEvents(canvasId || '', stageId, ['STATE_PENDING', 'STATE_WAITING'])
      });
      await queryClient.invalidateQueries({
        queryKey: canvasKeys.stageExecutions(canvasId || '', stageId)
      });
    } catch (error) {
      console.error('Failed to approve stage event:', error);
    }
  };

  // Wrapper function to handle discard with query invalidation
  const handleDiscardStageEvent = async (stageEventId: string, stageId: string) => {
    try {
      await discardStageEvent(stageEventId, stageId);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({
        queryKey: canvasKeys.stageEvents(canvasId || '', stageId, ['STATE_PENDING', 'STATE_WAITING'])
      });
    } catch (error) {
      console.error('Failed to discard stage event:', error);
    }
  };

  // Wrapper function to handle execution cancellation with query invalidation
  const handleCancelStageExecution = async (executionId: string, stageId: string) => {
    try {
      await cancelStageExecution(executionId, stageId);
      
      // Invalidate queries to refresh the data
      await queryClient.invalidateQueries({
        queryKey: canvasKeys.stageExecutions(canvasId || '', stageId)
      });
    } catch (error) {
      console.error('Failed to cancel stage execution:', error);
    }
  };

  const tabs = useMemo(() => [
    { key: 'activity', label: 'Activity' },
    { key: 'history', label: 'History' },
    { key: 'settings', label: 'Settings' },
  ], []);









  // Fetch executions directly from the API
  const {
    data: executionsData
  } = useStageExecutions(canvasId || '', selectedStage.metadata?.id || '');

  // Fetch pending and waiting stage events
  const {
    data: queueEventsData
  } = useStageQueueEvents(canvasId || '', selectedStage.metadata?.id || '', ['STATE_PENDING', 'STATE_WAITING']);

  const partialExecutions = useMemo(() =>
    executionsData?.pages.flatMap(page => page.executions) || [],
    [executionsData?.pages]
  );

  const executionRunning = useMemo(() =>
    partialExecutions.some(execution => execution.state === 'STATE_STARTED'),
    [partialExecutions]
  );

  // Get all queue events from the API
  const allQueueEvents = useMemo(() =>
    queueEventsData?.pages.flatMap(page => page.events) || [],
    [queueEventsData?.pages]
  );

  // Filter events by their state
  const pendingEvents = useMemo(() =>
    allQueueEvents.filter(event => event.state === 'STATE_PENDING'),
    [allQueueEvents]
  );

  const waitingEvents = useMemo(() =>
    allQueueEvents.filter(event => event.state === 'STATE_WAITING'),
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
            partialExecutions={partialExecutions}
            approveStageEvent={handleApproveStageEvent}
            executionRunning={executionRunning}
            organizationId={organizationId!}
            discardStageEvent={handleDiscardStageEvent}
            cancelStageExecution={handleCancelStageExecution}
          />
        );

      case 'history':
        return <HistoryTab
          canvasId={canvasId!}
          approveStageEvent={handleApproveStageEvent}
          organizationId={organizationId!}
          selectedStage={selectedStage}
          discardStageEvent={handleDiscardStageEvent}
          cancelStageExecution={handleCancelStageExecution}
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