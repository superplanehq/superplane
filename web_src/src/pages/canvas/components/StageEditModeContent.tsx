import { useState, useEffect } from 'react';
import { StageNodeType } from '@/canvas/types/flow';
import { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneValueDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneCondition, SuperplaneConditionType } from '@/api-client/types.gen';
import { useParams } from 'react-router-dom';
import { useSecrets } from '../hooks/useSecrets';
import { useIntegrations } from '../hooks/useIntegrations';
import { useEditModeState } from '../hooks/useEditModeState';
import { useArrayEditor } from '../hooks/useArrayEditor';
import { useConnectionManager } from '../hooks/useConnectionManager';
import { useValidation } from '../hooks/useValidation';
import { EditableAccordionSection } from './shared/EditableAccordionSection';
import { InlineEditor } from './shared/InlineEditor';
import { ValidationField } from './shared/ValidationField';
import { ConnectionSelector } from './shared/ConnectionSelector';
import { Field } from './Field';
import { Label } from './Label';

interface StageEditModeContentProps {
  data: StageNodeType['data'];
  currentStageId?: string;
  onDataChange?: (data: {
    label: string;
    description?: string;
    inputs: SuperplaneInputDefinition[];
    outputs: SuperplaneOutputDefinition[];
    connections: SuperplaneConnection[];
    executor: SuperplaneExecutor;
    secrets: SuperplaneValueDefinition[];
    conditions: SuperplaneCondition[];
    isValid: boolean
  }) => void;
}

