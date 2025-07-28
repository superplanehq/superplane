import { SuperplaneValueDefinition } from "@/api-client";
import { StageWithEventQueue } from "../../store/types";
import { useState } from "react";

interface SettingsTabProps {
  selectedStage: StageWithEventQueue;
}

export const SettingsTab = ({ selectedStage }: SettingsTabProps) => {
  const [viewMode, setViewMode] = useState<'form' | 'yaml'>('form');

  const convertToYaml = (obj: StageWithEventQueue): string => {
    const yamlify = (value: string | number | boolean | object | null | undefined, indent: number = 0): string => {
      const spaces = '  '.repeat(indent);
      
      if (value === null || value === undefined) {
        return 'null';
      }
      
      if (typeof value === 'string') {
        return value.includes('\n') || value.includes(':') || value.includes('#') 
          ? `"${value.replace(/"/g, '\\"')}"` 
          : value;
      }
      
      if (typeof value === 'number' || typeof value === 'boolean') {
        return String(value);
      }
      
      if (Array.isArray(value)) {
        if (value.length === 0) return '[]';
        return '\n' + value.map(item => `${spaces}- ${yamlify(item, indent + 1)}`).join('\n');
      }
      
      if (typeof value === 'object') {
        const entries = Object.entries(value).filter(([, v]) => v !== undefined);
        if (entries.length === 0) return '{}';
        
        return '\n' + entries.map(([key, val]) => {
          const yamlValue = yamlify(val, indent + 1);
          return yamlValue.startsWith('\n') 
            ? `${spaces}${key}:${yamlValue}`
            : `${spaces}${key}: ${yamlValue}`;
        }).join('\n');
      }
      
      return String(value);
    };
    
    return yamlify(obj).trim();
  };

  const getAllInputMappings = (inputName: string) => {
    if (!selectedStage.spec!.inputMappings) return [];
    
    const mappings = [];
    for (const mapping of selectedStage.spec!.inputMappings) {
      const valueMappings = mapping.values?.filter(v => v.name === inputName) || [];
      for (const valueMapping of valueMappings) {
        mappings.push({
          mapping: valueMapping,
          triggeredBy: mapping.when?.triggeredBy?.connection || valueMapping.valueFrom?.eventData?.connection || 'Unknown'
        });
      }
    }
    return mappings;
  };

  const formatValueSource = (mapping: SuperplaneValueDefinition) => {
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
    <div className="h-full flex flex-col">
      {/* View Mode Toggle */}
      <div className="flex items-center justify-between border-b border-gray-200 pb-2 px-3 pt-3">
        <div className="flex items-center">
          <button
            className={`px-3 py-1 text-sm font-medium rounded-l border ${
              viewMode === 'form' 
                ? 'bg-gray-100 text-gray-900 border-gray-300 shadow-inner' 
                : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
            }`}
            onClick={() => setViewMode('form')}
          >
            Form
          </button>
          <button
            className={`px-3 py-1 text-sm font-medium rounded-r border-l-0 border ${
              viewMode === 'yaml' 
                ? 'bg-gray-100 text-gray-900 border-gray-300 shadow-inner' 
                : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
            }`}
            onClick={() => setViewMode('yaml')}
          >
            Yaml
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="bg-white flex-1 overflow-auto ml-2">
        {viewMode === 'form' ? (
          <div className='text-sm p-3'>
            {/* Stage Details */}
            <div className='mt-3 mb-2 uppercase text-sm font-bold text-gray-700 tracking-wide text-left'>
              Stage Details
            </div>
            <div className="space-y-2">
              <div className="flex items-start w-full">
                <div className='text-gray-600 w-1/4 text-left'>Name</div>
                <div className="block w-full font-mono text-sm text-left">{selectedStage.metadata!.name || '—'}</div>
              </div>
              <div className="flex items-start w-full">
                <div className='text-gray-600 w-1/4 text-left'>ID</div>
                <div className="block w-full font-mono text-sm text-left">{selectedStage.metadata!.id || '—'}</div>
              </div>
            </div>

            {/* Executor */}
            {selectedStage.spec!.executor && (
              <>
                <div className='mt-6 mb-2 uppercase text-sm font-bold text-gray-700 tracking-wide text-left'>
                  Executor
                </div>
                <div className="space-y-2">
                  <div className="flex items-start w-full">
                    <div className='text-gray-600 w-1/4 text-left'>Type</div>
                    <div className="block w-full text-left">{selectedStage.spec!.executor!.type}</div>
                  </div>
                  {selectedStage.spec!.executor!.spec && selectedStage.spec!.executor!.type == "semaphore" && (
                    <>
                      <div className="flex items-start w-full">
                        <div className='text-gray-600 w-1/4 text-left'>Project</div>
                        <div className="block w-full font-mono text-sm text-left">{selectedStage.spec!.executor!.resource!.name || '—'}</div>
                      </div>
                      <div className="flex items-start w-full">
                        <div className='text-gray-600 w-1/4 text-left'>Branch</div>
                        <div className="block w-full font-mono text-sm text-left">{selectedStage.spec!.executor!.spec.branch as string || '—'}</div>
                      </div>
                      <div className="flex items-start w-full">
                        <div className='text-gray-600 w-1/4 text-left'>Pipeline file</div>
                        <div className="block w-full font-mono text-sm text-left">{selectedStage.spec!.executor!.spec.pipelineFile as string || '—'}</div>
                      </div>
                    </>
                  )}
                  {selectedStage.spec!.executor!.spec && selectedStage.spec!.executor!.type == "http" && (
                    <>
                      <div className="flex items-start w-full">
                        <div className='text-gray-600 w-1/4 text-left'>URL</div>
                        <div className="block w-full font-mono text-sm text-left">{selectedStage.spec!.executor!.spec.url as string || '—'}</div>
                      </div>
                    </>
                  )}
                </div>
              </>
            )}

            {/* Gates */}
            {selectedStage.spec!.conditions && selectedStage.spec!.conditions.length > 0 && (
              <>
                <div className='mt-6 mb-2 uppercase text-sm font-bold text-gray-700 tracking-wide text-left'>
                  Gates
                </div>
                <div className="space-y-2">
                  {selectedStage.spec!.conditions.map((condition, index) => (
                    <div key={index} className="flex items-start w-full">
                      <div className='text-gray-600 w-1/4 text-left'>Manual approval</div>
                      <div className="block w-full text-left">
                        {condition.approval ? `Required: ${condition.approval.count} approvals` : 'Enabled'}
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* Connections */}
            {selectedStage.spec!.connections && selectedStage.spec!.connections.length > 0 && (
              <>
                <div className='mt-6 mb-2 uppercase text-sm font-bold text-gray-700 tracking-wide text-left'>
                  Connections
                </div>
                <div className="space-y-2">
                  {selectedStage.spec!.connections.map((connection, index) => {
                    const getConnectionTypeLabel = (type?: string) => {
                      switch (type) {
                        case 'TYPE_CONNECTION_GROUP':
                          return 'Connection Group';
                        case 'TYPE_STAGE':
                          return 'Stage';
                        case 'TYPE_EVENT_SOURCE':
                          return 'Event Source';
                        default:
                          return 'Stage';
                      }
                    };

                    return (
                      <div key={connection.name || `connection-${index + 1}`} className="flex items-center justify-between w-full">
                        <span className="bg-gray-100 h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono text-left">
                          <span className="material-symbols-outlined mr-1 text-sm">rocket_launch</span>
                          {connection.name || `Connection ${index + 1}`}
                        </span>
                        <span className='text-sm text-green-600 text-left'>{getConnectionTypeLabel(connection.type)}</span>
                      </div>
                    );
                  })}
                </div>
              </>
            )}

            {/* Inputs */}
            {selectedStage.spec!.inputs && selectedStage.spec!.inputs.length > 0 && (
              <>
                <div className='mt-6 mb-2 uppercase text-sm font-bold text-gray-700 tracking-wide text-left'>
                  Inputs
                </div>
                <div className="space-y-3">
                  {selectedStage.spec!.inputs.map((input, index) => {
                    const inputMappings = getAllInputMappings(input.name || '');
                    
                    return (
                      <div key={`input_${input.name}_${index + 1}`}>
                        <div className="flex items-start w-full mb-2">
                          <div className='text-gray-600 w-1/4 text-left'>{input.name || `Input ${index + 1}`}</div>
                          <div className="block w-full text-left">
                            {inputMappings.length > 0 ? (
                              <div className="space-y-1">
                                {inputMappings.map((inputMapping, mappingIndex) => {
                                  const valueSource = formatValueSource(inputMapping.mapping);
                                  return (
                                    <div key={`mapping_${index}_${mappingIndex}`} className="flex items-center w-full">
                                      <span className="bg-gray-100 h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono mr-1 text-left">
                                        <span className="material-symbols-outlined mr-1 text-sm">rocket_launch</span>
                                        {inputMapping.triggeredBy}
                                      </span>
                                      <span className="text-sm">.</span>
                                      <span className="bg-purple-100 h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono mx-1 text-left">
                                        outputs
                                      </span>
                                      <span className="text-sm">.</span>
                                      <span className="bg-yellow-100 h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono ml-1 text-left">
                                        {valueSource?.source || input.name || 'VALUE'}
                                      </span>
                                    </div>
                                  );
                                })}
                              </div>
                            ) : (
                              <span className="text-gray-500 text-sm text-left">No mappings configured</span>
                            )}
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </>
            )}

            {/* Outputs */}
            {selectedStage.spec!.outputs && selectedStage.spec!.outputs.length > 0 && (
              <>
                <div className='mt-6 mb-2 uppercase text-sm font-bold text-gray-700 tracking-wide text-left'>
                  Outputs
                </div>
                <div className="space-y-2">
                  {selectedStage.spec!.outputs.map((output, index) => (
                    <div key={index} className="flex items-center justify-between w-full">
                      <div className='flex items-center'>
                        <span className="h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono mr-1 text-left">this</span>
                        <span className="text-sm">.</span>
                        <span className="bg-purple-100 h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono mx-1 text-left">outputs</span>
                        <span className="text-sm">.</span>
                        <span className="bg-yellow-100 h-[26px] text-gray-600 text-sm px-2 py-1 rounded leading-none flex items-center border border-gray-200 font-mono ml-1 text-left">{output.name || `output_${index + 1}`}</span>
                      </div>
                      <span className={`text-sm text-left ${output.required ? 'text-red-600' : 'text-gray-500'}`}>
                        {output.required ? 'required' : 'optional'}
                      </span>
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        ) : (
          <div className="h-full">
            <pre className="bg-white p-4 font-mono text-sm h-full overflow-auto text-left">
              {convertToYaml(selectedStage)}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
};