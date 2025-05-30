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
  approveStageEvent: (stageEventId: string, stageId: string) => void;
}

export const Sidebar = ({ selectedStage, onClose, approveStageEvent }: SidebarProps) => {
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

  const allExecutions = useMemo(() =>
    selectedStage.queue
      ?.flatMap(event => event.execution as SuperplaneExecution)
      .filter(execution => execution)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
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

  // Helper functions for better display
  const formatRelativeTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  const getExecutionStatusIcon = (state: string, result?: string) => {
    switch (state) {
      case 'STATE_PENDING': return '⏳';
      case 'STATE_STARTED': return '🔄';
      case 'STATE_FINISHED':
        return result === 'RESULT_PASSED' ? '✅' : result === 'RESULT_FAILED' ? '❌' : '⚪';
      default: return '⚪';
    }
  };

  const getExecutionStatusColor = (state: string, result?: string) => {
    switch (state) {
      case 'STATE_PENDING': return 'text-amber-600 bg-amber-50';
      case 'STATE_STARTED': return 'text-blue-600 bg-blue-50';
      case 'STATE_FINISHED':
        return result === 'RESULT_PASSED' ? 'text-green-600 bg-green-50' : 
               result === 'RESULT_FAILED' ? 'text-red-600 bg-red-50' : 'text-gray-600 bg-gray-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  };

  // Render the appropriate content based on the active tab
  const renderTabContent = () => {
    switch (activeTab) {
      case 'general':
        return (
          <div className="p-6 space-y-6">
            {/* Stage Overview Card */}
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">Stage Overview</h3>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-gray-500">Total Events</span>
                  <div className="font-medium text-gray-900">{selectedStage.queue?.length || 0}</div>
                </div>
                <div>
                  <span className="text-gray-500">Pending</span>
                  <div className="font-medium text-amber-600">{pendingEvents.length}</div>
                </div>
                <div>
                  <span className="text-gray-500">Waiting</span>
                  <div className="font-medium text-blue-600">{waitingEvents.length}</div>
                </div>
                <div>
                  <span className="text-gray-500">Processed</span>
                  <div className="font-medium text-green-600">{processedEvents.length}</div>
                </div>
              </div>
            </div>

            {/* Pending Runs Section */}
            <div className="bg-white rounded-lg border border-gray-200">
              <div className="p-4 border-b border-gray-200">
                <h4 className="text-sm font-medium text-gray-700 flex items-center">
                  <span className="material-symbols-outlined text-amber-600 mr-2">pending</span>
                  Pending Runs ({pendingEvents.length})
                </h4>
              </div>
              <div className="p-4">
                {pendingEvents.length > 0 ? (
                  <div className="space-y-3">
                    {pendingEvents.slice(0, 3).map((event, index) => (
                      <div key={event.id} className="flex items-center justify-between p-3 bg-amber-50 rounded-lg">
                        <div className="flex items-center space-x-3">
                          <div className="material-symbols-outlined text-amber-600">pending</div>
                          <div>
                            <div className="text-sm font-medium text-gray-900">
                              {formatRelativeTime(event.createdAt || '')}
                            </div>
                            <div className="text-xs text-gray-500">ID: {event.id?.substring(0, 8)}...</div>
                          </div>
                        </div>
                        <div className="text-xs text-amber-600 bg-amber-100 px-2 py-1 rounded">
                          {event.state?.replace('STATE_', '')}
                        </div>
                      </div>
                    ))}
                    {pendingEvents.length > 3 && (
                      <div className="text-center">
                        <button className="text-sm text-amber-600 hover:text-amber-800">
                          View {pendingEvents.length - 3} more pending
                        </button>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-6 text-gray-500">
                    <div className="material-symbols-outlined text-4xl mb-2 opacity-50">pending</div>
                    <div className="text-sm">No pending runs</div>
                  </div>
                )}
              </div>
            </div>

            {/* Waiting for Approval Section */}
            {waitingEvents.length > 0 && (
              <div className="bg-white rounded-lg border border-gray-200">
                <div className="p-4 border-b border-gray-200">
                  <h4 className="text-sm font-medium text-gray-700 flex items-center">
                    <span className="material-symbols-outlined text-blue-600 mr-2">hourglass_empty</span>
                    Waiting for Approval ({waitingEvents.length})
                  </h4>
                </div>
                <div className="p-4 space-y-3">
                  {waitingEvents.slice(0, 2).map((event) => (
                    <div key={event.id} className="flex items-center justify-between p-3 bg-blue-50 rounded-lg">
                      <div className="flex items-center space-x-3">
                        <div className="material-symbols-outlined text-blue-600">hourglass_empty</div>
                        <div>
                          <div className="text-sm font-medium text-gray-900">
                            {formatRelativeTime(event.createdAt || '')}
                          </div>
                          <div className="text-xs text-gray-500">ID: {event.id?.substring(0, 8)}...</div>
                        </div>
                      </div>
                      <button onClick={() => approveStageEvent(event.id!, selectedStage.id!)} className="px-3 py-1.5 text-xs font-medium text-white rounded-md transition-colors" style={{ backgroundColor: '#2563eb' }} onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#1d4ed8'} onMouseLeave={(e) => e.currentTarget.style.backgroundColor = '#2563eb'}>
                        Approve
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Recent Activity */}
            <div className="bg-white rounded-lg border border-gray-200">
              <div className="p-4 border-b border-gray-200">
                <h4 className="text-sm font-medium text-gray-700">Recent Activity</h4>
              </div>
              <div className="p-4">
                {allExecutions.length > 0 ? (
                  <div className="space-y-3">
                    {allExecutions.slice(0, 5).map((execution, index) => (
                      <div key={execution.id || index} className="flex items-center justify-between">
                        <div className="flex items-center space-x-3">
                          <span className="text-lg">{getExecutionStatusIcon(execution.state || '', execution.result)}</span>
                          <div>
                            <div className="text-sm text-gray-900">
                              {execution.state?.replace('STATE_', '').toLowerCase()}
                            </div>
                            <div className="text-xs text-gray-500">
                              {formatRelativeTime(execution.createdAt || '')}
                            </div>
                          </div>
                        </div>
                        <div className={`text-xs px-2 py-1 rounded ${getExecutionStatusColor(execution.state || '', execution.result)}`}>
                          {execution.result?.replace('RESULT_', '') || 'N/A'}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-6 text-gray-500">
                    <div className="text-4xl mb-2">📊</div>
                    <div className="text-sm">No recent activity</div>
                  </div>
                )}
              </div>
            </div>
          </div>
        );

      case 'history':
        return (
          <div className="p-6">
            <div className="mb-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-2">Run History</h2>
              <p className="text-gray-600">Complete execution history for this stage</p>
            </div>

            {/* Execution Stats */}
            <div className="grid grid-cols-3 gap-4 mb-6">
              <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
                <div className="text-2xl font-bold text-green-600">
                  {allExecutions.filter(e => e.result === 'RESULT_PASSED').length}
                </div>
                <div className="text-sm text-gray-500">Passed</div>
              </div>
              <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
                <div className="text-2xl font-bold text-red-600">
                  {allExecutions.filter(e => e.result === 'RESULT_FAILED').length}
                </div>
                <div className="text-sm text-gray-500">Failed</div>
              </div>
              <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
                <div className="text-2xl font-bold text-amber-600">
                  {allExecutions.filter(e => e.state === 'STATE_PENDING').length}
                </div>
                <div className="text-sm text-gray-500">Pending</div>
              </div>
            </div>

            {/* Execution Timeline */}
            <div className="bg-white rounded-lg border border-gray-200">
              <div className="p-4 border-b border-gray-200">
                <h3 className="font-medium text-gray-900">Execution Timeline</h3>
              </div>
              <div className="p-4">
                {allExecutions.length > 0 ? (
                  <div className="space-y-4">
                    {allExecutions.map((execution, index) => (
                      <div key={execution.id || index} className="flex items-start space-x-4">
                        <div className="flex-shrink-0">
                          <span className="text-2xl">{getExecutionStatusIcon(execution.state || '', execution.result)}</span>
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between">
                            <div className="text-sm font-medium text-gray-900">
                              Execution {execution.state?.replace('STATE_', '').toLowerCase()}
                            </div>
                            <div className="text-xs text-gray-500">
                              {formatRelativeTime(execution.createdAt || '')}
                            </div>
                          </div>
                          <div className="text-xs text-gray-500 mt-1">
                            ID: {execution.id?.substring(0, 12)}...
                          </div>
                          {execution.referenceId && (
                            <div className="text-xs text-gray-500">
                              Ref: {execution.referenceId}
                            </div>
                          )}
                          <div className="flex items-center space-x-4 mt-2 text-xs text-gray-500">
                            {execution.startedAt && (
                              <span>Started: {new Date(execution.startedAt).toLocaleString()}</span>
                            )}
                            {execution.finishedAt && (
                              <span>Finished: {new Date(execution.finishedAt).toLocaleString()}</span>
                            )}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-8 text-gray-500">
                    <div className="text-4xl mb-3">📋</div>
                    <div className="text-sm">No execution history</div>
                  </div>
                )}
              </div>
            </div>
          </div>
        );

      case 'queue':
        return (
          <div className="p-6">
            <div className="mb-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-2">Event Queue</h2>
              <p className="text-gray-600">All events in the stage queue</p>
            </div>

            {/* Queue Status Cards */}
            <div className="grid grid-cols-1 gap-4 mb-6">
              {/* Pending Events */}
              {pendingEvents.length > 0 && (
                <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-medium text-amber-800 flex items-center">
                      <span className="material-symbols-outlined mr-2">pending</span>
                      Pending Events ({pendingEvents.length})
                    </h3>
                  </div>
                  <div className="space-y-2">
                    {pendingEvents.map((event, index) => (
                      <div key={event.id} className="bg-white rounded p-3 border border-amber-200">
                        <div className="flex justify-between items-start">
                          <div>
                            <div className="text-sm font-medium text-gray-900">
                              Event #{event.id?.substring(0, 8)}...
                            </div>
                            <div className="text-xs text-gray-500 mt-1">
                              Created: {formatRelativeTime(event.createdAt || '')}
                            </div>
                            {event.stateReason && (
                              <div className="text-xs text-amber-600 mt-1">
                                Reason: {event.stateReason.replace('STATE_REASON_', '')}
                              </div>
                            )}
                          </div>
                          <div className="text-xs bg-amber-100 text-amber-800 px-2 py-1 rounded">
                            PENDING
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Waiting Events */}
              {waitingEvents.length > 0 && (
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-medium text-blue-800 flex items-center">
                      <span className="material-symbols-outlined mr-2">hourglass_empty</span>
                      Waiting for Approval ({waitingEvents.length})
                    </h3>
                  </div>
                  <div className="space-y-2">
                    {waitingEvents.map((event) => (
                      <div key={event.id} className="bg-white rounded p-3 border border-blue-200">
                        <div className="flex justify-between items-start">
                          <div>
                            <div className="text-sm font-medium text-gray-900">
                              Event #{event.id?.substring(0, 8)}...
                            </div>
                            <div className="text-xs text-gray-500 mt-1">
                              Created: {formatRelativeTime(event.createdAt || '')}
                            </div>
                            {event.approvals && event.approvals.length > 0 && (
                              <div className="text-xs text-blue-600 mt-1">
                                Approvals: {event.approvals.length}
                              </div>
                            )}
                          </div>
                          <button onClick={() => approveStageEvent(event.id!, selectedStage.id!)} className="text-xs text-white px-3 py-1 rounded transition-colors" style={{ backgroundColor: '#2563eb' }} onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#1d4ed8'} onMouseLeave={(e) => e.currentTarget.style.backgroundColor = '#2563eb'}>
                            Approve
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Processed Events */}
              {processedEvents.length > 0 && (
                <div className="bg-green-50 border border-green-200 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="font-medium text-green-800 flex items-center">
                      <span className="material-symbols-outlined mr-2">check_circle</span>
                      Processed Events ({processedEvents.length})
                    </h3>
                  </div>
                  <div className="space-y-2 max-h-60 overflow-y-auto">
                    {processedEvents.slice(0, 10).map((event) => (
                      <div key={event.id} className="bg-white rounded p-3 border border-green-200">
                        <div className="flex justify-between items-start">
                          <div>
                            <div className="text-sm font-medium text-gray-900">
                              Event #{event.id?.substring(0, 8)}...
                            </div>
                            <div className="text-xs text-gray-500 mt-1">
                              Processed: {formatRelativeTime(event.createdAt || '')}
                            </div>
                          </div>
                          <div className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded">
                            PROCESSED
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Empty State */}
            {selectedStage.queue?.length === 0 && (
              <div className="text-center py-12 text-gray-500">
                <div className="text-6xl mb-4">📋</div>
                <div className="text-lg font-medium mb-2">No Events in Queue</div>
                <div className="text-sm">Events will appear here when they're triggered</div>
              </div>
            )}
          </div>
        );

      case 'settings':
        return (
          <div className="p-6">
            <div className="mb-6">
              <h2 className="text-lg font-semibold text-gray-900 mb-2">Stage Settings</h2>
              <p className="text-gray-600">Configuration and details for this stage</p>
            </div>

            {/* Basic Information */}
            <div className="bg-white rounded-lg border border-gray-200 mb-6">
              <div className="p-4 border-b border-gray-200">
                <h3 className="font-medium text-gray-900">Basic Information</h3>
              </div>
              <div className="p-4 space-y-4">
                <div className="flex justify-between items-center py-2">
                  <span className="text-gray-700 font-medium">Stage Name</span>
                  <span className="text-gray-900 font-mono text-sm">{selectedStage.name}</span>
                </div>
                <div className="flex justify-between items-center py-2">
                  <span className="text-gray-700 font-medium">Stage ID</span>
                  <span className="text-gray-900 font-mono text-sm">{selectedStage.id?.substring(0, 16)}...</span>
                </div>
                <div className="flex justify-between items-center py-2">
                  <span className="text-gray-700 font-medium">Canvas ID</span>
                  <span className="text-gray-900 font-mono text-sm">{selectedStage.canvasId?.substring(0, 16)}...</span>
                </div>
                <div className="flex justify-between items-center py-2">
                  <span className="text-gray-700 font-medium">Created</span>
                  <span className="text-gray-900">{selectedStage.createdAt ? new Date(selectedStage.createdAt).toLocaleString() : 'N/A'}</span>
                </div>
              </div>
            </div>

            {/* Connections */}
            <div className="bg-white rounded-lg border border-gray-200 mb-6">
              <div className="p-4 border-b border-gray-200">
                <h3 className="font-medium text-gray-900">Connections</h3>
              </div>
              <div className="p-4">
                {selectedStage.connections && selectedStage.connections.length > 0 ? (
                  <div className="space-y-3">
                    {selectedStage.connections.map((connection, index) => (
                      <div key={index} className="border border-gray-200 rounded p-3">
                        <div className="flex justify-between items-start mb-2">
                          <div className="font-medium text-gray-900">
                            {connection.name || `Connection ${index + 1}`}
                          </div>
                          <div className="text-xs bg-gray-100 text-gray-700 px-2 py-1 rounded">
                            {connection.type?.replace('TYPE_', '')}
                          </div>
                        </div>
                        {connection.filters && connection.filters.length > 0 && (
                          <div className="text-sm text-gray-600">
                            Filters: {connection.filters.length} configured
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-4 text-gray-500 text-sm">No connections configured</div>
                )}
              </div>
            </div>

            {/* Conditions */}
            <div className="bg-white rounded-lg border border-gray-200 mb-6">
              <div className="p-4 border-b border-gray-200">
                <h3 className="font-medium text-gray-900">Conditions</h3>
              </div>
              <div className="p-4">
                {selectedStage.conditions && selectedStage.conditions.length > 0 ? (
                  <div className="space-y-3">
                    {selectedStage.conditions.map((condition, index) => (
                      <div key={index} className="border border-gray-200 rounded p-3">
                        <div className="flex justify-between items-start mb-2">
                          <div className="font-medium text-gray-900">
                            {condition.type?.replace('CONDITION_TYPE_', '').replace('_', ' ')}
                          </div>
                        </div>
                        {condition.approval && (
                          <div className="text-sm text-gray-600">
                            Required approvals: {condition.approval.count}
                          </div>
                        )}
                        {condition.timeWindow && (
                          <div className="text-sm text-gray-600">
                            Time window: {condition.timeWindow.start} - {condition.timeWindow.end}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-4 text-gray-500 text-sm">No conditions configured</div>
                )}
              </div>
            </div>

            {/* Run Template */}
            <div className="bg-white rounded-lg border border-gray-200">
              <div className="p-4 border-b border-gray-200">
                <h3 className="font-medium text-gray-900">Run Template</h3>
              </div>
              <div className="p-4">
                {selectedStage.runTemplate ? (
                  <div className="space-y-3">
                    <div className="flex justify-between items-center">
                      <span className="text-gray-700 font-medium">Type</span>
                      <span className="text-gray-900">{selectedStage.runTemplate.type?.replace('TYPE_', '')}</span>
                    </div>
                    {selectedStage.runTemplate.semaphore && (
                      <>
                        <div className="flex justify-between items-center">
                          <span className="text-gray-700 font-medium">Project ID</span>
                          <span className="text-gray-900 font-mono text-sm">{selectedStage.runTemplate.semaphore.projectId}</span>
                        </div>
                        <div className="flex justify-between items-center">
                          <span className="text-gray-700 font-medium">Branch</span>
                          <span className="text-gray-900 font-mono text-sm">{selectedStage.runTemplate.semaphore.branch}</span>
                        </div>
                        <div className="flex justify-between items-center">
                          <span className="text-gray-700 font-medium">Pipeline File</span>
                          <span className="text-gray-900 font-mono text-sm">{selectedStage.runTemplate.semaphore.pipelineFile}</span>
                        </div>
                      </>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-4 text-gray-500 text-sm">No run template configured</div>
                )}
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