import { StageWithEventQueue } from "../../store/types";
import { SuperplaneStageEvent } from "@/api-client";

interface QueueTabProps {
  selectedStage: StageWithEventQueue;
  pendingEvents: SuperplaneStageEvent[];
  waitingEvents: SuperplaneStageEvent[];
  processedEvents: SuperplaneStageEvent[];
  approveStageEvent: (stageEventId: string, stageId: string) => void;
}

export const QueueTab = ({
  selectedStage,
  pendingEvents,
  waitingEvents,
  processedEvents,
  approveStageEvent
}: QueueTabProps) => {
  if (selectedStage.queue?.length === 0) {
    return (
      <div className="p-6">
        <div className="mb-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-2">Event Queue</h2>
          <p className="text-gray-600">All events in the stage queue</p>
        </div>
        <div className="text-center py-12 text-gray-500">
          <div className="text-6xl mb-4">📋</div>
          <div className="text-lg font-medium mb-2">No Events in Queue</div>
          <div className="text-sm">Events will appear here when they're triggered</div>
        </div>
      </div>
    );
  }

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
              {pendingEvents.map((event) => (
                <div key={event.id} className="bg-white rounded p-3 border border-amber-200">
                  <div className="flex justify-between items-start">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        Event #{event.id?.substring(0, 8)}...
                      </div>
                      <div className="text-xs text-gray-500 mt-1">
                        Created: {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'N/A'}
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
                        Created: {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'N/A'}
                      </div>
                      {event.approvals && event.approvals.length > 0 && (
                        <div className="text-xs text-blue-600 mt-1">
                          Approvals: {event.approvals.length}
                        </div>
                      )}
                    </div>
                    <button 
                      onClick={() => approveStageEvent(event.id!, selectedStage.id!)} 
                      className="text-xs text-white px-3 py-1 rounded transition-colors" 
                      style={{ backgroundColor: '#2563eb' }} 
                      onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#1d4ed8'} 
                      onMouseLeave={(e) => e.currentTarget.style.backgroundColor = '#2563eb'}
                    >
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
                        Processed: {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'N/A'}
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
    </div>
  );
};