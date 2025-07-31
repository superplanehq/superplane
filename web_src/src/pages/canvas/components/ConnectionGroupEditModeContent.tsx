import React, { useState, useEffect } from 'react';
import { ConnectionGroupNodeType } from '@/canvas/types/flow';
import { SuperplaneConnection, SuperplaneConnectionType, GroupByField, SpecTimeoutBehavior } from '@/api-client/types.gen';
import { AccordionItem } from './AccordionItem';
import { Label } from './Label';
import { Field } from './Field';
import { Button } from '@/components/Button/button';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { RevertButton } from './RevertButton';
import { useCanvasStore } from '../store/canvasStore';

interface ConnectionGroupEditModeContentProps {
  data: ConnectionGroupNodeType['data'];
  currentConnectionGroupId?: string;
  onDataChange?: (data: { 
    name: string; 
    description?: string; 
    connections: SuperplaneConnection[]; 
    groupByFields: GroupByField[]; 
    timeout?: number; 
    timeoutBehavior?: SpecTimeoutBehavior;
    isValid: boolean;
  }) => void;
}

export function ConnectionGroupEditModeContent({ data, currentConnectionGroupId, onDataChange }: ConnectionGroupEditModeContentProps) {
  const [openSections, setOpenSections] = useState<string[]>(['general', 'connections', 'groupBy']);
  
  // Original data state for change tracking
  const [originalData] = useState({
    connections: data.connections || [],
    groupByFields: data.groupBy?.fields || [],
    timeout: data.timeout,
    timeoutBehavior: data.timeoutBehavior || 'TIMEOUT_BEHAVIOR_DROP'
  });
  
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [groupByFields, setGroupByFields] = useState<GroupByField[]>(data.groupBy?.fields || []);
  const [timeout, setTimeout] = useState<number | undefined>(data.timeout);
  const [timeoutBehavior, setTimeoutBehavior] = useState<SpecTimeoutBehavior>(data.timeoutBehavior || 'TIMEOUT_BEHAVIOR_DROP');
  const [editingConnectionIndex, setEditingConnectionIndex] = useState<number | null>(null);
  const [editingGroupByIndex, setEditingGroupByIndex] = useState<number | null>(null);

  // Validation states
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  // Get available connection sources from canvas store
  const { stages, eventSources, connectionGroups } = useCanvasStore();

  // Helper function to check if a section has been modified
  const isSectionModified = (section: string): boolean => {
    switch (section) {
      case 'connections':
        return JSON.stringify(connections) !== JSON.stringify(originalData.connections);
      case 'groupBy':
        return JSON.stringify(groupByFields) !== JSON.stringify(originalData.groupByFields);
      case 'timeout':
        return timeout !== originalData.timeout || timeoutBehavior !== originalData.timeoutBehavior;
      default:
        return false;
    }
  };

  // Revert function for each section
  const revertSection = (section: string) => {
    switch (section) {
      case 'connections':
        setConnections([...originalData.connections]);
        setEditingConnectionIndex(null);
        break;
      case 'groupBy':
        setGroupByFields([...originalData.groupByFields]);
        setEditingGroupByIndex(null);
        break;
      case 'timeout':
        setTimeout(originalData.timeout);
        setTimeoutBehavior(originalData.timeoutBehavior);
        break;
    }
  };

  const handleAccordionToggle = (sectionId: string) => {
    setOpenSections(prev => {
      return prev.includes(sectionId)
        ? prev.filter(id => id !== sectionId)
        : [...prev, sectionId]
    });
  };

  // Generate connection options based on connection type
  const getConnectionOptions = (connectionType: SuperplaneConnectionType | undefined) => {
    const options: Array<{ value: string; label: string; group: string }> = [];

    switch (connectionType) {
      case 'TYPE_STAGE':
        stages.forEach(stage => {
          if (stage.metadata?.name && stage.metadata?.id !== currentConnectionGroupId) {
            options.push({
              value: stage.metadata.name,
              label: stage.metadata.name,
              group: 'Stages'
            });
          }
        });
        break;

      case 'TYPE_EVENT_SOURCE':
        eventSources.forEach(eventSource => {
          if (eventSource.metadata?.name) {
            options.push({
              value: eventSource.metadata.name,
              label: eventSource.metadata.name,
              group: 'Event Sources'
            });
          }
        });
        break;

      case 'TYPE_CONNECTION_GROUP':
        connectionGroups.forEach(group => {
          if (group.metadata?.name && group.metadata?.id !== currentConnectionGroupId) {
            options.push({
              value: group.metadata.name,
              label: group.metadata.name,
              group: 'Connection Groups'
            });
          }
        });
        break;

      default:
        // If no type selected, show all available connections
        stages.forEach(stage => {
          if (stage.metadata?.name && stage.metadata?.id !== currentConnectionGroupId) {
            options.push({
              value: stage.metadata.name,
              label: stage.metadata.name,
              group: 'Stages'
            });
          }
        });
        eventSources.forEach(eventSource => {
          if (eventSource.metadata?.name) {
            options.push({
              value: eventSource.metadata.name,
              label: eventSource.metadata.name,
              group: 'Event Sources'
            });
          }
        });
        connectionGroups.forEach(group => {
          if (group.metadata?.name && group.metadata?.id !== currentConnectionGroupId) {
            options.push({
              value: group.metadata.name,
              label: group.metadata.name,
              group: 'Connection Groups'
            });
          }
        });
    }

    return options;
  };

  // Connection management
  const addConnection = () => {
    const newConnection: SuperplaneConnection = {
      name: '',
      type: 'TYPE_EVENT_SOURCE',
      filters: []
    };
    const newIndex = connections.length;
    setConnections(prev => [...prev, newConnection]);
    setEditingConnectionIndex(newIndex);
  };

  const updateConnection = (index: number, field: keyof SuperplaneConnection, value: SuperplaneConnectionType | string) => {
    setConnections(prev => prev.map((conn, i) => {
      if (i === index) {
        const updatedConnection = { ...conn, [field]: value };

        // If connection type changed, clear the connection name since available options will be different
        if (field === 'type' && updatedConnection.name) {
          const newOptions = getConnectionOptions(value as SuperplaneConnectionType);
          const isCurrentNameValid = newOptions.some(option => option.value === updatedConnection.name);
          if (!isCurrentNameValid) {
            updatedConnection.name = '';
          }
        }

        return updatedConnection;
      }
      return conn;
    }));
  };

  const removeConnection = (index: number) => {
    setConnections(prev => prev.filter((_, i) => i !== index));
    setEditingConnectionIndex(null);
  };

  const cancelEditConnection = (index: number) => {
    const connection = connections[index];
    // If this is a newly added connection with no name, remove it
    if (!connection.name || connection.name.trim() === '') {
      removeConnection(index);
    } else {
      setEditingConnectionIndex(null);
    }
  };

  // Group By field management
  const addGroupByField = () => {
    const newField: GroupByField = {
      name: '',
      expression: ''
    };
    const newIndex = groupByFields.length;
    setGroupByFields(prev => [...prev, newField]);
    setEditingGroupByIndex(newIndex);
  };

  const updateGroupByField = (index: number, field: keyof GroupByField, value: string) => {
    setGroupByFields(prev => prev.map((groupByField, i) =>
      i === index ? { ...groupByField, [field]: value } : groupByField
    ));
  };

  const removeGroupByField = (index: number) => {
    setGroupByFields(prev => prev.filter((_, i) => i !== index));
    setEditingGroupByIndex(null);
  };

  const cancelEditGroupByField = (index: number) => {
    const groupByField = groupByFields[index];
    // If this is a newly added field with no name, remove it
    if (!groupByField.name || groupByField.name.trim() === '') {
      removeGroupByField(index);
    } else {
      setEditingGroupByIndex(null);
    }
  };

  // Validation functions
  const validateConnection = (connection: SuperplaneConnection): string[] => {
    const errors: string[] = [];
    if (!connection.name || connection.name.trim() === '') {
      errors.push('Connection name is required');
    }
    if (!connection.type) {
      errors.push('Connection type is required');
    }
    return errors;
  };

  const validateGroupByField = (field: GroupByField): string[] => {
    const errors: string[] = [];
    if (!field.name || field.name.trim() === '') {
      errors.push('Field name is required');
    }
    if (!field.expression || field.expression.trim() === '') {
      errors.push('Field expression is required');
    }
    return errors;
  };

  const validateAllFields = React.useCallback((): boolean => {
    const errors: Record<string, string> = {};

    // Validate connections
    connections.forEach((connection, index) => {
      const connectionErrors = validateConnection(connection);
      if (connectionErrors.length > 0) {
        errors[`connection_${index}`] = connectionErrors.join(', ');
      }
    });

    // Validate group by fields
    groupByFields.forEach((field, index) => {
      const fieldErrors = validateGroupByField(field);
      if (fieldErrors.length > 0) {
        errors[`groupBy_${index}`] = fieldErrors.join(', ');
      }
    });

    // Validate that we have at least one connection
    if (connections.length === 0) {
      errors.connections = 'At least one connection is required';
    }

    // Validate that we have at least one group by field
    if (groupByFields.length === 0) {
      errors.groupByFields = 'At least one group by field is required';
    }

    // Validate timeout if provided
    if (timeout !== undefined && timeout < 0) {
      errors.timeout = 'Timeout must be a positive number';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  }, [connections, groupByFields, timeout]);

  // Update the onDataChange to include validation
  const handleDataChange = React.useCallback(() => {
    if (onDataChange) {
      const isValid = validateAllFields();
      onDataChange({
        name: data.name,
        description: data.description,
        connections,
        groupByFields,
        timeout,
        timeoutBehavior,
        isValid
      });
    }
  }, [data.name, data.description, connections, groupByFields, timeout, timeoutBehavior, onDataChange, validateAllFields]);

  // Notify parent of data changes with validation
  useEffect(() => {
    handleDataChange();
  }, [handleDataChange]);

  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      {/* Accordion Sections */}
      <div className="">

        {/* Connections Section */}
        <AccordionItem
          id="connections"
          title={
            <div className="flex items-center justify-between w-full">
              <div className="flex items-center gap-2">
                <span>Connections</span>
                <RevertButton 
                  sectionId="connections" 
                  isModified={isSectionModified('connections')} 
                  onRevert={revertSection} 
                />
              </div>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {connections.length} connections
              </span>
            </div>
          }
          isOpen={openSections.includes('connections')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {validationErrors.connections && (
              <div className="text-xs text-red-600 mb-2">
                {validationErrors.connections}
              </div>
            )}
            {connections.map((connection, index) => (
              <div key={index}>
                {editingConnectionIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Connection Type</Label>
                      <select
                        value={connection.type || 'TYPE_EVENT_SOURCE'}
                        onChange={(e) => updateConnection(index, 'type', e.target.value as SuperplaneConnectionType)}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`connection_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                      >
                        <option value="TYPE_EVENT_SOURCE">Event Source</option>
                        <option value="TYPE_STAGE">Stage</option>
                        <option value="TYPE_CONNECTION_GROUP">Connection Group</option>
                      </select>
                      {validationErrors[`connection_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`connection_${index}`]}
                        </div>
                      )}
                    </Field>
                    <Field>
                      <Label>Connection Name</Label>
                      <select
                        value={connection.name || ''}
                        onChange={(e) => updateConnection(index, 'name', e.target.value)}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`connection_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                      >
                        <option value="">
                          {connection.type ? 'Select a connection...' : 'Select connection type first'}
                        </option>
                        {(() => {
                          const options = getConnectionOptions(connection.type);

                          if (options.length === 0 && connection.type) {
                            return (
                              <option value="" disabled>
                                No {connection.type.replace('TYPE_', '').replace('_', ' ').toLowerCase()}s available
                              </option>
                            );
                          }

                          const groupedOptions: Record<string, typeof options> = {};

                          // Group options by their group property
                          options.forEach(option => {
                            if (!groupedOptions[option.group]) {
                              groupedOptions[option.group] = [];
                            }
                            groupedOptions[option.group].push(option);
                          });

                          return Object.entries(groupedOptions).map(([groupName, groupOptions]) => (
                            <optgroup key={groupName} label={groupName}>
                              {groupOptions.map(option => (
                                <option key={option.value} value={option.value}>
                                  {option.label}
                                </option>
                              ))}
                            </optgroup>
                          ));
                        })()}
                      </select>
                    </Field>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => cancelEditConnection(index)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingConnectionIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{connection.name || `Connection ${index + 1}`}</span>
                      {connection.type && (
                        <span className="text-xs bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-300 px-2 py-0.5 rounded">
                          {connection.type.replace('TYPE_', '').replace('_', ' ').toLowerCase()}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingConnectionIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeConnection(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <button
              onClick={addConnection}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Connection
            </button>
          </div>
        </AccordionItem>

        {/* Group By Fields Section */}
        <AccordionItem
          id="groupBy"
          title={
            <div className="flex items-center justify-between w-full">
              <div className="flex items-center gap-2">
                <span>Group By Fields</span>
                <RevertButton 
                  sectionId="groupBy" 
                  isModified={isSectionModified('groupBy')} 
                  onRevert={revertSection} 
                />
              </div>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {groupByFields.length} fields
              </span>
            </div>
          }
          isOpen={openSections.includes('groupBy')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {validationErrors.groupByFields && (
              <div className="text-xs text-red-600 mb-2">
                {validationErrors.groupByFields}
              </div>
            )}
            {groupByFields.map((field, index) => (
              <div key={index}>
                {editingGroupByIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Field Name</Label>
                      <input
                        type="text"
                        value={field.name || ''}
                        onChange={(e) => updateGroupByField(index, 'name', e.target.value)}
                        placeholder="Field name (e.g., version)"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`groupBy_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                      />
                      {validationErrors[`groupBy_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`groupBy_${index}`]}
                        </div>
                      )}
                    </Field>
                    <Field>
                      <Label>Expression</Label>
                      <input
                        type="text"
                        value={field.expression || ''}
                        onChange={(e) => updateGroupByField(index, 'expression', e.target.value)}
                        placeholder="Field expression (e.g., outputs.version)"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`groupBy_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                      />
                    </Field>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => cancelEditGroupByField(index)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingGroupByIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{field.name || `Field ${index + 1}`}</span>
                      {field.expression && (
                        <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded">
                          {field.expression}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingGroupByIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeGroupByField(index)}
                        className="text-red-600 hover:text-red-700"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))}
            <button
              onClick={addGroupByField}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Group By Field
            </button>
          </div>
        </AccordionItem>

        {/* Timeout Configuration Section */}
        <AccordionItem
          id="timeout"
          title={
            <div className="flex items-center justify-between w-full">
              <div className="flex items-center gap-2">
                <span>Timeout Configuration</span>
                <RevertButton 
                  sectionId="timeout" 
                  isModified={isSectionModified('timeout')} 
                  onRevert={revertSection} 
                />
              </div>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {timeout ? `${timeout}s` : 'No timeout'}
              </span>
            </div>
          }
          isOpen={openSections.includes('timeout')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-3">
            <Field>
              <Label>Timeout (seconds)</Label>
              <input
                type="number"
                min="0"
                value={timeout || ''}
                onChange={(e) => setTimeout(e.target.value ? parseInt(e.target.value) : undefined)}
                placeholder="No timeout (optional)"
                className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.timeout
                  ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                  : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                }`}
              />
              {validationErrors.timeout && (
                <div className="text-xs text-red-600 mt-1">
                  {validationErrors.timeout}
                </div>
              )}
              <div className="text-xs text-zinc-500 mt-1">
                How long to wait for all connections to send events with the same grouping fields
              </div>
            </Field>

            <Field>
              <Label>Timeout Behavior</Label>
              <select
                value={timeoutBehavior}
                onChange={(e) => setTimeoutBehavior(e.target.value as SpecTimeoutBehavior)}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
              >
                <option value="TIMEOUT_BEHAVIOR_DROP">Drop - Do not emit anything</option>
                <option value="TIMEOUT_BEHAVIOR_EMIT">Emit - Emit event with missing connections indicated</option>
              </select>
              <div className="text-xs text-zinc-500 mt-1">
                What to do when the timeout is reached
              </div>
            </Field>
          </div>
        </AccordionItem>
      </div>
    </div>
  );
}