export function StageEditModeContent({ data, currentStageId, onDataChange }: StageEditModeContentProps) {
  // Component-specific state
  const [inputs, setInputs] = useState<SuperplaneInputDefinition[]>(data.inputs || []);
  const [outputs, setOutputs] = useState<SuperplaneOutputDefinition[]>(data.outputs || []);
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [secrets, setSecrets] = useState<SuperplaneValueDefinition[]>(data.secrets || []);
  const [conditions, setConditions] = useState<SuperplaneCondition[]>(data.conditions || []);
  const [executor, setExecutor] = useState<SuperplaneExecutor>(data.executor || { type: '', spec: {} });
  const [inputMappings, setInputMappings] = useState<Record<number, SuperplaneValueDefinition[]>>({});
  const [responsePolicyStatusCodesDisplay, setResponsePolicyStatusCodesDisplay] = useState(
    ((executor.spec?.responsePolicy as Record<string, unknown>)?.statusCodes as number[] || []).join(', ')
  );
  const [semaphoreExecutionType, setSemaphoreExecutionType] = useState<'workflow' | 'task'>(
    (executor.spec?.task as string) ? 'task' : 'workflow'
  );

  // Validation
  const { validateName } = useValidation();

  // Get URL params and canvas data
  const { orgId, canvasId } = useParams<{ orgId: string, canvasId: string }>();
  const organizationId = orgId || '';

  // Fetch secrets and integrations
  const { data: canvasSecrets = [], isLoading: loadingCanvasSecrets } = useSecrets(canvasId!, "DOMAIN_TYPE_CANVAS");
  const { data: organizationSecrets = [], isLoading: loadingOrganizationSecrets } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const { data: canvasIntegrations = [] } = useIntegrations(canvasId!, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  // Helper functions
  const getAllSecrets = () => {
    const allSecrets: Array<{ name: string; source: 'Canvas' | 'Organization'; data: Record<string, string> }> = [];

    canvasSecrets.forEach(secret => {
      if (secret.metadata?.name) {
        allSecrets.push({
          name: secret.metadata.name,
          source: 'Canvas',
          data: secret.spec?.local?.data || {}
        });
      }
    });

    organizationSecrets.forEach(secret => {
      if (secret.metadata?.name) {
        allSecrets.push({
          name: secret.metadata.name,
          source: 'Organization',
          data: secret.spec?.local?.data || {}
        });
      }
    });

    return allSecrets;
  };

  const getAllIntegrations = () => [...canvasIntegrations, ...orgIntegrations];


  const getSecretKeys = (secretName: string) => {
    const allSecrets = getAllSecrets();
    const selectedSecret = allSecrets.find(secret => secret.name === secretName);
    return selectedSecret ? Object.keys(selectedSecret.data) : [];
  };

  // Validation functions
  const validateInput = (input: SuperplaneInputDefinition, index: number): string[] => {
    const nameErrors = validateName(input.name, inputs, index);
    return nameErrors;
  };

  const validateOutput = (output: SuperplaneOutputDefinition, index: number): string[] => {
    const nameErrors = validateName(output.name, outputs, index);
    return nameErrors;
  };

  const validateSecret = (secret: SuperplaneValueDefinition, index: number): string[] => {
    const errors: string[] = [];
    const nameErrors = validateName(secret.name, secrets, index);
    errors.push(...nameErrors);

    if (!secret.valueFrom && (!secret.value || secret.value.trim() === '')) {
      errors.push('Secret value is required');
    }

    if (secret.valueFrom && secret.valueFrom.secret) {
      if (!secret.valueFrom.secret.name || secret.valueFrom.secret.name.trim() === '') {
        errors.push('Secret reference name is required');
      }
      if (!secret.valueFrom.secret.key || secret.valueFrom.secret.key.trim() === '') {
        errors.push('Secret reference key is required');
      }
    }

    return errors;
  };

  const validateCondition = (condition: SuperplaneCondition): string[] => {
    const errors: string[] = [];
    if (!condition.type || condition.type === 'CONDITION_TYPE_UNKNOWN') {
      errors.push('Condition type is required');
    }

    if (condition.type === 'CONDITION_TYPE_APPROVAL') {
      if (!condition.approval || condition.approval.count === undefined || condition.approval.count < 1) {
        errors.push('Approval condition requires a count of at least 1');
      }
    } else if (condition.type === 'CONDITION_TYPE_TIME_WINDOW') {
      if (!condition.timeWindow) {
        errors.push('Time window condition requires time window configuration');
      } else {
        if (!condition.timeWindow.start || condition.timeWindow.start.trim() === '') {
          errors.push('Time window start time is required');
        }
        if (!condition.timeWindow.end || condition.timeWindow.end.trim() === '') {
          errors.push('Time window end time is required');
        }
        if (!condition.timeWindow.weekDays || condition.timeWindow.weekDays.length === 0) {
          errors.push('At least one weekday must be selected');
        }
      }
    }

    return errors;
  };

  const validateAllFields = () => {
    const errors: Record<string, string> = {};

    // Validate all arrays
    inputs.forEach((input, index) => {
      const inputErrors = validateInput(input, index);
      if (inputErrors.length > 0) {
        errors[`input_${index}`] = inputErrors.join(', ');
      }
    });

    outputs.forEach((output, index) => {
      const outputErrors = validateOutput(output, index);
      if (outputErrors.length > 0) {
        errors[`output_${index}`] = outputErrors.join(', ');
      }
    });

    connections.forEach((connection, index) => {
      const connectionErrors = connectionManager.validateConnection(connection);
      if (connectionErrors.length > 0) {
        errors[`connection_${index}`] = connectionErrors.join(', ');
      }
    });

    secrets.forEach((secret, index) => {
      const secretErrors = validateSecret(secret, index);
      if (secretErrors.length > 0) {
        errors[`secret_${index}`] = secretErrors.join(', ');
      }
    });

    conditions.forEach((condition, index) => {
      const conditionErrors = validateCondition(condition);
      if (conditionErrors.length > 0) {
        errors[`condition_${index}`] = conditionErrors.join(', ');
      }
    });

    // Validate executor
    if (executor.type && executor.type !== '') {
      if (!executor.spec || Object.keys(executor.spec).length === 0) {
        errors.executor = 'Executor specification is required when executor type is set';
      }
    }

    return Object.keys(errors).length === 0;
  };

  // Shared state management
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
      inputs: data.inputs || [],
      outputs: data.outputs || [],
      connections: data.connections || [],
      secrets: data.secrets || [],
      conditions: data.conditions || [],
      executor: data.executor || { type: '', spec: {} }
    },
    onDataChange,
    validateAllFields
  });

  // Initialize open sections
  useEffect(() => {
    setOpenSections(['general']);
  }, [setOpenSections]);

  // Connection management
  const connectionManager = useConnectionManager({
    connections,
    setConnections,
    currentEntityId: currentStageId
  });

  // Array editors
  const inputsEditor = useArrayEditor({
    items: inputs,
    setItems: setInputs,
    createNewItem: () => ({ name: '', description: '' }),
    validateItem: validateInput,
    setValidationErrors,
    errorPrefix: 'input'
  });

  const outputsEditor = useArrayEditor({
    items: outputs,
    setItems: setOutputs,
    createNewItem: () => ({ name: '', description: '', required: false }),
    validateItem: validateOutput,
    setValidationErrors,
    errorPrefix: 'output'
  });

  const connectionsEditor = useArrayEditor({
    items: connections,
    setItems: setConnections,
    createNewItem: () => ({ name: '', type: 'TYPE_EVENT_SOURCE' as SuperplaneConnection['type'], filters: [] }),
    validateItem: connectionManager.validateConnection,
    setValidationErrors,
    errorPrefix: 'connection'
  });

  const secretsEditor = useArrayEditor({
    items: secrets,
    setItems: setSecrets,
    createNewItem: () => ({ name: '', value: '' }),
    validateItem: validateSecret,
    setValidationErrors,
    errorPrefix: 'secret'
  });

  const conditionsEditor = useArrayEditor({
    items: conditions,
    setItems: setConditions,
    createNewItem: () => ({ type: 'CONDITION_TYPE_APPROVAL' as SuperplaneConditionType, approval: { count: 1 } }),
    validateItem: (item) => validateCondition(item),
    setValidationErrors,
    errorPrefix: 'condition'
  });

  // Sync component state with incoming data prop changes
  useEffect(() => {
    syncWithIncomingData(
      {
        inputs: data.inputs || [],
        outputs: data.outputs || [],
        connections: data.connections || [],
        secrets: data.secrets || [],
        conditions: data.conditions || [],
        executor: data.executor || { type: '', spec: {} }
      },
      (incomingData) => {
        setInputs(incomingData.inputs);
        setOutputs(incomingData.outputs);
        setConnections(incomingData.connections);
        setSecrets(incomingData.secrets);
        setConditions(incomingData.conditions);
        setExecutor(incomingData.executor);
        setResponsePolicyStatusCodesDisplay(
          ((incomingData.executor?.spec?.responsePolicy as Record<string, unknown>)?.statusCodes as number[] || []).join(', ')
        );
      }
    );
  }, [data, syncWithIncomingData]);

  // Notify parent of data changes
  useEffect(() => {
    if (onDataChange) {
      handleDataChange({
        label: data.label,
        description: data.description,
        inputs,
        outputs,
        connections,
        executor,
        secrets,
        conditions
      });
    }
  }, [data.label, data.description, inputs, outputs, connections, executor, secrets, conditions, onDataChange, handleDataChange]);

  // Revert function for each section
  const revertSection = (section: string) => {
    switch (section) {
      case 'connections':
        setConnections([...originalData.connections]);
        connectionsEditor.setEditingIndex(null);
        break;
      case 'inputs':
        setInputs([...originalData.inputs]);
        inputsEditor.setEditingIndex(null);
        break;
      case 'outputs':
        setOutputs([...originalData.outputs]);
        outputsEditor.setEditingIndex(null);
        break;
      case 'conditions':
        setConditions([...originalData.conditions]);
        conditionsEditor.setEditingIndex(null);
        break;
      case 'secrets':
        setSecrets([...originalData.secrets]);
        secretsEditor.setEditingIndex(null);
        break;
      case 'executor':
        setExecutor({ ...originalData.executor });
        break;
    }
  };

  // Helper functions for mappings and executor
  const addMapping = (inputIndex: number) => {
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: [...(prev[inputIndex] || []), { name: '', value: '' }]
    }));
  };

  const updateMapping = (inputIndex: number, mappingIndex: number, field: keyof SuperplaneValueDefinition, value: string) => {
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: prev[inputIndex]?.map((mapping, i) =>
        i === mappingIndex ? { ...mapping, [field]: value } : mapping
      ) || []
    }));
  };

  const removeMapping = (inputIndex: number, mappingIndex: number) => {
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: prev[inputIndex]?.filter((_, i) => i !== mappingIndex) || []
    }));
  };

  const updateSecretMode = (index: number, useValueFrom: boolean) => {
    secretsEditor.updateItem(index, 'value', useValueFrom ? undefined : '');
    secretsEditor.updateItem(index, 'valueFrom', useValueFrom ? { secret: { name: '', key: '' } } : undefined);
  };

  const updateConditionType = (index: number, type: SuperplaneConditionType) => {
    const newCondition: SuperplaneCondition = { type };
    if (type === 'CONDITION_TYPE_APPROVAL') {
      newCondition.approval = { count: 1 };
      newCondition.timeWindow = undefined;
    } else if (type === 'CONDITION_TYPE_TIME_WINDOW') {
      newCondition.timeWindow = { start: '', end: '', weekDays: [] };
      newCondition.approval = undefined;
    }
    setConditions(prev => prev.map((condition, i) => i === index ? newCondition : condition));
  };

  // Executor helper functions
  const updateExecutorField = (field: string, value: unknown) => {
    setExecutor(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        [field]: value
      }
    }));
  };

  const updateExecutorNestedField = (parentField: string, field: string, value: unknown) => {
    setExecutor(prev => ({
      ...prev,
      spec: {
        ...prev.spec,
        [parentField]: {
          ...(prev.spec?.[parentField] as Record<string, unknown> || {}),
          [field]: value
        }
      }
    }));
  };

  const addExecutorParameter = () => {
    const currentParams = (executor.spec?.parameters as Record<string, string>) || {};
    const newKey = `PARAM_${Object.keys(currentParams).length + 1}`;
    updateExecutorField('parameters', {
      ...currentParams,
      [newKey]: ''
    });
  };

  const updateExecutorParameter = (oldKey: string, newKey: string, value: string) => {
    const currentParams = (executor.spec?.parameters as Record<string, string>) || {};
    const updatedParams = { ...currentParams };

    if (oldKey !== newKey) {
      delete updatedParams[oldKey];
    }
    updatedParams[newKey] = value;

    updateExecutorField('parameters', updatedParams);
  };

  const removeExecutorParameter = (key: string) => {
    const currentParams = (executor.spec?.parameters as Record<string, string>) || {};
    const updatedParams = { ...currentParams };
    delete updatedParams[key];
    updateExecutorField('parameters', updatedParams);
  };

  const updateExecutorIntegration = (integrationName: string) => {
    const availableIntegrations = getAllIntegrations();
    const integration = availableIntegrations.find(int => int.metadata?.name === integrationName);
    if (integration) {
      setExecutor(prev => ({
        ...prev,
        integration: {
          name: integration.metadata?.name,
          domainType: integration.metadata?.domainType
        }
      }));
    }
  };

  const updateExecutorResource = (field: 'type' | 'name', value: string) => {
    setExecutor(prev => ({
      ...prev,
      resource: {
        ...prev.resource,
        [field]: value
      }
    }));
  };

  const updateSemaphoreExecutionType = (type: 'workflow' | 'task') => {
    setSemaphoreExecutionType(type);

    // If switching to workflow, clear the task field
    if (type === 'workflow') {
      updateExecutorField('task', '');
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
                  <span className="text-xs bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-300 px-2 py-0.5 rounded">
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
                    currentEntityId={currentStageId}
                    validationError={validationErrors[`connection_${index}`]}
                    showFilters={true}
                  />
                }
              />
            </div>
          ))}
          <button
            onClick={connectionsEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Connection
          </button>
        </EditableAccordionSection>

        {/* Inputs Section */}
        <EditableAccordionSection
          id="inputs"
          title="Inputs"
          isOpen={openSections.includes('inputs')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(inputs, 'inputs')}
          onRevert={revertSection}
          count={inputs.length}
          countLabel="inputs"
        >
          {inputs.map((input, index) => (
            <div key={index}>
              <InlineEditor
                isEditing={inputsEditor.editingIndex === index}
                onSave={inputsEditor.saveEdit}
                onCancel={() => inputsEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                onEdit={() => inputsEditor.startEdit(index)}
                onDelete={() => inputsEditor.removeItem(index)}
                displayName={input.name || `Input ${index + 1}`}
                badge={null}
                editForm={
                  <div className="space-y-3">
                    <ValidationField
                      label="Name"
                      error={validationErrors[`input_${index}`]}
                    >
                      <input
                        type="text"
                        value={input.name || ''}
                        onChange={(e) => inputsEditor.updateItem(index, 'name', e.target.value)}
                        placeholder="Input name"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`input_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>
                    <ValidationField label="Description">
                      <textarea
                        value={input.description || ''}
                        onChange={(e) => inputsEditor.updateItem(index, 'description', e.target.value)}
                        placeholder="Input description"
                        rows={2}
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </ValidationField>

                    {/* Mappings Section */}
                    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                      <div className="flex justify-between items-center mb-2">
                        <label className="text-sm font-medium">Mappings</label>
                        <button
                          onClick={() => addMapping(index)}
                          className="text-blue-600 hover:text-blue-700 text-sm"
                        >
                          + Add Mapping
                        </button>
                      </div>
                      <div className="space-y-2">
                        {(inputMappings[index] || []).map((mapping, mappingIndex) => (
                          <div key={mappingIndex} className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                            <select
                              value={mapping.name || ''}
                              onChange={(e) => updateMapping(index, mappingIndex, 'name', e.target.value)}
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            >
                              <option value="">Select connection</option>
                              {data.connections.map((conn, connIndex) => (
                                <option key={connIndex} value={conn.name}>{conn.name}</option>
                              ))}
                            </select>
                            <input
                              value={mapping.value || ''}
                              onChange={(e) => updateMapping(index, mappingIndex, 'value', e.target.value)}
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <button
                              onClick={() => removeMapping(index, mappingIndex)}
                              className="text-red-600 hover:text-red-700"
                            >
                              <span className="material-symbols-outlined text-sm">delete</span>
                            </button>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                }
              />
            </div>
          ))}
          <button
            onClick={inputsEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Input
          </button>
        </EditableAccordionSection>

        {/* Outputs Section */}
        <EditableAccordionSection
          id="outputs"
          title="Outputs"
          isOpen={openSections.includes('outputs')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(outputs, 'outputs')}
          onRevert={revertSection}
          count={outputs.length}
          countLabel="outputs"
        >
          {outputs.map((output, index) => (
            <div key={index}>
              <InlineEditor
                isEditing={outputsEditor.editingIndex === index}
                onSave={outputsEditor.saveEdit}
                onCancel={() => outputsEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                onEdit={() => outputsEditor.startEdit(index)}
                onDelete={() => outputsEditor.removeItem(index)}
                displayName={output.name || `Output ${index + 1}`}
                badge={output.required && (
                  <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">Required</span>
                )}
                editForm={
                  <div className="space-y-3">
                    <ValidationField
                      label="Name"
                      error={validationErrors[`output_${index}`]}
                    >
                      <input
                        type="text"
                        value={output.name || ''}
                        onChange={(e) => outputsEditor.updateItem(index, 'name', e.target.value)}
                        placeholder="Output name"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`output_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>
                    <ValidationField label="Description">
                      <textarea
                        value={output.description || ''}
                        onChange={(e) => outputsEditor.updateItem(index, 'description', e.target.value)}
                        placeholder="Output description"
                        rows={2}
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </ValidationField>
                    <ValidationField label="Required">
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={`required-${index}`}
                          checked={output.required || false}
                          onChange={(e) => outputsEditor.updateItem(index, 'required', e.target.checked)}
                          className="w-4 h-4 text-blue-600 bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600 rounded focus:ring-blue-500"
                        />
                        <label htmlFor={`required-${index}`}>Required</label>
                      </div>
                    </ValidationField>
                  </div>
                }
              />
            </div>
          ))}
          <button
            onClick={outputsEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Output
          </button>
        </EditableAccordionSection>

        {/* Conditions Section */}
        <EditableAccordionSection
          id="conditions"
          title="Conditions"
          isOpen={openSections.includes('conditions')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(conditions, 'conditions')}
          onRevert={revertSection}
          count={conditions.length}
          countLabel="conditions"
        >
          {conditions.map((condition, index) => (
            <div key={index} className="p-3 border border-zinc-200 dark:border-zinc-700 rounded-md bg-zinc-50 dark:bg-zinc-800">
              {conditionsEditor.editingIndex === index ? (
                <div className="space-y-3">
                  <ValidationField
                    label="Condition Type"
                    error={validationErrors[`condition_${index}`]}
                  >
                    <select
                      value={condition.type || 'CONDITION_TYPE_APPROVAL'}
                      onChange={(e) => updateConditionType(index, e.target.value as SuperplaneConditionType)}
                      className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`condition_${index}`]
                        ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                    >
                      <option value="CONDITION_TYPE_APPROVAL">Approval</option>
                      <option value="CONDITION_TYPE_TIME_WINDOW">Time Window</option>
                    </select>
                  </ValidationField>

                  {condition.type === 'CONDITION_TYPE_APPROVAL' && (
                    <ValidationField label="Required Approvals">
                      <input
                        type="number"
                        min="1"
                        value={condition.approval?.count || 1}
                        onChange={(e) => conditionsEditor.updateItem(index, 'approval', { count: parseInt(e.target.value) || 1 })}
                        placeholder="Number of required approvals"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </ValidationField>
                  )}

                  {condition.type === 'CONDITION_TYPE_TIME_WINDOW' && (
                    <div className="space-y-3">
                      <div className="grid grid-cols-2 gap-3">
                        <ValidationField label="Start Time">
                          <input
                            type="time"
                            value={condition.timeWindow?.start || ''}
                            onChange={(e) => conditionsEditor.updateItem(index, 'timeWindow', {
                              ...condition.timeWindow,
                              start: e.target.value
                            })}
                            className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </ValidationField>
                        <ValidationField label="End Time">
                          <input
                            type="time"
                            value={condition.timeWindow?.end || ''}
                            onChange={(e) => conditionsEditor.updateItem(index, 'timeWindow', {
                              ...condition.timeWindow,
                              end: e.target.value
                            })}
                            className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </ValidationField>
                      </div>
                      <ValidationField label="Days of Week">
                        <div className="grid grid-cols-7 gap-1">
                          {['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'].map((day) => (
                            <label key={day} className="flex flex-col items-center gap-1 p-2 border rounded text-xs">
                              <input
                                type="checkbox"
                                checked={condition.timeWindow?.weekDays?.includes(day) || false}
                                onChange={(e) => {
                                  const currentDays = condition.timeWindow?.weekDays || [];
                                  const newDays = e.target.checked
                                    ? [...currentDays, day]
                                    : currentDays.filter(d => d !== day);
                                  conditionsEditor.updateItem(index, 'timeWindow', {
                                    ...condition.timeWindow,
                                    weekDays: newDays
                                  });
                                }}
                                className="w-3 h-3"
                              />
                              <span>{day.slice(0, 3)}</span>
                            </label>
                          ))}
                        </div>
                      </ValidationField>
                    </div>
                  )}

                  <div className="flex items-center justify-between pt-2">
                    <button
                      onClick={() => conditionsEditor.saveEdit()}
                      className="text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                    >
                      <span className="material-symbols-outlined text-sm">check</span>
                    </button>
                    <button
                      onClick={() => conditionsEditor.removeItem(index)}
                      className="text-red-600 hover:text-red-700"
                    >
                      <span className="material-symbols-outlined text-sm">delete</span>
                    </button>
                  </div>
                </div>
              ) : (
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="font-medium text-sm">
                      {condition.type === 'CONDITION_TYPE_APPROVAL' && `Approval (${condition.approval?.count || 1} required)`}
                      {condition.type === 'CONDITION_TYPE_TIME_WINDOW' && `Time Window (${condition.timeWindow?.start || 'No start'} - ${condition.timeWindow?.end || 'No end'})`}
                    </div>
                    {condition.type === 'CONDITION_TYPE_TIME_WINDOW' && condition.timeWindow?.weekDays && (
                      <div className="text-xs text-zinc-500 mt-1">
                        {condition.timeWindow.weekDays.map(day => day.slice(0, 3)).join(', ')}
                      </div>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => conditionsEditor.startEdit(index)}
                      className="text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                    >
                      <span className="material-symbols-outlined text-sm">edit</span>
                    </button>
                    <button
                      onClick={() => conditionsEditor.removeItem(index)}
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
            onClick={conditionsEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Condition
          </button>
        </EditableAccordionSection>

        {/* Secrets Section */}
        <EditableAccordionSection
          id="secrets"
          title="Secrets Management"
          isOpen={openSections.includes('secrets')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(secrets, 'secrets')}
          onRevert={revertSection}
          count={secrets.length}
          countLabel="secrets"
        >
          {secrets.map((secret, index) => (
            <div key={index}>
              <InlineEditor
                isEditing={secretsEditor.editingIndex === index}
                onSave={secretsEditor.saveEdit}
                onCancel={() => secretsEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                onEdit={() => secretsEditor.startEdit(index)}
                onDelete={() => secretsEditor.removeItem(index)}
                displayName={secret.name || `Secret ${index + 1}`}
                badge={null}
                editForm={
                  <div className="space-y-3">
                    <ValidationField
                      label="Secret Name"
                      error={validationErrors[`secret_${index}`]}
                    >
                      <input
                        type="text"
                        value={secret.name || ''}
                        onChange={(e) => secretsEditor.updateItem(index, 'name', e.target.value)}
                        placeholder="Secret name"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField label="Value Source">
                      <div className="flex items-center gap-4 mb-2">
                        <div className="flex items-center gap-4">
                          <label className="flex items-center gap-2">
                            <input
                              type="radio"
                              name={`valueSource-${index}`}
                              checked={!secret.valueFrom}
                              onChange={() => updateSecretMode(index, false)}
                              className="w-4 h-4"
                            />
                            <span className="text-sm">Direct Value</span>
                          </label>
                          <label className="flex items-center gap-2">
                            <input
                              type="radio"
                              name={`valueSource-${index}`}
                              checked={!!secret.valueFrom}
                              onChange={() => updateSecretMode(index, true)}
                              className="w-4 h-4"
                            />
                            <span className="text-sm">From Secret</span>
                          </label>
                        </div>
                      </div>

                      {!secret.valueFrom ? (
                        <input
                          type="password"
                          value={secret.value || ''}
                          onChange={(e) => secretsEditor.updateItem(index, 'value', e.target.value)}
                          placeholder="Secret value"
                          className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                            ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                            : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                            }`}
                        />
                      ) : (
                        <div className="space-y-2">
                          <ValidationField label="Secret Name">
                            <select
                              value={secret.valueFrom?.secret?.name || ''}
                              onChange={(e) => {
                                const selectedSecretName = e.target.value;
                                secretsEditor.updateItem(index, 'valueFrom', {
                                  ...secret.valueFrom,
                                  secret: {
                                    ...secret.valueFrom?.secret,
                                    name: selectedSecretName,
                                    key: ''
                                  }
                                });
                              }}
                              className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                              disabled={loadingCanvasSecrets || loadingOrganizationSecrets}
                            >
                              <option value="">
                                {loadingCanvasSecrets || loadingOrganizationSecrets
                                  ? 'Loading secrets...'
                                  : 'Select a secret...'}
                              </option>
                              {(() => {
                                if (loadingCanvasSecrets || loadingOrganizationSecrets) {
                                  return null;
                                }

                                const allSecrets = getAllSecrets();
                                if (allSecrets.length === 0) {
                                  return (
                                    <option value="" disabled>
                                      No secrets available
                                    </option>
                                  );
                                }

                                const canvasSecretsFiltered = allSecrets.filter(s => s.source === 'Canvas');
                                const orgSecretsFiltered = allSecrets.filter(s => s.source === 'Organization');

                                return (
                                  <>
                                    {canvasSecretsFiltered.length > 0 && (
                                      <optgroup label="Canvas Secrets">
                                        {canvasSecretsFiltered.map(secretItem => (
                                          <option key={`canvas-${secretItem.name}`} value={secretItem.name}>
                                            {secretItem.name}
                                          </option>
                                        ))}
                                      </optgroup>
                                    )}
                                    {orgSecretsFiltered.length > 0 && (
                                      <optgroup label="Organization Secrets">
                                        {orgSecretsFiltered.map(secretItem => (
                                          <option key={`org-${secretItem.name}`} value={secretItem.name}>
                                            {secretItem.name}
                                          </option>
                                        ))}
                                      </optgroup>
                                    )}
                                  </>
                                );
                              })()}
                            </select>
                          </ValidationField>
                          <ValidationField label="Secret Key">
                            <select
                              value={secret.valueFrom?.secret?.key || ''}
                              onChange={(e) => secretsEditor.updateItem(index, 'valueFrom', {
                                ...secret.valueFrom,
                                secret: { ...secret.valueFrom?.secret, key: e.target.value }
                              })}
                              className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                              disabled={!secret.valueFrom?.secret?.name || loadingCanvasSecrets || loadingOrganizationSecrets}
                            >
                              <option value="">
                                {!secret.valueFrom?.secret?.name
                                  ? 'Select a secret first...'
                                  : 'Select a key...'}
                              </option>
                              {secret.valueFrom?.secret?.name && (() => {
                                const availableKeys = getSecretKeys(secret.valueFrom.secret.name);

                                if (availableKeys.length === 0) {
                                  return (
                                    <option value="" disabled>
                                      No keys available in this secret
                                    </option>
                                  );
                                }

                                return availableKeys.map(key => (
                                  <option key={key} value={key}>
                                    {key}
                                  </option>
                                ));
                              })()}
                            </select>
                          </ValidationField>
                        </div>
                      )}
                    </ValidationField>
                  </div>
                }
              />
            </div>
          ))}
          <button
            onClick={secretsEditor.addItem}
            className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
          >
            <span className="material-symbols-outlined text-sm">add</span>
            Add Secret
          </button>
        </EditableAccordionSection>

        {/* Executor Management Section */}
        <EditableAccordionSection
          id="executor"
          title="Executor Configuration"
          isOpen={openSections.includes('executor')}
          onToggle={handleAccordionToggle}
          isModified={isSectionModified(executor, 'executor')}
          onRevert={revertSection}
        >
          <div className="space-y-3">
            {executor.type === 'semaphore' && (
              <div className="space-y-4">
                <div className="text-xs text-zinc-500 mb-2">
                  Configure your Semaphore executor. You can use ${'{{ inputs.NAME }}'} and ${'{{ secrets.NAME }}'} syntax.
                </div>

                <Field>
                  <Label>Integration</Label>
                  <select
                    value={executor.integration?.name || ''}
                    onChange={(e) => updateExecutorIntegration(e.target.value)}
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  >
                    <option value="">Select an integration...</option>
                    {getAllIntegrations()
                      .filter(integration => integration.spec?.type === 'semaphore')
                      .map((integration) => (
                        <option key={integration.metadata?.id} value={integration.metadata?.name}>
                          {integration.metadata?.name}
                        </option>
                      ))}
                  </select>
                  {getAllIntegrations().filter(int => int.spec?.type === 'semaphore').length === 0 && (
                    <div className="text-xs text-zinc-500 mt-1">
                      No Semaphore integrations available. Create one in canvas settings.
                    </div>
                  )}
                </Field>

                <Field>
                  <Label>Project Name</Label>
                  <input
                    type="text"
                    value={(executor.resource?.name as string) || ''}
                    onChange={(e) => updateExecutorResource('name', e.target.value)}
                    placeholder="my-semaphore-project"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>

                <Field>
                  <Label>Execution Type</Label>
                  <div className="flex items-center gap-6 mt-2">
                    <label className="flex items-center gap-2">
                      <input
                        type="radio"
                        name="semaphore-execution-type"
                        value="workflow"
                        checked={semaphoreExecutionType === 'workflow'}
                        onChange={() => updateSemaphoreExecutionType('workflow')}
                        className="w-4 h-4 text-blue-600 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                      />
                      <span className="text-sm">Workflow</span>
                    </label>
                    <label className="flex items-center gap-2">
                      <input
                        type="radio"
                        name="semaphore-execution-type"
                        value="task"
                        checked={semaphoreExecutionType === 'task'}
                        onChange={() => updateSemaphoreExecutionType('task')}
                        className="w-4 h-4 text-blue-600 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                      />
                      <span className="text-sm">Task</span>
                    </label>
                  </div>
                  <div className="text-xs text-zinc-500 mt-1">
                    {semaphoreExecutionType === 'workflow'
                      ? 'Uses the workflows API to run a workflow'
                      : 'Uses the tasks API to run a specific task'}
                  </div>
                </Field>

                {semaphoreExecutionType === 'task' && (
                  <Field>
                    <Label>Task</Label>
                    <input
                      type="text"
                      value={(executor.spec?.task as string) || ''}
                      onChange={(e) => updateExecutorField('task', e.target.value)}
                      placeholder="my-task"
                      className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                    />
                  </Field>
                )}

                <Field>
                  <Label>Branch</Label>
                  <input
                    type="text"
                    value={(executor.spec?.branch as string) || ''}
                    onChange={(e) => updateExecutorField('branch', e.target.value)}
                    placeholder="main"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>

                <Field>
                  <Label>Pipeline File</Label>
                  <input
                    type="text"
                    value={(executor.spec?.pipelineFile as string) || ''}
                    onChange={(e) => updateExecutorField('pipelineFile', e.target.value)}
                    placeholder=".semaphore/pipeline.yml"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>

                <Field>
                  <div className="flex justify-between items-center mb-2">
                    <Label>Parameters</Label>
                    <button
                      onClick={addExecutorParameter}
                      className="text-blue-600 hover:text-blue-700 text-sm"
                    >
                      + Add Parameter
                    </button>
                  </div>
                  <div className="space-y-2">
                    {Object.entries((executor.spec?.parameters as Record<string, string>) || {}).map(([key, value]) => (
                      <div key={key} className="w-full flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                        <input
                          type="text"
                          value={key}
                          onChange={(e) => updateExecutorParameter(key, e.target.value, value)}
                          placeholder="Parameter name"
                          className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                        />
                        <input
                          type="text"
                          value={value}
                          onChange={(e) => updateExecutorParameter(key, key, e.target.value)}
                          placeholder="Parameter value"
                          className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                        />
                        <button
                          onClick={() => removeExecutorParameter(key)}
                          className="text-red-600 hover:text-red-700"
                        >
                          <span className="material-symbols-outlined text-sm">delete</span>
                        </button>
                      </div>
                    ))}
                  </div>
                </Field>
              </div>
            )}

            {executor.type === 'http' && (
              <div className="space-y-4">
                <div className="text-xs text-zinc-500 mb-2">
                  Configure your HTTP executor. You can use ${'{{ inputs.NAME }}'} and ${'{{ secrets.NAME }}'} syntax.
                </div>

                <Field>
                  <Label>URL</Label>
                  <input
                    type="text"
                    value={(executor.spec?.url as string) || ''}
                    onChange={(e) => updateExecutorField('url', e.target.value)}
                    placeholder="https://api.example.com/endpoint"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>

                <Field>
                  <Label>Payload (JSON)</Label>
                  <textarea
                    value={JSON.stringify(executor.spec?.payload || {}, null, 2)}
                    onChange={(e) => {
                      try {
                        const parsed = JSON.parse(e.target.value);
                        updateExecutorField('payload', parsed);
                      } catch {
                        // Invalid JSON, but still update the field to show user input
                        updateExecutorField('payload', e.target.value);
                      }
                    }}
                    placeholder='{\n  "key1": "value1",\n  "key2": "{{ inputs.KEY2 }}"\n}'
                    rows={6}
                    className="nodrag w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500 font-mono"
                  />
                </Field>

                <Field>
                  <div className="flex justify-between items-center mb-2">
                    <Label>Headers</Label>
                    <button
                      onClick={() => {
                        const currentHeaders = (executor.spec?.headers as Record<string, string>) || {};
                        const newKey = `Header_${Object.keys(currentHeaders).length + 1}`;
                        updateExecutorField('headers', {
                          ...currentHeaders,
                          [newKey]: ''
                        });
                      }}
                      className="text-blue-600 hover:text-blue-700 text-sm"
                    >
                      + Add Header
                    </button>
                  </div>
                  <div className="space-y-2">
                    {Object.entries((executor.spec?.headers as Record<string, string>) || {}).map(([key, value]) => (
                      <div key={key} className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                        <input
                          type="text"
                          value={key}
                          onChange={(e) => {
                            const currentHeaders = (executor.spec?.headers as Record<string, string>) || {};
                            const updatedHeaders = { ...currentHeaders };
                            if (e.target.value !== key) {
                              delete updatedHeaders[key];
                              updatedHeaders[e.target.value] = value;
                            }
                            updateExecutorField('headers', updatedHeaders);
                          }}
                          placeholder="Header name"
                          className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                        />
                        <input
                          type="text"
                          value={value}
                          onChange={(e) => {
                            const currentHeaders = (executor.spec?.headers as Record<string, string>) || {};
                            updateExecutorField('headers', {
                              ...currentHeaders,
                              [key]: e.target.value
                            });
                          }}
                          placeholder="Header value"
                          className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                        />
                        <button
                          onClick={() => {
                            const currentHeaders = (executor.spec?.headers as Record<string, string>) || {};
                            const updatedHeaders = { ...currentHeaders };
                            delete updatedHeaders[key];
                            updateExecutorField('headers', updatedHeaders);
                          }}
                          className="text-red-600 hover:text-red-700"
                        >
                          <span className="material-symbols-outlined text-sm">delete</span>
                        </button>
                      </div>
                    ))}
                  </div>
                </Field>

                <Field>
                  <Label>Response Policy - Success Status Codes</Label>
                  <input
                    type="text"
                    value={responsePolicyStatusCodesDisplay}
                    onChange={(e) => {
                      const endsWithComma = e.target.value.endsWith(',');
                      const codes = e.target.value.split(',').map(code => parseInt(code.trim())).filter(code => !isNaN(code));
                      updateExecutorNestedField('responsePolicy', 'statusCodes', codes);
                      setResponsePolicyStatusCodesDisplay(endsWithComma ? codes.join(',') + ',' : codes.join(','));
                    }}
                    placeholder="200, 201, 202"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>
              </div>
            )}

            {!executor.type && (
              <div className="text-sm text-zinc-500 bg-zinc-50 dark:bg-zinc-800 p-3 rounded-md">
                Select an executor type to configure how this stage will execute when triggered.
              </div>
            )}
          </div>
        </EditableAccordionSection>
      </div>
    </div>
  );
}