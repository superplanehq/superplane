import { SuperplaneExecution } from "@/api-client";
import { formatRelativeTime, getExecutionStatusIcon, getExecutionStatusColor } from '../utils/stageEventUtils';

interface ExecutionTimelineProps {
  executions: SuperplaneExecution[];
  title?: string;
  showStats?: boolean;
}

export const ExecutionTimeline = ({ 
  executions, 
  title = "Recent Activity",
  showStats = false 
}: ExecutionTimelineProps) => {
  if (executions.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200">
        <div className="p-4 border-b border-gray-200">
          <h4 className="text-sm font-medium text-gray-700">{title}</h4>
        </div>
        <div className="p-4">
          <div className="text-center py-6 text-gray-500">
            <div className="text-4xl mb-2">📊</div>
            <div className="text-sm">No recent activity</div>
          </div>
        </div>
      </div>
    );
  }

  const passedCount = executions.filter(e => e.result === 'RESULT_PASSED').length;
  const failedCount = executions.filter(e => e.result === 'RESULT_FAILED').length;
  const pendingCount = executions.filter(e => e.state === 'STATE_PENDING').length;

  return (
    <>
      {showStats && (
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-green-600">{passedCount}</div>
            <div className="text-sm text-gray-500">Passed</div>
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-red-600">{failedCount}</div>
            <div className="text-sm text-gray-500">Failed</div>
          </div>
          <div className="bg-white rounded-lg border border-gray-200 p-4 text-center">
            <div className="text-2xl font-bold text-amber-600">{pendingCount}</div>
            <div className="text-sm text-gray-500">Pending</div>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg border border-gray-200">
        <div className="p-4 border-b border-gray-200">
          <h3 className="font-medium text-gray-900">{title}</h3>
        </div>
        <div className="p-4">
          <div className="space-y-4">
            {executions.map((execution, index) => (
              <div key={execution.id || index} className="flex items-start space-x-4">
                <div className="flex-shrink-0">
                  <span className="text-2xl">
                    {getExecutionStatusIcon(execution.state || '', execution.result)}
                  </span>
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
                  {!showStats && (
                    <div className={`inline-block text-xs px-2 py-1 rounded mt-2 ${getExecutionStatusColor(execution.state || '', execution.result)}`}>
                      {execution.result?.replace('RESULT_', '') || 'N/A'}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
};