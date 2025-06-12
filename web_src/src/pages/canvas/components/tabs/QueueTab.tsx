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
  const renderEventInputs = (event: SuperplaneStageEvent) => {
    if (!event.inputs || event.inputs.length === 0) {
      return (
        <div className="text-xs text-gray-500 mt-2">No inputs</div>
      );
    }

    return (
      <div className="mt-2">
        <div className="text-xs font-medium text-gray-700 mb-1">Inputs:</div>
        <div className="space-y-1">
          {event.inputs.map((input, index) => (
            <div key={index} className="text-xs bg-gray-100 rounded px-2 py-1">
              <span className="font-mono text-gray-900">{input.name}</span>
              <span className="text-gray-600 ml-2">= {input.value}</span>
            </div>
          ))}
        </div>
      </div>
    );
  };

  const renderExecutionOutputs = (event: SuperplaneStageEvent) => {
    if (!event.execution?.outputs || event.execution.outputs.length === 0) {
      return (
        <div className="text-xs text-gray-500 mt-2">No Execution outputs</div>
      );
    }

    return (
      <div className="mt-2">
        <div className="text-xs font-medium text-gray-700 mb-1">Execution Outputs:</div>
        <div className="space-y-1">
          {event.execution.outputs.map((output, index) => (
            <div key={index} className="text-xs bg-green-100 rounded px-2 py-1">
              <span className="font-mono text-green-900">{output.name}</span>
              <span className="text-green-700 ml-2">= {output.value}</span>
            </div>
          ))}
        </div>
      </div>
    );
  };

  const renderExecutionStatus = (event: SuperplaneStageEvent) => {
    if (!event.execution) return null;

    const execution = event.execution;
    const getStateColor = (state: string) => {
      switch (state) {
        case 'STATE_PENDING': return 'bg-yellow-100 text-yellow-800';
        case 'STATE_STARTED': return 'bg-blue-100 text-blue-800';
        case 'STATE_FINISHED': return 'bg-green-100 text-green-800';
        default: return 'bg-gray-100 text-gray-800';
      }
    };

    const getResultColor = (result: string) => {
      switch (result) {
        case 'RESULT_PASSED': return 'bg-green-100 text-green-800';
        case 'RESULT_FAILED': return 'bg-red-100 text-red-800';
        default: return 'bg-gray-100 text-gray-800';
      }
    };

    return (
      <div className="mt-2 flex flex-wrap gap-1 flex-col">
        <div className="flex justify-center items-center gap-2 w-full text-center">
          <span className="text-xs font-medium text-gray-500">Execution Status:</span>
          <span className={`text-xs px-2 py-1 rounded ${getStateColor(execution.state || '')}`}>
            {execution.state?.replace('STATE_', '')}
          </span>
          {execution.result && execution.result !== 'RESULT_UNKNOWN' && (
          <>
          <span className="text-xs font-medium text-gray-500">Execution Result:</span>
          <span className={`text-xs px-2 py-1 rounded ${getResultColor(execution.result)}`}>
            {execution.result?.replace('RESULT_', '')}
          </span>
          </>
          )}
        </div>
        
        <div className="flex justify-center items-center gap-2 w-full text-center">
        {execution.startedAt && (
          <span className="text-xs text-gray-500">
            Started: {new Date(execution.startedAt).toLocaleTimeString()}
          </span>
        )}
        {execution.finishedAt && (
          <span className="text-xs text-gray-500">
            Finished: {new Date(execution.finishedAt).toLocaleTimeString()}
          </span>
        )}
        </div>
      </div>
    );
  };

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
        <p className="text-gray-600">All events in the stage queue with inputs and outputs</p>
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
            <div className="space-y-3">
              {pendingEvents.map((event) => (
                <div key={event.id} className="bg-white rounded-lg p-4 border border-amber-200">
                  <div className="flex justify-between items-start mb-2">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        Event #{event.id?.substring(0, 8)}...
                      </div>
                      <div className="text-xs text-gray-500">
                        Created: {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'N/A'}
                      </div>
                      {event.stateReason && (
                        <div className="text-xs text-amber-600">
                          Reason: {event.stateReason.replace('STATE_REASON_', '')}
                        </div>
                      )}
                    </div>
                    <div className="text-xs bg-amber-100 text-amber-800 px-2 py-1 rounded">
                      PENDING
                    </div>
                  </div>
                  {renderEventInputs(event)}
                  {renderExecutionStatus(event)}
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
            <div className="space-y-3">
              {waitingEvents.map((event) => (
                <div key={event.id} className="bg-white rounded-lg p-4 border border-blue-200">
                  <div className="flex justify-between items-start mb-2">
                    <div className="flex-1">
                      <div className="text-sm font-medium text-gray-900">
                        Event #{event.id?.substring(0, 8)}...
                      </div>
                      <div className="text-xs text-gray-500">
                        Created: {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'N/A'}
                      </div>
                      {event.approvals && event.approvals.length > 0 && (
                        <div className="text-xs text-blue-600">
                          Approvals: {event.approvals.length}
                          {event.approvals.map((approval, index) => (
                            <div key={index} className="text-xs text-gray-500 ml-2">
                              ✓ {approval.approvedBy?.substring(0, 8)}... at {approval.approvedAt ? new Date(approval.approvedAt).toLocaleTimeString() : 'N/A'}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                    <button 
                      onClick={() => approveStageEvent(event.id!, selectedStage.id!)} 
                      style={{ backgroundColor: '#2563eb' }}
                      className="text-xs text-white px-3 py-1 rounded transition-colors bg-blue-600 hover:bg-blue-700"
                    >
                      Approve
                    </button>
                  </div>
                  {renderEventInputs(event)}
                  {renderExecutionStatus(event)}
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
            <div className="space-y-3 max-h-96 overflow-y-auto">
              {processedEvents.slice(0, 10).map((event) => (
                <div key={event.id} className="bg-white rounded-lg p-4 border border-green-200">
                  <div className="flex justify-between items-start mb-2">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        Event #{event.id?.substring(0, 8)}...
                      </div>
                      <div className="text-xs text-gray-500">
                        Processed: {event.createdAt ? new Date(event.createdAt).toLocaleString() : 'N/A'}
                      </div>
                    </div>
                    <div className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded">
                      PROCESSED
                    </div>
                  </div>
                  {renderEventInputs(event)}
                  {event.execution?.state === 'STATE_FINISHED' && renderExecutionOutputs(event)}
                  {renderExecutionStatus(event)}
                </div>
              ))}
              {processedEvents.length > 10 && (
                <div className="text-center text-xs text-gray-500 pt-2">
                  ... and {processedEvents.length - 10} more processed events
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};