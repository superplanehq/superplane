import { SuperplaneExecution } from "@/api-client";
import { ExecutionTimeline } from '../ExecutionTimeline';

interface HistoryTabProps {
  allExecutions: SuperplaneExecution[];
}

export const HistoryTab = ({ allExecutions }: HistoryTabProps) => {
  const getExecutionStatusStyle = (execution: SuperplaneExecution) => {
    if (execution.result === 'RESULT_PASSED') {
      return 'bg-green-50 border-green-200';
    } else if (execution.result === 'RESULT_FAILED') {
      return 'bg-red-50 border-red-200';
    } else if (execution.state === 'STATE_STARTED') {
      return 'bg-blue-50 border-blue-200';
    } else {
      return 'bg-gray-50 border-gray-200';
    }
  };

  const getResultBadgeStyle = (result: string) => {
    switch (result) {
      case 'RESULT_PASSED': return 'bg-green-100 text-green-800';
      case 'RESULT_FAILED': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getStateBadgeStyle = (state: string) => {
    switch (state) {
      case 'STATE_PENDING': return 'bg-yellow-100 text-yellow-800';
      case 'STATE_STARTED': return 'bg-blue-100 text-blue-800';
      case 'STATE_FINISHED': return 'bg-green-100 text-green-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getExecutionDuration = (execution: SuperplaneExecution) => {
    if (!execution.startedAt || !execution.finishedAt) return null;
    const start = new Date(execution.startedAt);
    const end = new Date(execution.finishedAt);
    const durationMs = end.getTime() - start.getTime();
    const seconds = Math.floor(durationMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-2">Run History</h2>
        <p className="text-gray-600">Complete execution history with inputs and outputs</p>
      </div>

      {/* Timeline Overview */}
      <div className="mb-8">
        <ExecutionTimeline 
          executions={allExecutions} 
          title="Execution Timeline"
          showStats={true}
        />
      </div>

      {/* Detailed Execution List */}
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-gray-900">Execution Details</h3>
        
        {allExecutions.length === 0 ? (
          <div className="text-center py-12 text-gray-500">
            <div className="text-6xl mb-4">📜</div>
            <div className="text-lg font-medium mb-2">No Execution History</div>
            <div className="text-sm">Executions will appear here after they run</div>
          </div>
        ) : (
          <div className="space-y-4">
            {allExecutions.map((execution) => (
              <div
                key={execution.id}
                className={`rounded-lg border p-6 ${getExecutionStatusStyle(execution)}`}
              >
                {/* Execution Header */}
                <div className="flex justify-between items-start mb-4">
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <h4 className="text-lg font-medium text-gray-900">
                        Execution #{execution.id?.substring(0, 8)}...
                      </h4>
                      <div className="flex gap-2">
                        <span className={`text-xs px-2 py-1 rounded ${getStateBadgeStyle(execution.state || '')}`}>
                          {execution.state?.replace('STATE_', '')}
                        </span>
                        {execution.result && execution.result !== 'RESULT_UNKNOWN' && (
                          <span className={`text-xs px-2 py-1 rounded ${getResultBadgeStyle(execution.result)}`}>
                            {execution.result?.replace('RESULT_', '')}
                          </span>
                        )}
                      </div>
                    </div>
                    
                    {/* Execution Timing */}
                    <div className="text-sm text-gray-600 space-y-1">
                      <div>Created: {execution.createdAt ? new Date(execution.createdAt).toLocaleString() : 'N/A'}</div>
                      {execution.startedAt && (
                        <div>Started: {new Date(execution.startedAt).toLocaleString()}</div>
                      )}
                      {execution.finishedAt && (
                        <div>Finished: {new Date(execution.finishedAt).toLocaleString()}</div>
                      )}
                      {getExecutionDuration(execution) && (
                        <div>Duration: <span className="font-mono">{getExecutionDuration(execution)}</span></div>
                      )}
                    </div>
                  </div>
                  
                  {execution.referenceId && (
                    <div className="text-right">
                      <div className="text-xs text-gray-500">Reference ID</div>
                      <div className="text-sm font-mono text-gray-700">{execution.referenceId}</div>
                    </div>
                  )}
                </div>

                {/* Execution Outputs */}
                {execution.outputs && execution.outputs.length > 0 && (
                  <div className="mt-4">
                    <h5 className="text-sm font-medium text-gray-900 mb-2">📤 Outputs</h5>
                    <div className="bg-white rounded-lg border p-3">
                      <div className="flex flex-col gap-2">
                        {execution.outputs.map((output, index) => (
                          <div key={index} className="bg-green-50 border border-green-200 rounded p-3">
                            <div className="text-sm font-medium text-green-900 mb-1">
                              {output.name}
                            </div>
                            <div className="text-sm font-mono text-green-800 bg-white px-2 py-1 rounded border">
                              {output.value || '<empty>'}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}

                {/* No Outputs Message */}
                {(!execution.outputs || execution.outputs.length === 0) && execution.state === 'STATE_FINISHED' && (
                  <div className="mt-4">
                    <h5 className="text-sm font-medium text-gray-900 mb-2">📤 Outputs</h5>
                    <div className="bg-white rounded-lg border p-3">
                      <div className="text-sm text-gray-500 text-center py-2">
                        No outputs produced by this execution
                      </div>
                    </div>
                  </div>
                )}

                {/* Execution Still Running */}
                {execution.state !== 'STATE_FINISHED' && (
                  <div className="mt-4">
                    <div className="bg-white rounded-lg border border-blue-200 p-3">
                      <div className="flex items-center gap-2 text-blue-700">
                        <div className="animate-spin rounded-full h-4 w-4 border-2 border-blue-700 border-t-transparent"></div>
                        <span className="text-sm">
                          {execution.state === 'STATE_PENDING' ? 'Waiting to start...' : 'Execution in progress...'}
                        </span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};