import { StageWithEventQueue } from "../../store/types";

interface SettingsTabProps {
  selectedStage: StageWithEventQueue;
}

export const SettingsTab = ({ selectedStage }: SettingsTabProps) => {
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

      {/* Executor */}
      <div className="bg-white rounded-lg border border-gray-200">
        <div className="p-4 border-b border-gray-200">
          <h3 className="font-medium text-gray-900">Executor</h3>
        </div>
        <div className="p-4">
          {selectedStage.executor ? (
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-gray-700 font-medium">Type</span>
                <span className="text-gray-900">{selectedStage.executor.type?.replace('TYPE_', '')}</span>
              </div>
              {selectedStage.executor.semaphore && (
                <>
                  <div className="flex justify-between items-center">
                    <span className="text-gray-700 font-medium">Project ID</span>
                    <span className="text-gray-900 font-mono text-sm">{selectedStage.executor.semaphore.projectId}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-gray-700 font-medium">Branch</span>
                    <span className="text-gray-900 font-mono text-sm">{selectedStage.executor.semaphore.branch}</span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-gray-700 font-medium">Pipeline File</span>
                    <span className="text-gray-900 font-mono text-sm">{selectedStage.executor.semaphore.pipelineFile}</span>
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
};