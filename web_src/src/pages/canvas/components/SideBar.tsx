import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { StageWithEventQueue } from "../store/types";

// import tailwindcss classes
import 'tippy.js/dist/tippy.css';
import 'tippy.js/themes/light.css';
import '@xyflow/react/dist/style.css';
import { SuperplaneExecution } from "@/api-client";

interface SidebarProps {
  selectedStage: StageWithEventQueue;
  onClose: () => void;
}

export const Sidebar = ({ selectedStage, onClose }: SidebarProps) => {
  const [activeTab, setActiveTab] = useState('general');
  const [width, setWidth] = useState(600);
  const isDragging = useRef(false);
  const sidebarRef = useRef<HTMLDivElement>(null);
  const animationFrameRef = useRef<number | null>(null);

  // Sidebar tab definitions - memoized to prevent unnecessary re-renders
  const tabs = useMemo(() => [
    { key: 'general', label: 'General' },
    { key: 'history', label: 'History' },
    { key: 'queue', label: 'Queue' },
    { key: 'settings', label: 'Settings' },
  ], []);

  // Cleanup function for animation frame and event listeners
  useEffect(() => {
    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

  // Handle mouse down on resize handle - memoized to prevent recreation on each render
  const handleMouseDown = useCallback(() => {
    isDragging.current = true;
    document.body.style.cursor = 'ew-resize';
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }, []);

  // Handle mouse move during resize - memoized with dependencies
  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isDragging.current) return;
    // Cancel any pending animation frame to prevent queuing multiple updates
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
    }

    // Schedule width update in next animation frame to prevent layout thrashing
    animationFrameRef.current = requestAnimationFrame(() => {
      const newWidth = Math.max(300, Math.min(800, window.innerWidth - e.clientX));
      setWidth(newWidth);
      animationFrameRef.current = null;
    });
  }, []);

  // Handle mouse up to stop resizing - memoized to prevent recreation
  const handleMouseUp = useCallback(() => {
    isDragging.current = false;
    document.body.style.cursor = '';
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  }, []);

  const pendingExecutions = useMemo(() =>
    selectedStage.queue
      ?.flatMap(event => event.execution as SuperplaneExecution)
      .filter(execution => execution?.state === 'STATE_PENDING')
      .sort((a, b) => new Date(a?.createdAt || '').getTime() - new Date(b?.createdAt || '').getTime()) || [],
    [selectedStage.queue]
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
          <div>
            <h4 className="text-sm font-medium text-gray-700 mb-2">Pending Runs</h4>
            {pendingEvents.length > 0 ? (
              <>
                {/* Show the first pending item with details */}
                <div className="flex items-center p-2 bg-amber-50 rounded mb-1">
                  <div className="material-symbols-outlined text-amber-600 mr-2">pending</div>
                  <div className="flex-1">
                    <div className="text-sm font-medium">{new Date(pendingEvents[0].createdAt || '').toLocaleString()}</div>
                    <div className="text-xs text-gray-600">ID: {pendingEvents[0].id!.substring(0, 8)}...</div>
                  </div>
                </div>
                {/* Show count of additional pending items */}
                {pendingEvents.length > 1 && (
                  <div className="text-xs text-amber-600 hover:text-amber-800 mb-3">
                    <a href="#" className="no-underline hover:underline">{pendingEvents.length - 1} more pending</a>
                  </div>
                )}
              </>
            ) : (
              <div className="text-sm text-gray-500 italic mb-3">No pending items</div>
            )}

            {waitingEvents.length > 0 && (
              <>
                <h4 className="text-sm font-medium text-gray-700 mb-2 border-t pt-2">Waiting for Approval</h4>
                <div className="flex items-center p-2 bg-blue-50 rounded mb-1">
                  <div className="material-symbols-outlined text-blue-600 mr-2">hourglass_empty</div>
                  <div className="flex-1">
                    <div className="text-sm font-medium">{new Date(waitingEvents[0].createdAt!).toLocaleString()}</div>
                    <div className="text-xs text-gray-600">ID: {waitingEvents[0].id!.substring(0, 8)}...</div>
                  </div>
                  <button
                    onClick={() => { }}
                    className="ml-2 inline-flex items-center px-2.5 py-1.5 border border-transparent text-xs font-medium rounded text-white bg-blue-600! hover:bg-blue-700! focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  >
                    Approve
                  </button>
                </div>
                {waitingEvents.length > 1 && (
                  <div className="text-xs text-blue-600 hover:text-blue-800 mb-3">
                    <a href="#" className="no-underline hover:underline">{waitingEvents.length - 1} more waiting</a>
                  </div>
                )}
              </>
            )}
            {/* PROCESSED Queue Section - Only show the count */}
        {processedEvents.length > 0 && (
          <>
            <h4 className="text-sm font-medium text-gray-700 mb-2 border-t pt-2">Processed Recently</h4>
            <div className="flex items-center p-2 bg-green-50 rounded mb-1">
              <div className="material-symbols-outlined text-green-600 mr-2">check_circle</div>
              <div className="flex-1">
                <div className="text-sm">{processedEvents.length} processed</div>
                <div className="text-xs text-gray-600">Latest: {new Date(processedEvents[0].createdAt!).toLocaleString()}</div>
              </div>
            </div>
          </>
        )}
        
        {/* Show message when no queues exist */}
        {(!pendingEvents.length && !waitingEvents.length && !processedEvents.length) && (
          <div className="text-sm text-gray-500 italic">No queue activity</div>
        )}
            </div>
        );

      case 'history':
        return (
          <div className="py-6 px-8">

            <h2 className="text-lg font-semibold mb-0">Run History</h2>
            <p className="mb-6 text-gray-600">A record of recent executions for this stage.</p>

            {pendingExecutions.length > 0 && (
              <div className="space-y-4">
                {pendingExecutions.map((execution, index) => (
                  <div key={index} className="flex items-center p-4 bg-gray-50 rounded-lg">
                    <span className="text-2xl mr-4">⏳</span>
                    <span className="text-gray-900">{execution.state}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        );

      case 'queue':
        return (
          <div className="py-6 px-8">
            {pendingEvents.length > 0 && (
              <div className="space-y-4">
                {pendingEvents.map((event, index) => (
                  <div key={index} className="flex items-center p-4 bg-gray-50 rounded-lg">
                    <span className="text-2xl mr-4">⏳</span>
                    <span className="text-gray-900">{event.state}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

        );

      case 'settings':
        return (
          <div className="py-6 px-8">
            <h3 className="text-lg font-semibold mb-6">Settings</h3>
            <div className="space-y-6">
              <div className="flex justify-between items-center py-3 border-b border-gray-200">
                <span className="text-gray-700 font-medium">Stage Name</span>
                <span className="text-gray-900">{selectedStage.name}</span>
              </div>
              <div className="flex justify-between items-center py-3 border-b border-gray-200">
                <span className="text-gray-700 font-medium">Type</span>
                <span className="text-gray-900">{selectedStage.name}</span>
              </div>
              <div className="flex justify-between items-center py-3 border-b border-gray-200">
                <span className="text-gray-700 font-medium">Status</span>
                <span className="text-gray-900">{selectedStage.name}</span>
              </div>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <aside
      ref={sidebarRef}
      className={`fixed top-0 right-0 h-screen z-10 bg-white flex flex-col ${isDragging.current ? '' : 'transition-all duration-200'
        }`}
      style={{
        width: width,
        minWidth: 300,
        maxWidth: 800,
        boxShadow: 'rgba(0,0,0,0.07) -2px 0 12px',
      }}
    >
      {/* Sidebar Header with Stage Name */}
      <div className="flex items-center justify-between p-6 border-b border-gray-200 bg-gray-50">
        <div className="flex items-center">
          <span className="text-black font-bold mr-2 text-xl">📋</span>
          <span className="text-lg font-bold text-gray-900">{selectedStage.name}</span>
        </div>
        <button
          className="text-gray-500 hover:text-gray-700 text-2xl font-bold w-8 h-8 flex items-center justify-center rounded hover:bg-gray-200 transition-colors"
          onClick={onClose}
          title="Close sidebar"
        >
          ×
        </button>
      </div>

      {/* Sidebar Tabs */}
      <div className="flex border-b border-gray-200 bg-white">
        {tabs.map(tab => (
          <button
            key={tab.key}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${activeTab === tab.key
                ? 'text-indigo-600 border-b-2 border-indigo-600 bg-indigo-50'
                : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
              }`}
            onClick={() => setActiveTab(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Sidebar Content */}
      <div className="flex-1 overflow-y-auto bg-gray-50">
        {renderTabContent()}
      </div>

      {/* Resize Handle */}
      <div
        className={`absolute left-0 top-0 bottom-0 w-2 cursor-ew-resize rounded transition-colors ${isDragging.current ? 'bg-gray-300' : 'bg-gray-200 hover:bg-gray-300'
          }`}
        style={{ zIndex: 100 }}
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