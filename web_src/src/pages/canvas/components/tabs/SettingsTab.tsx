import { InputMappingValueDefinition } from "@/api-client";
import { StageWithEventQueue } from "../../store/types";

interface SettingsTabProps {
  selectedStage: StageWithEventQueue;
}

export const SettingsTab = ({ selectedStage }: SettingsTabProps) => {
  const getAllInputMappings = (inputName: string) => {
    if (!selectedStage.inputMappings) return [];
    
    const mappings = [];
    for (const mapping of selectedStage.inputMappings) {
      const valueMappings = mapping.values?.filter(v => v.name === inputName) || [];
      for (const valueMapping of valueMappings) {
        mappings.push({
          mapping: valueMapping,
          triggeredBy: mapping.when?.triggeredBy?.connection || 'Unknown'
        });
      }
    }
    return mappings;
  };

  const formatValueSource = (mapping: InputMappingValueDefinition) => {
    if (mapping.value && mapping.value.trim() !== '') {
      return {
        type: 'Static Value',
        source: mapping.value,
        icon: '📝'
      };
    }
    
    if (mapping.valueFrom?.eventData?.connection) {
      return {
        type: 'From Connection',
        source: `${mapping.valueFrom.eventData.connection}${mapping.valueFrom.eventData.expression ? ` → ${mapping.valueFrom.eventData.expression}` : ''}`,
        icon: '🔗'
      };
    }
    
    if (mapping.valueFrom?.lastExecution?.results) {
      return {
        type: 'From Last Execution',
        source: `Results: ${mapping.valueFrom.lastExecution.results.join(', ')}`,
        icon: '⏮️'
      };
    }
    
    return null;
  };

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

      {/* Inputs */}
      <div className="bg-white rounded-lg border border-gray-200 mb-6">
        <div className="p-4 border-b border-gray-200">
          <h3 className="font-medium text-gray-900">Inputs</h3>
        </div>
        <div className="p-4">
          {selectedStage.inputs && selectedStage.inputs.length > 0 ? (
            <div className="space-y-4">
              {selectedStage.inputs.map((input, index) => {
                const inputMappings = getAllInputMappings(input.name || '');
                
                return (
                  <div key={`input_${input.name}_${index + 1}`} className="border border-gray-200 rounded-lg p-4">
                    <div className="mb-3">
                      <div className="font-medium text-gray-900 mb-1">
                        {input.name || `Input ${index + 1}`}
                      </div>
                      {input.description && (
                        <div className="text-sm text-gray-600 mb-2">{input.description}</div>
                      )}
                    </div>
                    
                    {/* Value Mappings */}
                    {inputMappings.length > 0 ? (
                      <div className="space-y-3">
                        {inputMappings.map((inputMapping, mappingIndex) => {
                          const valueSource = formatValueSource(inputMapping.mapping);
                          return (
                            <div key={`mapping_${index}_${mappingIndex}`} className="bg-gray-50 rounded-lg p-3 border">
                              <div className="flex items-start gap-2">
                                <span className="text-lg">{valueSource?.icon || '❓'}</span>
                                <div className="flex-1">
                                  <div className="flex items-center gap-2 mb-1">
                                    <div className="text-sm font-medium text-gray-700">
                                      {valueSource?.type || 'Unknown'}
                                    </div>
                                    <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded font-mono">
                                      from: {inputMapping.triggeredBy}
                                    </span>
                                  </div>
                                  <div className="text-sm text-gray-900 font-mono bg-white px-2 py-1 rounded border">
                                    {valueSource?.source || 'No source defined'}
                                  </div>
                                </div>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                        <div className="flex items-center gap-2">
                          <span className="text-yellow-600">⚠️</span>
                          <span className="text-sm text-yellow-800">No value mappings configured</span>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="text-center py-4 text-gray-500 text-sm">No inputs configured</div>
          )}
        </div>
      </div>

      {/* Outputs */}
      <div className="bg-white rounded-lg border border-gray-200 mb-6">
        <div className="p-4 border-b border-gray-200">
          <h3 className="font-medium text-gray-900">Outputs</h3>
        </div>
        <div className="p-4">
          {selectedStage.outputs && selectedStage.outputs.length > 0 ? (
            <div className="space-y-4">
              {selectedStage.outputs.map((output, index) => (
                <div key={index} className="border border-gray-200 rounded-lg p-4">
                  <div className="flex justify-between items-start mb-2">
                    <div className="font-medium text-gray-900">
                      {output.name || `Output ${index + 1}`}
                    </div>
                    <div className="flex gap-2">
                      {output.required && (
                        <span className="text-xs bg-red-100 text-red-700 px-2 py-1 rounded font-medium">
                          Required
                        </span>
                      )}
                    </div>
                  </div>
                  {output.description && (
                    <div className="text-sm text-gray-600 mb-3">{output.description}</div>
                  )}
                  
                  {/* Output Usage Info */}
                  <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
                    <div className="flex items-start gap-2">
                      <span className="text-blue-600">💡</span>
                      <div className="text-sm text-blue-800">
                        <div className="font-medium mb-1">Available for downstream stages</div>
                        <div className="text-xs text-blue-600 font-mono">
                          Reference: outputs.{output.name || `output_${index + 1}`}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-4 text-gray-500 text-sm">No outputs configured</div>
          )}
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
            <div className="text-center py-4 text-gray-500 text-sm">No executor configured</div>
          )}
        </div>
      </div>
    </div>
  );
};