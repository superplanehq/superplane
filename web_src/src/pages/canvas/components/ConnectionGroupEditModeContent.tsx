import { useState, useEffect } from 'react';
import { ConnectionGroupNodeType } from '@/canvas/types/flow';
import { SuperplaneConnection, GroupByField, SpecTimeoutBehavior } from '@/api-client/types.gen';
import { useEditModeState } from '../hooks/useEditModeState';
import { useArrayEditor } from '../hooks/useArrayEditor';
import { useConnectionManager } from '../hooks/useConnectionManager';
import { useValidation } from '../hooks/useValidation';
import { EditableAccordionSection } from './shared/EditableAccordionSection';
import { InlineEditor } from './shared/InlineEditor';
import { ValidationField } from './shared/ValidationField';
import { ConnectionSelector } from './shared/ConnectionSelector';

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
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [groupByFields, setGroupByFields] = useState<GroupByField[]>(data.groupBy?.fields || []);
  const [timeout, setTimeout] = useState<number | undefined>(undefined);
  const [timeoutBehavior, setTimeoutBehavior] = useState<SpecTimeoutBehavior>('TIMEOUT_BEHAVIOR_DROP');

  useValidation();

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

  const validateAllFields = () => {
    const errors: Record<string, string> = {};


    connections.forEach((connection, index) => {
      const connectionErrors = validateConnection(connection);
      if (connectionErrors.length > 0) {
        errors[`connection_${index}`] = connectionErrors.join(', ');
      }
    });

    groupByFields.forEach((field, index) => {
      const fieldErrors = validateGroupByField(field);
      if (fieldErrors.length > 0) {
        errors[`groupBy_${index}`] = fieldErrors.join(', ');
      }
    });

    if (connections.length === 0) {
      errors.connections = 'At least one connection is required';
    }


    if (groupByFields.length === 0) {
      errors.groupByFields = 'At least one group by field is required';
    }

    if (timeout !== undefined && timeout < 0) {
      errors.timeout = 'Timeout must be a positive number';
    }

    return Object.keys(errors).length === 0;
  };

  const {
    openSections,
    setOpenSections,
    originalData,
    validationErrors,
    setValidationErrors,
    handleAccordionToggle,
    isSectionModified,
    handleDataChange,
    syncWithIncomingData
  } = useEditModeState({
    initialData: {
      name: data.name || '',
      description: data.description,
      connections: data.connections || [],
      groupByFields: data.groupBy?.fields || [],
      timeout: undefined,
      timeoutBehavior: 'TIMEOUT_BEHAVIOR_DROP',
      isValid: true
    },
    onDataChange,
    validateAllFields
  });


  useEffect(() => {
    setOpenSections(['general', 'connections', 'groupBy']);
  }, [setOpenSections]);


  const connectionManager = useConnectionManager({
    connections,
    setConnections,
    currentEntityId: currentConnectionGroupId
  });


  const connectionsEditor = useArrayEditor({
    items: connections,
    setItems: setConnections,
    createNewItem: () => ({
      name: '',
      type: 'TYPE_EVENT_SOURCE' as SuperplaneConnection['type'],
      filters: []
    }),
    validateItem: validateConnection,
    setValidationErrors,
    errorPrefix: 'connection'
  });

  const groupByEditor = useArrayEditor({
    items: groupByFields,
    setItems: setGroupByFields,
    createNewItem: () => ({
      name: '',
      expression: ''
    }),
    validateItem: validateGroupByField,
    setValidationErrors,
    errorPrefix: 'groupBy'
  });


  useEffect(() => {
    syncWithIncomingData(
      {
        name: data.name || '',
        description: data.description,
        connections: data.connections || [],
        groupByFields: data.groupBy?.fields || [],
        timeout: undefined,
        timeoutBehavior: 'TIMEOUT_BEHAVIOR_DROP',
        isValid: true
      },
      (incomingData) => {
        setConnections(incomingData.connections);
        setGroupByFields(incomingData.groupByFields);
        setTimeout(incomingData.timeout);
        setTimeoutBehavior(incomingData.timeoutBehavior as SpecTimeoutBehavior);
      }
    );
  }, [data, syncWithIncomingData]);


  useEffect(() => {
    if (onDataChange) {
      handleDataChange({
        name: data.name,
        description: data.description,
        connections,
        groupByFields,
        timeout,
        timeoutBehavior
      });
    }
  }, [data.name, data.description, connections, groupByFields, timeout, timeoutBehavior, onDataChange, handleDataChange]);


  const revertSection = (section: string) => {
    switch (section) {
      case 'connections':
        setConnections([...originalData.connections]);
        connectionsEditor.setEditingIndex(null);
        break;
      case 'groupBy':
        setGroupByFields([...originalData.groupByFields]);
        groupByEditor.setEditingIndex(null);
        break;
      case 'timeout':
        setTimeout(originalData.timeout);
        setTimeoutBehavior(originalData.timeoutBehavior as SpecTimeoutBehavior);
        break;
    }
  };

  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      <div className="">
        {/* Connections Section */}
        <EditableAccordionSection
          id="connections"
          title="Connections"
          isOpen={openSections.includes('connections')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(connections, 'connections')}
          onRevert={revertSection}
          count={connections.length}
          countLabel="connections"
          validationError={validationErrors.connections}
        >
          {connections.map((connection, index) => (
            <div key={index}>
              <InlineEditor
                isEditing={connectionsEditor.editingIndex === index}
                onSave={connectionsEditor.saveEdit}
                onCancel={() => connectionsEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                onEdit={() => connectionsEditor.startEdit(index)}
                onDelete={() => connectionsEditor.removeItem(index)}
                displayName={connection.name || `Connection ${index + 1}`}
                badge={connection.type && (
                  <span className="text-xs bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-400 dark:text-zinc-300 px-2 py-0.5 rounded">
                    {connection.type.replace('TYPE_', '').replace('_', ' ').toLowerCase()}
                  </span>
                )}
                editForm={
                  <ConnectionSelector
                    connection={connection}
                    index={index}
                    onConnectionUpdate={connectionManager.updateConnection}
                    onFilterAdd={connectionManager.addFilter}
                    onFilterUpdate={connectionManager.updateFilter}
                    onFilterRemove={connectionManager.removeFilter}
                    onFilterOperatorToggle={connectionManager.toggleFilterOperator}
                    currentEntityId={currentConnectionGroupId}
                    validationError={validationErrors[`connection_${index}`]}
                    showFilters={false}
                  />
                }
              />
            </div>
          ))}
          <button
            onClick={connectionsEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Connection
          </button>
        </EditableAccordionSection>

        {/* Group By Fields Section */}
        <EditableAccordionSection
          id="groupBy"
          title="Group By Fields"
          isOpen={openSections.includes('groupBy')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(groupByFields, 'groupByFields')}
          onRevert={revertSection}
          count={groupByFields.length}
          countLabel="fields"
          validationError={validationErrors.groupByFields}
        >
          {groupByFields.map((field, index) => (
            <div key={index}>
              <InlineEditor
                isEditing={groupByEditor.editingIndex === index}
                onSave={groupByEditor.saveEdit}
                onCancel={() => groupByEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                onEdit={() => groupByEditor.startEdit(index)}
                onDelete={() => groupByEditor.removeItem(index)}
                displayName={field.name || `Field ${index + 1}`}
                badge={field.expression && (
                  <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded">
                    {field.expression}
                  </span>
                )}
                editForm={
                  <div className="space-y-3">
                    <ValidationField
                      label="Field Name"
                      error={validationErrors[`groupBy_${index}`]}
                    >
                      <input
                        type="text"
                        value={field.name || ''}
                        onChange={(e) => groupByEditor.updateItem(index, 'name', e.target.value)}
                        placeholder="Field name (e.g., version)"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`groupBy_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>
                    <ValidationField label="Expression">
                      <input
                        type="text"
                        value={field.expression || ''}
                        onChange={(e) => groupByEditor.updateItem(index, 'expression', e.target.value)}
                        placeholder="Field expression (e.g., outputs.version)"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`groupBy_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>
                  </div>
                }
              />
            </div>
          ))}
          <button
            onClick={groupByEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Group By Field
          </button>
        </EditableAccordionSection>

        {/* Timeout Configuration Section */}
        <EditableAccordionSection
          id="timeout"
          title="Timeout Configuration"
          isOpen={openSections.includes('timeout')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified({ timeout, timeoutBehavior }, 'timeout')}
          onRevert={revertSection}
        >
          <div className="space-y-3">
            <ValidationField
              label="Timeout (seconds)"
              error={validationErrors.timeout}
            >
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
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                How long to wait for all connections to send events with the same grouping fields
              </div>
            </ValidationField>

            <ValidationField label="Timeout Behavior">
              <select
                value={timeoutBehavior}
                onChange={(e) => setTimeoutBehavior(e.target.value as SpecTimeoutBehavior)}
                className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
              >
                <option value="TIMEOUT_BEHAVIOR_DROP">Drop - Do not emit anything</option>
                <option value="TIMEOUT_BEHAVIOR_EMIT">Emit - Emit event with missing connections indicated</option>
              </select>
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                What to do when the timeout is reached
              </div>
            </ValidationField>
          </div>
        </EditableAccordionSection>
      </div>
    </div>
  );
}