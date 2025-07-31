import React, { useState, useEffect } from 'react';
import { StageNodeType } from '@/canvas/types/flow';
import { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneValueDefinition, SuperplaneConnection, SuperplaneFilter, SuperplaneFilterOperator, SuperplaneFilterType, SuperplaneConnectionType, SuperplaneValueFrom, SuperplaneExecutor, SuperplaneCondition, SuperplaneConditionType } from '@/api-client/types.gen';
import { AccordionItem } from './AccordionItem';
import { Label } from './Label';
import { Field } from './Field';
import { Button } from '@/components/Button/button';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { useCanvasStore } from '../store/canvasStore';
import { useSecrets } from '../hooks/useSecrets';
import { useIntegrations } from '../hooks/useIntegrations';
import { useParams } from 'react-router-dom';

interface EditModeContentProps {
  data: StageNodeType['data'];
  currentStageId?: string;
  onDataChange?: (data: { label: string; description?: string; inputs: SuperplaneInputDefinition[]; outputs: SuperplaneOutputDefinition[]; connections: SuperplaneConnection[]; executor: SuperplaneExecutor; secrets: SuperplaneValueDefinition[]; conditions: SuperplaneCondition[]; isValid: boolean }) => void;
}

export function EditModeContent({ data, currentStageId, onDataChange }: EditModeContentProps) {
  const [openSections, setOpenSections] = useState<string[]>(['general']);
  const [inputs, setInputs] = useState<SuperplaneInputDefinition[]>(data.inputs || []);
  const [outputs, setOutputs] = useState<SuperplaneOutputDefinition[]>(data.outputs || []);
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [secrets, setSecrets] = useState<SuperplaneValueDefinition[]>(data.secrets || []);
  const [conditions, setConditions] = useState<SuperplaneCondition[]>(data.conditions || []);
  const [executor, setExecutor] = useState<SuperplaneExecutor>(data.executor || { type: '', spec: {} });
  const [editingInputIndex, setEditingInputIndex] = useState<number | null>(null);
  const [editingOutputIndex, setEditingOutputIndex] = useState<number | null>(null);
  const [editingConnectionIndex, setEditingConnectionIndex] = useState<number | null>(null);
  const [editingSecretIndex, setEditingSecretIndex] = useState<number | null>(null);
  const [editingConditionIndex, setEditingConditionIndex] = useState<number | null>(null);
  const [inputMappings, setInputMappings] = useState<Record<number, SuperplaneValueDefinition[]>>({});

  // Validation states
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  // Get available connection sources from canvas store
  const { stages, eventSources, connectionGroups } = useCanvasStore();

  // Get organization ID from URL params
  const { orgId, canvasId } = useParams<{ orgId: string, canvasId: string }>();

  // Get canvas ID and organization ID for secrets
  const organizationId = orgId || '';

  // Fetch secrets from both canvas and organization levels
  const { data: canvasSecrets = [], isLoading: loadingCanvasSecrets } = useSecrets(canvasId!, "DOMAIN_TYPE_CANVAS");
  const { data: organizationSecrets = [], isLoading: loadingOrganizationSecrets } = useSecrets(
    organizationId,
    "DOMAIN_TYPE_ORGANIZATION"
  );

  // Fetch integrations from both canvas and organization levels
  const { data: canvasIntegrations = [] } = useIntegrations(canvasId!, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  // Helper function to get all available secrets (canvas + organization)
  const getAllSecrets = (): Array<{ name: string; source: 'Canvas' | 'Organization'; data: Record<string, string> }> => {
    const allSecrets: Array<{ name: string; source: 'Canvas' | 'Organization'; data: Record<string, string> }> = [];

    // Add canvas secrets
    canvasSecrets.forEach(secret => {
      if (secret.metadata?.name) {
        allSecrets.push({
          name: secret.metadata.name,
          source: 'Canvas',
          data: secret.spec?.local?.data || {}
        });
      }
    });

    // Add organization secrets
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

  // Helper function to get all available integrations (canvas + organization)
  const getAllIntegrations = () => {
    return [...canvasIntegrations, ...orgIntegrations];
  };

  // Helper function to get available keys for a selected secret
  const getSecretKeys = (secretName: string) => {
    const allSecrets = getAllSecrets();
    const selectedSecret = allSecrets.find(secret => secret.name === secretName);
    return selectedSecret ? Object.keys(selectedSecret.data) : [];
  };

  // Helper function to get outputs from a selected connection
  const getOutputsFromConnection = (connectionName: string) => {
    if (!connectionName) return [];

    // Find the stage that matches the connection name
    const connectedStage = stages.find(stage => stage.metadata?.name === connectionName);

    if (connectedStage && connectedStage.spec?.outputs) {
      return connectedStage.spec.outputs;
    }

    // For event sources and connection groups, they typically don't have outputs
    // but if they do, we can add logic here later
    return [];
  };


  // Generate connection options based on connection type
  const getConnectionOptions = (connectionType: SuperplaneConnectionType | undefined) => {
    const options: Array<{ value: string; label: string; group: string }> = [];

    switch (connectionType) {
      case 'TYPE_STAGE':
        stages.forEach(stage => {
          if (stage.metadata?.name && stage.metadata?.id !== currentStageId) { // Exclude current stage
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
          if (group.metadata?.name) {
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
          if (stage.metadata?.name && stage.metadata?.id !== currentStageId) {
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
          if (group.metadata?.name) {
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

  const handleAccordionToggle = (sectionId: string) => {
    setOpenSections(prev => {
      return prev.includes(sectionId)
        ? prev.filter(id => id !== sectionId)
        : [...prev, sectionId]
    });
  };

  const addInput = () => {
    const newInput: SuperplaneInputDefinition = {
      name: '',
      description: ''
    };
    const newIndex = inputs.length;
    setInputs(prev => [...prev, newInput]);
    setEditingInputIndex(newIndex);
  };

  const updateInput = (index: number, field: keyof SuperplaneInputDefinition, value: string) => {
    setInputs(prev => prev.map((input, i) =>
      i === index ? { ...input, [field]: value } : input
    ));
    // Validate the input field after update
    setTimeout(() => validateInputField(index), 200);
  };

  const removeInput = (index: number) => {
    setInputs(prev => prev.filter((_, i) => i !== index));
    setEditingInputIndex(null);
  };

  const cancelEditInput = (index: number) => {
    const input = inputs[index];
    // If this is a newly added input with no name, remove it
    if (!input.name || input.name.trim() === '') {
      removeInput(index);
    } else {
      setEditingInputIndex(null);
    }
  };

  const validateInputField = (index: number) => {
    const input = inputs[index];
    const errors = validateInput(input, index);
    if (errors.length > 0) {
      setValidationErrors(prev => ({
        ...prev,
        [`input_${index}`]: errors.join(', ')
      }));
    } else {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[`input_${index}`];
        return newErrors;
      });
    }
  };

  const validateOutputField = (index: number) => {
    const output = outputs[index];
    const errors = validateOutput(output, index);
    if (errors.length > 0) {
      setValidationErrors(prev => ({
        ...prev,
        [`output_${index}`]: errors.join(', ')
      }));
    } else {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[`output_${index}`];
        return newErrors;
      });
    }
  };

  const validateSecretField = (index: number) => {
    const secret = secrets[index];
    const errors = validateSecret(secret, index);
    if (errors.length > 0) {
      setValidationErrors(prev => ({
        ...prev,
        [`secret_${index}`]: errors.join(', ')
      }));
    } else {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[`secret_${index}`];
        return newErrors;
      });
    }
  };

  const addOutput = () => {
    const newOutput: SuperplaneOutputDefinition = {
      name: '',
      description: '',
      required: false
    };
    const newIndex = outputs.length;
    setOutputs(prev => [...prev, newOutput]);
    setEditingOutputIndex(newIndex);
  };

  const updateOutput = (index: number, field: keyof SuperplaneOutputDefinition, value: string | boolean) => {
    setOutputs(prev => prev.map((output, i) =>
      i === index ? { ...output, [field]: value } : output
    ));
    // Validate the output field after update
    setTimeout(() => validateOutputField(index), 0);
  };

  const removeOutput = (index: number) => {
    setOutputs(prev => prev.filter((_, i) => i !== index));
    setEditingOutputIndex(null);
  };

  const cancelEditOutput = (index: number) => {
    const output = outputs[index];
    // If this is a newly added output with no name, remove it
    if (!output.name || output.name.trim() === '') {
      removeOutput(index);
    } else {
      setEditingOutputIndex(null);
    }
  };

  const addMapping = (inputIndex: number) => {
    const newMapping: SuperplaneValueDefinition = {
      name: '',
      value: ''
    };
    setInputMappings(prev => ({
      ...prev,
      [inputIndex]: [...(prev[inputIndex] || []), newMapping]
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

  const updateConnection = (index: number, field: keyof SuperplaneConnection, value: SuperplaneConnectionType | SuperplaneFilterOperator | string) => {
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

  const addFilter = (connectionIndex: number) => {
    const newFilter: SuperplaneFilter = {
      type: 'FILTER_TYPE_DATA',
      data: { expression: '' }
    };

    setConnections(prev => prev.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: [...(conn.filters || []), newFilter]
      } : conn
    ));
  };

  const updateFilter = (connectionIndex: number, filterIndex: number, updates: Partial<SuperplaneFilter>) => {
    setConnections(prev => prev.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: conn.filters?.map((filter, j) =>
          j === filterIndex ? { ...filter, ...updates } : filter
        )
      } : conn
    ));
  };

  const removeFilter = (connectionIndex: number, filterIndex: number) => {
    setConnections(prev => prev.map((conn, i) =>
      i === connectionIndex ? {
        ...conn,
        filters: conn.filters?.filter((_, j) => j !== filterIndex)
      } : conn
    ));
  };

  const toggleFilterOperator = (connectionIndex: number) => {
    const current = connections[connectionIndex]?.filterOperator || 'FILTER_OPERATOR_AND';
    const newOperator: SuperplaneFilterOperator =
      current === 'FILTER_OPERATOR_AND' ? 'FILTER_OPERATOR_OR' : 'FILTER_OPERATOR_AND';

    updateConnection(connectionIndex, 'filterOperator', newOperator);
  };

  // Secret management
  const addSecret = () => {
    const newSecret: SuperplaneValueDefinition = {
      name: '',
      value: ''
    };
    const newIndex = secrets.length;
    setSecrets(prev => [...prev, newSecret]);
    setEditingSecretIndex(newIndex);
  };

  const updateSecret = (index: number, field: keyof SuperplaneValueDefinition, value: string | SuperplaneValueFrom) => {
    setSecrets(prev => prev.map((secret, i) =>
      i === index ? { ...secret, [field]: value } : secret
    ));
    // Validate the secret field after update
    setTimeout(() => validateSecretField(index), 0);
  };

  const updateSecretMode = (index: number, useValueFrom: boolean) => {
    setSecrets(prev => prev.map((secret, i) =>
      i === index ? {
        ...secret,
        value: useValueFrom ? undefined : '',
        valueFrom: useValueFrom ? { secret: { name: '', key: '' } } : undefined
      } : secret
    ));
    // Validate the secret field after update
    setTimeout(() => validateSecretField(index), 0);
  };

  const removeSecret = (index: number) => {
    setSecrets(prev => prev.filter((_, i) => i !== index));
    setEditingSecretIndex(null);
  };

  // Conditions management
  const addCondition = () => {
    const newCondition: SuperplaneCondition = {
      type: 'CONDITION_TYPE_APPROVAL',
      approval: { count: 1 }
    };
    const newIndex = conditions.length;
    setConditions(prev => [...prev, newCondition]);
    setEditingConditionIndex(newIndex);
  };

  const updateCondition = (index: number, field: keyof SuperplaneCondition, value: unknown) => {
    setConditions(prev => prev.map((condition, i) =>
      i === index ? { ...condition, [field]: value } : condition
    ));
    // Validate the condition field after update
    setTimeout(() => validateConditionField(index), 0);
  };

  const updateConditionType = (index: number, type: SuperplaneConditionType) => {
    setConditions(prev => prev.map((condition, i) => {
      if (i === index) {
        const newCondition: SuperplaneCondition = { type };
        if (type === 'CONDITION_TYPE_APPROVAL') {
          newCondition.approval = { count: 1 };
          newCondition.timeWindow = undefined;
        } else if (type === 'CONDITION_TYPE_TIME_WINDOW') {
          newCondition.timeWindow = { start: '', end: '', weekDays: [] };
          newCondition.approval = undefined;
        }
        return newCondition;
      }
      return condition;
    }));
    // Validate the condition field after update
    setTimeout(() => validateConditionField(index), 0);
  };

  const removeCondition = (index: number) => {
    setConditions(prev => prev.filter((_, i) => i !== index));
    setEditingConditionIndex(null);
  };

  const cancelEditCondition = (index: number) => {
    const condition = conditions[index];
    // If this is a newly added condition with no valid type, remove it
    if (!condition.type || condition.type === 'CONDITION_TYPE_UNKNOWN') {
      removeCondition(index);
    } else {
      setEditingConditionIndex(null);
    }
  };

  // Helper functions for executor field updates
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

  // Validation functions
  const validateInput = (input: SuperplaneInputDefinition, index: number): string[] => {
    const errors: string[] = [];
    if (!input.name || input.name.trim() === '') {
      errors.push('Input name is required');
    }
    if (input.name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(input.name)) {
      errors.push('Input name must start with a letter and contain only letters, numbers, and underscores');
    }
    // Check for duplicate names
    const duplicateIndex = inputs.findIndex((inp, i) => i !== index && inp.name === input.name);
    if (duplicateIndex !== -1) {
      errors.push('Input name must be unique');
    }
    return errors;
  };

  const validateOutput = (output: SuperplaneOutputDefinition, index: number): string[] => {
    const errors: string[] = [];
    if (!output.name || output.name.trim() === '') {
      errors.push('Output name is required');
    }
    if (output.name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(output.name)) {
      errors.push('Output name must start with a letter and contain only letters, numbers, and underscores');
    }
    // Check for duplicate names
    const duplicateIndex = outputs.findIndex((out, i) => i !== index && out.name === output.name);
    if (duplicateIndex !== -1) {
      errors.push('Output name must be unique');
    }
    return errors;
  };

  const validateSecret = (secret: SuperplaneValueDefinition, index: number): string[] => {
    const errors: string[] = [];
    if (!secret.name || secret.name.trim() === '') {
      errors.push('Secret name is required');
    }
    if (secret.name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(secret.name)) {
      errors.push('Secret name must start with a letter and contain only letters, numbers, and underscores');
    }
    // Check for duplicate names
    const duplicateIndex = secrets.findIndex((sec, i) => i !== index && sec.name === secret.name);
    if (duplicateIndex !== -1) {
      errors.push('Secret name must be unique');
    }

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

  const validateConditionField = (index: number) => {
    const condition = conditions[index];
    const errors = validateCondition(condition);
    if (errors.length > 0) {
      setValidationErrors(prev => ({
        ...prev,
        [`condition_${index}`]: errors.join(', ')
      }));
    } else {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[`condition_${index}`];
        return newErrors;
      });
    }
  };

  const validateAllFields = React.useCallback((): boolean => {
    const errors: Record<string, string> = {};

    // Helper validation functions
    const validateInputItem = (input: SuperplaneInputDefinition, index: number): string[] => {
      const validationErrors: string[] = [];
      if (!input.name || input.name.trim() === '') {
        validationErrors.push('Input name is required');
      }
      if (input.name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(input.name)) {
        validationErrors.push('Input name must start with a letter and contain only letters, numbers, and underscores');
      }
      // Check for duplicate names
      const duplicateIndex = inputs.findIndex((inp, i) => i !== index && inp.name === input.name);
      if (duplicateIndex !== -1) {
        validationErrors.push('Input name must be unique');
      }
      return validationErrors;
    };

    const validateOutputItem = (output: SuperplaneOutputDefinition, index: number): string[] => {
      const validationErrors: string[] = [];
      if (!output.name || output.name.trim() === '') {
        validationErrors.push('Output name is required');
      }
      if (output.name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(output.name)) {
        validationErrors.push('Output name must start with a letter and contain only letters, numbers, and underscores');
      }
      // Check for duplicate names
      const duplicateIndex = outputs.findIndex((out, i) => i !== index && out.name === output.name);
      if (duplicateIndex !== -1) {
        validationErrors.push('Output name must be unique');
      }
      return validationErrors;
    };

    const validateConnectionItem = (connection: SuperplaneConnection): string[] => {
      const validationErrors: string[] = [];
      if (!connection.name || connection.name.trim() === '') {
        validationErrors.push('Connection name is required');
      }
      if (!connection.type) {
        validationErrors.push('Connection type is required');
      }
      return validationErrors;
    };

    const validateSecretItem = (secret: SuperplaneValueDefinition, index: number): string[] => {
      const validationErrors: string[] = [];
      if (!secret.name || secret.name.trim() === '') {
        validationErrors.push('Secret name is required');
      }
      if (secret.name && !/^[a-zA-Z][a-zA-Z0-9_]*$/.test(secret.name)) {
        validationErrors.push('Secret name must start with a letter and contain only letters, numbers, and underscores');
      }
      // Check for duplicate names
      const duplicateIndex = secrets.findIndex((sec, i) => i !== index && sec.name === secret.name);
      if (duplicateIndex !== -1) {
        validationErrors.push('Secret name must be unique');
      }

      if (!secret.valueFrom && (!secret.value || secret.value.trim() === '')) {
        validationErrors.push('Secret value is required');
      }

      if (secret.valueFrom && secret.valueFrom.secret) {
        if (!secret.valueFrom.secret.name || secret.valueFrom.secret.name.trim() === '') {
          validationErrors.push('Secret reference name is required');
        }
        if (!secret.valueFrom.secret.key || secret.valueFrom.secret.key.trim() === '') {
          validationErrors.push('Secret reference key is required');
        }
      }

      return validationErrors;
    };

    const validateConditionItem = (condition: SuperplaneCondition): string[] => {
      const validationErrors: string[] = [];
      if (!condition.type || condition.type === 'CONDITION_TYPE_UNKNOWN') {
        validationErrors.push('Condition type is required');
      }

      if (condition.type === 'CONDITION_TYPE_APPROVAL') {
        if (!condition.approval || condition.approval.count === undefined || condition.approval.count < 1) {
          validationErrors.push('Approval condition requires a count of at least 1');
        }
      } else if (condition.type === 'CONDITION_TYPE_TIME_WINDOW') {
        if (!condition.timeWindow) {
          validationErrors.push('Time window condition requires time window configuration');
        } else {
          if (!condition.timeWindow.start || condition.timeWindow.start.trim() === '') {
            validationErrors.push('Time window start time is required');
          }
          if (!condition.timeWindow.end || condition.timeWindow.end.trim() === '') {
            validationErrors.push('Time window end time is required');
          }
          if (!condition.timeWindow.weekDays || condition.timeWindow.weekDays.length === 0) {
            validationErrors.push('At least one weekday must be selected');
          }
        }
      }

      return validationErrors;
    };

    const validateExecutorItem = (executor: SuperplaneExecutor): string[] => {
      const validationErrors: string[] = [];
      if (executor.type && executor.type !== '') {
        if (!executor.spec || Object.keys(executor.spec).length === 0) {
          validationErrors.push('Executor specification is required when executor type is set');
        }

        // Type-specific validations
        if (executor.type === 'http' && executor.spec) {
          const spec = executor.spec as Record<string, unknown>;
          if (!spec.url) {
            validationErrors.push('HTTP executor requires a URL');
          } else if (typeof spec.url === 'string' && !/^https?:\/\/.+/.test(spec.url)) {
            validationErrors.push('HTTP executor URL must be a valid HTTP/HTTPS URL');
          }
        }

        if (executor.type === 'semaphore' && executor.spec) {
          const spec = executor.spec as Record<string, unknown>;
          if (!spec.pipelineFile) {
            validationErrors.push('Semaphore executor requires a pipeline file');
          }
        }
      }
      return validationErrors;
    };

    // Validate inputs
    inputs.forEach((input, index) => {
      const inputErrors = validateInputItem(input, index);
      if (inputErrors.length > 0) {
        errors[`input_${index}`] = inputErrors.join(', ');
      }
    });

    // Validate outputs
    outputs.forEach((output, index) => {
      const outputErrors = validateOutputItem(output, index);
      if (outputErrors.length > 0) {
        errors[`output_${index}`] = outputErrors.join(', ');
      }
    });

    // Validate connections
    connections.forEach((connection, index) => {
      const connectionErrors = validateConnectionItem(connection);
      if (connectionErrors.length > 0) {
        errors[`connection_${index}`] = connectionErrors.join(', ');
      }
    });

    // Validate secrets
    secrets.forEach((secret, index) => {
      const secretErrors = validateSecretItem(secret, index);
      if (secretErrors.length > 0) {
        errors[`secret_${index}`] = secretErrors.join(', ');
      }
    });

    // Validate conditions
    conditions.forEach((condition, index) => {
      const conditionErrors = validateConditionItem(condition);
      if (conditionErrors.length > 0) {
        errors[`condition_${index}`] = conditionErrors.join(', ');
      }
    });

    // Validate executor
    const executorErrors = validateExecutorItem(executor);
    if (executorErrors.length > 0) {
      errors.executor = executorErrors.join(', ');
    }


    setValidationErrors(errors);

    return Object.keys(errors).length === 0;
  }, [inputs, outputs, connections, secrets, conditions, executor]);


  // Update the onDataChange to include validation
  const handleDataChange = React.useCallback(() => {
    if (onDataChange) {
      const isValid = validateAllFields();
      onDataChange({
        label: data.label,
        description: data.description,
        inputs,
        outputs,
        connections,
        executor,
        secrets,
        conditions,
        isValid
      });
    }
  }, [data.label, data.description, inputs, outputs, connections, executor, secrets, conditions, onDataChange, validateAllFields]);

  // Notify parent of data changes with validation
  useEffect(() => {
    handleDataChange();
  }, [handleDataChange]);

  const cancelEditSecret = (index: number) => {
    const secret = secrets[index];
    // If this is a newly added secret with no name, remove it
    if (!secret.name || secret.name.trim() === '') {
      removeSecret(index);
    } else {
      setEditingSecretIndex(null);
    }
  };


  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      {/* Accordion Sections */}
      <div className="">

        {/* Connections Section */}
        <AccordionItem
          id="connections"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Connections</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {connections.length} connections
              </span>
            </div>
          }
          isOpen={openSections.includes('connections')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
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
                      {validationErrors[`connection_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`connection_${index}`]}
                        </div>
                      )}
                    </Field>

                    {/* Filters Section */}
                    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                      <div className="flex justify-between items-center mb-2">
                        <Label className="text-sm font-medium">Filters</Label>
                        <button
                          onClick={() => addFilter(index)}
                          className="text-blue-600 hover:text-blue-700 text-sm"
                        >
                          + Add Filter
                        </button>
                      </div>
                      <div className="space-y-2">
                        {(connection.filters || []).map((filter, filterIndex) => (
                          <div key={filterIndex}>
                            <div className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                              <select
                                value={filter.type || 'FILTER_TYPE_DATA'}
                                onChange={(e) => {
                                  const type = e.target.value as SuperplaneFilterType;
                                  const updates: Partial<SuperplaneFilter> = { type };
                                  if (type === 'FILTER_TYPE_DATA') {
                                    updates.data = { expression: filter.data?.expression || '' };
                                    updates.header = undefined;
                                  } else {
                                    updates.header = { expression: filter.header?.expression || '' };
                                    updates.data = undefined;
                                  }
                                  updateFilter(index, filterIndex, updates);
                                }}
                                className="px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                              >
                                <option value="FILTER_TYPE_DATA">Data</option>
                                <option value="FILTER_TYPE_HEADER">Header</option>
                              </select>
                              <input
                                type="text"
                                value={
                                  filter.type === 'FILTER_TYPE_HEADER'
                                    ? filter.header?.expression || ''
                                    : filter.data?.expression || ''
                                }
                                onChange={(e) => {
                                  const expression = e.target.value;
                                  const updates: Partial<SuperplaneFilter> = {};
                                  if (filter.type === 'FILTER_TYPE_HEADER') {
                                    updates.header = { expression };
                                  } else {
                                    updates.data = { expression };
                                  }
                                  updateFilter(index, filterIndex, updates);
                                }}
                                placeholder="Filter expression"
                                className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                              />
                              <button
                                onClick={() => removeFilter(index, filterIndex)}
                                className="text-red-600 hover:text-red-700"
                              >
                                <span className="material-symbols-outlined text-sm">delete</span>
                              </button>
                            </div>
                            {/* OR/AND toggle between filters */}
                            {filterIndex < (connection.filters?.length || 0) - 1 && (
                              <div className="flex justify-center py-1">
                                <button
                                  onClick={() => toggleFilterOperator(index)}
                                  className="px-3 py-1 text-xs bg-zinc-200 dark:bg-zinc-700 rounded-full hover:bg-zinc-300 dark:hover:bg-zinc-600"
                                >
                                  {connection.filterOperator === 'FILTER_OPERATOR_OR' ? 'OR' : 'AND'}
                                </button>
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>

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

        {/* Inputs Section */}
        <AccordionItem
          id="inputs"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Inputs</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {inputs.length} inputs
              </span>
            </div>
          }
          isOpen={openSections.includes('inputs')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {inputs.map((input, index) => (
              <div key={index}>
                {editingInputIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Name</Label>
                      <input
                        type="text"
                        value={input.name || ''}
                        onChange={(e) => updateInput(index, 'name', e.target.value)}
                        placeholder="Input name"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`input_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                      {validationErrors[`input_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`input_${index}`]}
                        </div>
                      )}
                    </Field>
                    <Field>
                      <Label>Description</Label>
                      <textarea
                        value={input.description || ''}
                        onChange={(e) => updateInput(index, 'description', e.target.value)}
                        placeholder="Input description"
                        rows={2}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`input_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </Field>

                    {/* Mappings Section */}
                    <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                      <div className="flex justify-between items-center mb-2">
                        <Label className="text-sm font-medium">Mappings</Label>
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
                              className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            >
                              <option value="">Select connection</option>
                              {data.connections.map((conn, connIndex) => (
                                <option key={connIndex} value={conn.name}>{conn.name}</option>
                              ))}
                            </select>
                            <select
                              value={mapping.value || ''}
                              onChange={(e) => updateMapping(index, mappingIndex, 'value', e.target.value)}
                              className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            >
                              <option value="">Select output value</option>
                              {getOutputsFromConnection(mapping.name || '').map((output, outputIndex) => (
                                <option key={outputIndex} value={output.name || ''}>{output.name || 'Unnamed Output'}</option>
                              ))}
                            </select>
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

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => cancelEditInput(index)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingInputIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <span className="text-sm font-medium">{input.name || `Input ${index + 1}`}</span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingInputIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeInput(index)}
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
              onClick={addInput}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Input
            </button>
          </div>
        </AccordionItem>

        {/* Outputs Section */}
        <AccordionItem
          id="outputs"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Outputs</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {outputs.length} outputs
              </span>
            </div>
          }
          isOpen={openSections.includes('outputs')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {outputs.map((output, index) => (
              <div key={index}>
                {editingOutputIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Name</Label>
                      <input
                        type="text"
                        value={output.name || ''}
                        onChange={(e) => updateOutput(index, 'name', e.target.value)}
                        placeholder="Output name"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`output_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                      {validationErrors[`output_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`output_${index}`]}
                        </div>
                      )}
                    </Field>
                    <Field>
                      <Label>Description</Label>
                      <textarea
                        value={output.description || ''}
                        onChange={(e) => updateOutput(index, 'description', e.target.value)}
                        placeholder="Output description"
                        rows={2}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`output_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </Field>
                    <Field>
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={`required-${index}`}
                          checked={output.required || false}
                          onChange={(e) => updateOutput(index, 'required', e.target.checked)}
                          className="w-4 h-4 text-blue-600 bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600 rounded focus:ring-blue-500"
                        />
                        <Label htmlFor={`required-${index}`}>Required</Label>
                      </div>
                    </Field>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => cancelEditOutput(index)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingOutputIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{output.name || `Output ${index + 1}`}</span>
                      {output.required && (
                        <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">Required</span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingOutputIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeOutput(index)}
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
              onClick={addOutput}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Output
            </button>
          </div>
        </AccordionItem>

        {/* Conditions Section */}
        <AccordionItem
          id="conditions"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Conditions</span>
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-500">{conditions.length} condition{conditions.length !== 1 ? 's' : ''}</span>
              </div>
            </div>
          }
          isOpen={openSections.includes('conditions')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-3">
            {conditions.map((condition, index) => (
              <div key={index} className="p-3 border border-zinc-200 dark:border-zinc-700 rounded-md bg-zinc-50 dark:bg-zinc-800">
                {editingConditionIndex === index ? (
                  <div className="space-y-3">
                    <div>
                      <Label className="text-xs mb-1">Condition Type</Label>
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
                      {validationErrors[`condition_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`condition_${index}`]}
                        </div>
                      )}
                    </div>

                    {condition.type === 'CONDITION_TYPE_APPROVAL' && (
                      <div>
                        <Label className="text-xs mb-1">Required Approvals</Label>
                        <input
                          type="number"
                          min="1"
                          value={condition.approval?.count || 1}
                          onChange={(e) => updateCondition(index, 'approval', { count: parseInt(e.target.value) || 1 })}
                          placeholder="Number of required approvals"
                          className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`condition_${index}`]
                            ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                            : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                            }`}
                        />
                      </div>
                    )}

                    {condition.type === 'CONDITION_TYPE_TIME_WINDOW' && (
                      <div className="space-y-3">
                        <div className="grid grid-cols-2 gap-3">
                          <div>
                            <Label className="text-xs mb-1">Start Time</Label>
                            <input
                              type="time"
                              value={condition.timeWindow?.start || ''}
                              onChange={(e) => updateCondition(index, 'timeWindow', {
                                ...condition.timeWindow,
                                start: e.target.value
                              })}
                              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`condition_${index}`]
                                ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                                : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                                }`}
                            />
                          </div>
                          <div>
                            <Label className="text-xs mb-1">End Time</Label>
                            <input
                              type="time"
                              value={condition.timeWindow?.end || ''}
                              onChange={(e) => updateCondition(index, 'timeWindow', {
                                ...condition.timeWindow,
                                end: e.target.value
                              })}
                              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`condition_${index}`]
                                ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                                : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                                }`}
                            />
                          </div>
                        </div>
                        <div>
                          <Label className="text-xs mb-1">Days of Week</Label>
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
                                    updateCondition(index, 'timeWindow', {
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
                        </div>
                      </div>
                    )}

                    <div className="flex items-center justify-between pt-2">
                      <button
                        onClick={() => cancelEditCondition(index)}
                        className="text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                      >
                        <span className="material-symbols-outlined text-sm">check</span>
                      </button>
                      <button
                        onClick={() => removeCondition(index)}
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
                        onClick={() => setEditingConditionIndex(index)}
                        className="text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeCondition(index)}
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
              onClick={addCondition}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Condition
            </button>
          </div>
        </AccordionItem>

        {/* Secrets Section */}
        <AccordionItem
          id="secrets"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Secrets Management</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {secrets.length} secrets
              </span>
            </div>
          }
          isOpen={openSections.includes('secrets')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-2">
            {secrets.map((secret, index) => (
              <div key={index}>
                {editingSecretIndex === index ? (
                  <div className="border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3">
                    <Field>
                      <Label>Secret Name</Label>
                      <input
                        type="text"
                        value={secret.name || ''}
                        onChange={(e) => updateSecret(index, 'name', e.target.value)}
                        placeholder="Secret name"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                      {validationErrors[`secret_${index}`] && (
                        <div className="text-xs text-red-600 mt-1">
                          {validationErrors[`secret_${index}`]}
                        </div>
                      )}
                    </Field>

                    <Field>
                      <div className="flex items-center gap-4 mb-2">
                        <Label>Value Source</Label>
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
                          onChange={(e) => updateSecret(index, 'value', e.target.value)}
                          placeholder="Secret value"
                          className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                            ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                            : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                            }`}
                        />
                      ) : (
                        <div className="space-y-2">
                          <div>
                            <Label className="text-xs mb-1">Secret Name</Label>
                            <select
                              value={secret.valueFrom?.secret?.name || ''}
                              onChange={(e) => {
                                const selectedSecretName = e.target.value;
                                updateSecret(index, 'valueFrom', {
                                  ...secret.valueFrom,
                                  secret: {
                                    ...secret.valueFrom?.secret,
                                    name: selectedSecretName,
                                    key: '' // Reset key when secret changes
                                  }
                                });
                              }}
                              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                                ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                                : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                                }`}
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
                          </div>
                          <div>
                            <Label className="text-xs mb-1">Secret Key</Label>
                            <select
                              value={secret.valueFrom?.secret?.key || ''}
                              onChange={(e) => updateSecret(index, 'valueFrom', {
                                ...secret.valueFrom,
                                secret: { ...secret.valueFrom?.secret, key: e.target.value }
                              })}
                              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                                ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                                : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                                }`}
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
                          </div>
                        </div>
                      )}
                    </Field>

                    <div className="flex justify-end gap-2 pt-2">
                      <Button outline onClick={() => cancelEditSecret(index)}>
                        Cancel
                      </Button>
                      <Button color="blue" onClick={() => setEditingSecretIndex(null)}>
                        <MaterialSymbol name="save" size="sm" data-slot="icon" />
                        Save
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded">
                    <span className="text-sm font-medium">{secret.name || `Secret ${index + 1}`}</span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingSecretIndex(index)}
                        className="text-zinc-500 hover:text-zinc-700"
                      >
                        <span className="material-symbols-outlined text-sm">edit</span>
                      </button>
                      <button
                        onClick={() => removeSecret(index)}
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
              onClick={addSecret}
              className="flex items-center gap-2 text-sm text-zinc-600 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <span className="material-symbols-outlined text-sm">add</span>
              Add Secret
            </button>
          </div>
        </AccordionItem>

        {/* Executor Management Section */}
        <AccordionItem
          id="executor"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Executor Configuration</span>
              <span className="text-xs text-zinc-600 dark:text-zinc-400 font-normal pr-2">
                {executor.type || 'Not configured'}
              </span>
            </div>
          }
          isOpen={openSections.includes('executor')}
          onToggle={handleAccordionToggle}
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
                  <Label>Resource Type</Label>
                  <input
                    type="text"
                    value={(executor.resource?.type as string) || ''}
                    onChange={(e) => updateExecutorResource('type', e.target.value)}
                    placeholder="project"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>

                <Field>
                  <Label>Resource Name</Label>
                  <input
                    type="text"
                    value={(executor.resource?.name as string) || ''}
                    onChange={(e) => updateExecutorResource('name', e.target.value)}
                    placeholder="my-semaphore-project"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />
                </Field>

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
                      <div key={key} className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                        <input
                          type="text"
                          value={key}
                          onChange={(e) => updateExecutorParameter(key, e.target.value, value)}
                          placeholder="Parameter name"
                          className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                        />
                        <input
                          type="text"
                          value={value}
                          onChange={(e) => updateExecutorParameter(key, key, e.target.value)}
                          placeholder="Parameter value"
                          className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
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
                          className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
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
                          className="flex-1 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
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
                    value={((executor.spec?.responsePolicy as Record<string, unknown>)?.statusCodes as number[] || []).join(', ')}
                    onChange={(e) => {
                      const codes = e.target.value.split(',').map(code => parseInt(code.trim())).filter(code => !isNaN(code));
                      updateExecutorNestedField('responsePolicy', 'statusCodes', codes);
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
        </AccordionItem>
      </div>
    </div>
  );
}