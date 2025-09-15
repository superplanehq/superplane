import { useState, useEffect, useCallback } from 'react';
import { StageNodeType } from '@/canvas/types/flow';
import { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneValueDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneCondition, SuperplaneConditionType, SuperplaneInputMapping } from '@/api-client/types.gen';
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
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { ControlledTabs } from '@/components/Tabs/tabs';
import IntegrationZeroState from '@/components/IntegrationZeroState';
import { createInputMappingHandlers } from '../utils/inputMappingHandlers';
import { twMerge } from 'tailwind-merge';
import { showErrorToast } from '@/utils/toast';
import { Button } from '@/components/Button/button';

interface StageEditModeContentProps {
  data: StageNodeType['data'];
  currentStageId?: string;
  canvasId: string;
  organizationId: string;
  onDataChange?: (data: {
    name: string;
    description?: string;
    inputs: SuperplaneInputDefinition[];
    outputs: SuperplaneOutputDefinition[];
    connections: SuperplaneConnection[];
    executor: SuperplaneExecutor;
    secrets: SuperplaneValueDefinition[];
    conditions: SuperplaneCondition[];
    inputMappings: SuperplaneInputMapping[];
    isValid: boolean
  }) => void;
  onTriggerSectionValidation?: { current: ((hasFieldErrors?: boolean) => void) | null };
}

interface ParameterWithId {
  id: string;
  key: string;
  value: string;
}


export function StageEditModeContent({ data, currentStageId, canvasId, organizationId, onDataChange, onTriggerSectionValidation }: StageEditModeContentProps) {
  // Component-specific state
  const [inputs, setInputs] = useState<SuperplaneInputDefinition[]>(data.inputs || []);
  const [outputs, setOutputs] = useState<SuperplaneOutputDefinition[]>(data.outputs || []);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [secrets, setSecrets] = useState<SuperplaneValueDefinition[]>(data.secrets || []);
  const [conditions, setConditions] = useState<SuperplaneCondition[]>(data.conditions || []);
  const [executor, setExecutor] = useState<SuperplaneExecutor>({ type: data.executor?.type || '', spec: data.executor?.spec || {} });
  const [inputMappings, setInputMappings] = useState<SuperplaneInputMapping[]>(data.inputMappings || []);
  const [responsePolicyStatusCodesDisplay, setResponsePolicyStatusCodesDisplay] = useState(
    ((executor.spec?.responsePolicy as Record<string, unknown>)?.statusCodes as number[] || []).join(', ')
  );
  const [payloadDisplay, setPayloadDisplay] = useState(
    JSON.stringify(executor.spec?.payload || {}, null, 2)
  );
  const [semaphoreExecutionType, setSemaphoreExecutionType] = useState<'workflow' | 'task'>(
    (executor.spec?.task as string) ? 'task' : 'workflow'
  );

  const [semaphoreParameters, setSemaphoreParameters] = useState<ParameterWithId[]>(() => {
    const params = executor.spec?.parameters as Record<string, string>;
    if (!params) return [];
    return Object.entries(params).map(([key, value], index) => ({
      id: `param_${Date.now()}_${index}`,
      key,
      value
    }));
  });
  const [githubInputs, setGithubInputs] = useState<ParameterWithId[]>(() => {
    const inputs = executor.spec?.inputs as Record<string, string>;
    if (!inputs) return [];
    return Object.entries(inputs).map(([key, value], index) => ({
      id: `input_${Date.now()}_${index}`,
      key,
      value
    }));
  });
  const [httpHeaders, setHttpHeaders] = useState<ParameterWithId[]>(() => {
    const headers = executor.spec?.headers as Record<string, string>;
    if (!headers) return [];
    return Object.entries(headers).map(([key, value], index) => ({
      id: `header_${Date.now()}_${index}`,
      key,
      value
    }));
  });
  const [nextIdCounter, setNextIdCounter] = useState(1);

  const generateId = useCallback(() => {
    const id = `param_${nextIdCounter}`;
    setNextIdCounter(prev => prev + 1);
    return id;
  }, [nextIdCounter]);


  const parametersWithIdToRecord = useCallback((params: ParameterWithId[]): Record<string, string> => {
    return params.reduce((acc, param) => {
      if (param.key.trim() !== '') {
        acc[param.key] = param.value;
      }
      return acc;
    }, {} as Record<string, string>);
  }, []);

  // Validation
  const { validateName } = useValidation();

  // Input mapping handlers
  const inputMappingHandlers = createInputMappingHandlers({
    inputMappings,
    setInputMappings,
    inputs
  });

  // Fetch secrets and integrations
  const { data: canvasSecrets = [], isLoading: loadingCanvasSecrets } = useSecrets(canvasId!, "DOMAIN_TYPE_CANVAS");
  const { data: organizationSecrets = [], isLoading: loadingOrganizationSecrets } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");
  const { data: canvasIntegrations = [] } = useIntegrations(canvasId!, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  // API Error parsing function
  const parseApiErrorMessage = useCallback((errorMessage: string): { field: string; message: string } | null => {
    if (!errorMessage) return null;

    const repositoryNotFoundMatch = errorMessage.match(/repository\s+([^\s]+)\s+not\s+found/i);
    if (repositoryNotFoundMatch) {
      return {
        field: 'repository',
        message: `Repository "${repositoryNotFoundMatch[1]}" not found`
      };
    }

    const workflowNotFoundMatch = errorMessage.match(/workflow\s+([^\s]+)\s+not\s+found/i);
    if (workflowNotFoundMatch) {
      return {
        field: 'workflow',
        message: `Workflow "${workflowNotFoundMatch[1]}" not found`
      };
    }

    // Check for project not found error
    const projectNotFoundMatch = errorMessage.match(/project\s+([^\s]+)\s+not\s+found/i);
    if (projectNotFoundMatch) {
      return {
        field: 'project',
        message: `Project "${projectNotFoundMatch[1]}" not found`
      };
    }

    return null;
  }, []);

  // Handle API errors by highlighting fields
  const handleApiError = useCallback((errorMessage: string) => {
    const parsedError = parseApiErrorMessage(errorMessage);
    if (parsedError) {
      setFieldErrors(prev => ({
        ...prev,
        [parsedError.field]: parsedError.message
      }));
      onTriggerSectionValidation?.current?.(true);
    }
    showErrorToast(errorMessage);
  }, [parseApiErrorMessage, onTriggerSectionValidation]);

  // Expose handleApiError to global scope for stage.tsx to call
  useEffect(() => {
    (window as { handleStageApiError?: (errorMessage: string) => void }).handleStageApiError = handleApiError;

    return () => {
      delete (window as { handleStageApiError?: (errorMessage: string) => void }).handleStageApiError;
    };
  }, [handleApiError]);

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

  // Helper function to ensure all inputs are included in input mappings
  const ensureAllInputsInMappings = useCallback((mappings: SuperplaneInputMapping[], allInputs: SuperplaneInputDefinition[]) => {
    return mappings.map(mapping => {
      const existingValues = mapping.values || [];
      const missingInputs = allInputs.filter(inp =>
        inp.name && !existingValues.some(v => v.name === inp.name)
      );

      const additionalValues = missingInputs.map(inp => ({
        name: inp.name,
        value: ''
      }));

      return {
        ...mapping,
        values: [...existingValues, ...additionalValues]
      };
    });
  }, []);


  const getSecretKeys = (secretName: string) => {
    const allSecrets = getAllSecrets();
    const selectedSecret = allSecrets.find(secret => secret.name === secretName);
    return selectedSecret ? Object.keys(selectedSecret.data) : [];
  };

  // Validation functions
  const validateInput = (input: SuperplaneInputDefinition, index: number): string[] => {
    const errors: string[] = [];

    const nameErrors = validateName(input.name, inputs, index);
    errors.push(...nameErrors);

    return errors;
  };

  const validateOutput = (output: SuperplaneOutputDefinition, index: number): string[] => {
    const errors: string[] = [];

    const nameErrors = validateName(output.name, outputs, index);
    errors.push(...nameErrors);

    return errors;
  };

  const validateSecret = (secret: SuperplaneValueDefinition, index: number): string[] => {
    const errors: string[] = [];
    const nameErrors = validateName(secret.name, secrets, index);
    errors.push(...nameErrors);

    if (!secret.valueFrom || !secret.valueFrom.secret) {
      errors.push('Secret must reference an existing secret');
    } else {
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

  const validateExecutor = useCallback((executor: SuperplaneExecutor): Record<string, string> => {
    const errors: Record<string, string> = {};

    if (!executor.type || executor.type === '') {
      errors.executorType = 'Executor type is required';
      return errors;
    }

    if (executor.type === 'semaphore') {
      if (!executor.integration?.name) {
        errors.executorIntegration = 'Semaphore integration is required';
      }
      if (!executor.resource?.name) {
        errors.executorProject = 'Project name is required';
      }
      if (!executor.spec?.ref) {
        errors.executorRef = 'Ref (branch/tag) is required';
      }
      if (!executor.spec?.pipelineFile) {
        errors.executorPipelineFile = 'Pipeline file is required';
      }
    } else if (executor.type === 'github') {
      if (!executor.integration?.name) {
        errors.executorIntegration = 'GitHub integration is required';
      }
      if (!executor.resource?.name) {
        errors.executorRepository = 'Repository name is required';
      }
      if (!executor.spec?.workflow) {
        errors.executorWorkflow = 'Workflow file is required';
      }
      if (!executor.spec?.ref) {
        errors.executorRef = 'Ref (branch/tag) is required';
      }
    } else if (executor.type === 'http') {
      const urlRegex = /^https?:\/\/(www\.)?([-a-zA-Z0-9@:%._+~#=]{1,256}(\.[a-zA-Z0-9()]{1,6})?|(\d{1,3}\.){3}\d{1,3})(:\d+)?([-a-zA-Z0-9()@:%_+.~#?&//=]*)?/;
      if (!executor.spec?.url || !urlRegex.test(executor.spec.url as string)) {
        errors.executorUrl = 'Valid URL is required';
      }
    }

    return errors;
  }, []);

  const validateAllFields = () => {
    let errors: Record<string, string> = {};

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


    if (connections.length === 0) {
      errors.connections = 'At least one connection is required';
    }


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

    // Validate input mappings and track input-specific errors
    const inputMappingErrors = new Map<string, string[]>();

    inputMappings.forEach((mapping, index) => {
      const mappingErrors: string[] = [];
      if (!mapping.when?.triggeredBy?.connection) {
        mappingErrors.push('Trigger connection is required');
      }

      // Check if all inputs are included in the mapping
      if (inputs.length > 0) {
        const mappingInputNames = mapping.values?.map(v => v.name) || [];
        const missingInputs = inputs
          .map(input => input.name)
          .filter(inputName => inputName && !mappingInputNames.includes(inputName));

        // Add missing input errors to individual inputs
        missingInputs.forEach(inputName => {
          if (inputName) {
            const inputIndex = inputs.findIndex(inp => inp.name === inputName);
            if (inputIndex !== -1) {
              const currentErrors = inputMappingErrors.get(`input_${inputIndex}`) || [];
              currentErrors.push(`Missing in mapping for connection "${mapping.when?.triggeredBy?.connection || 'Unknown'}"`);
              inputMappingErrors.set(`input_${inputIndex}`, currentErrors);
            }
          }
        });
      }

      // Check if all mapping values have either value or valueFrom defined
      if (mapping.values) {
        mapping.values.forEach((value) => {
          const hasStaticValue = value.value !== undefined && value.value !== '';
          const hasEventDataValue = value.valueFrom?.eventData?.expression !== undefined && value.valueFrom.eventData.expression !== '';
          const hasLastExecutionValue = value.valueFrom?.lastExecution?.results !== undefined && value.valueFrom.lastExecution.results.length > 0;

          if (!hasStaticValue && !hasEventDataValue && !hasLastExecutionValue) {
            const inputIndex = inputs.findIndex(inp => inp.name === value.name);
            if (inputIndex !== -1) {
              const currentErrors = inputMappingErrors.get(`input_${inputIndex}`) || [];
              currentErrors.push(`No value defined in mapping for connection "${mapping.when?.triggeredBy?.connection || 'Unknown'}"`);
              inputMappingErrors.set(`input_${inputIndex}`, currentErrors);
            }
          }
        });
      }

      // Only add mapping-level errors for connection issues
      if (mappingErrors.length > 0) {
        errors[`inputMapping_${index}`] = mappingErrors.join(', ');
      }
    });

    // Add input-specific mapping errors to the validation errors
    inputMappingErrors.forEach((errorList, inputKey) => {
      const existingErrors = errors[inputKey];
      const newErrors = errorList.join(', ');
      errors[inputKey] = existingErrors ? `${existingErrors}, ${newErrors}` : newErrors;
    });

    // Comprehensive executor validation
    const executorErrors = validateExecutor(executor);
    errors = {
      ...errors,
      ...executorErrors
    };

    setValidationErrors(errors);

    return Object.keys(errors).length === 0;
  };

  const isExecutorMisconfigured = useCallback(() => {
    const errors = validateExecutor(executor);
    return Object.keys(errors).length > 0;
  }, [executor, validateExecutor]);

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
      name: data.name,
      description: data.description,
      inputs: data.inputs || [],
      outputs: data.outputs || [],
      connections: data.connections || [],
      secrets: data.secrets || [],
      conditions: data.conditions || [],
      executor: data.executor || { type: '', spec: {} },
      inputMappings: data.inputMappings || [],
      isValid: true
    },
    onDataChange,
    validateAllFields: () => {
      const executorErrors = validateExecutor(executor);

      if (Object.keys(executorErrors).length > 0) {
        setOpenSections(prev => [...prev, 'executor']);
      }
      return validateAllFields();
    },
  });

  const triggerSectionValidation = useCallback((hasFieldErrors: boolean = false) => {
    const sectionsToOpen = [...openSections];

    const connectionsNeedConfiguration = connections.length === 0;
    if (connectionsNeedConfiguration && !sectionsToOpen.includes('connections')) {
      sectionsToOpen.push('connections');
    }

    const executorErrors = validateExecutor(executor);
    if (Object.keys(executorErrors).length > 0 || hasFieldErrors) {
      if (!sectionsToOpen.includes('executor')) {
        sectionsToOpen.push('executor');
      }
      setValidationErrors(prev => ({
        ...prev,
        ...executorErrors
      }));
    }

    setOpenSections(sectionsToOpen);
  }, [connections.length, executor, openSections, setOpenSections, setValidationErrors, validateExecutor]);

  // Expose the trigger function to parent
  useEffect(() => {
    if (onTriggerSectionValidation) {
      onTriggerSectionValidation.current = triggerSectionValidation;
    }
  }, [triggerSectionValidation, onTriggerSectionValidation]);

  // Initialize open sections
  useEffect(() => {
    const sectionsToOpen = ['general'];

    const connectionsNeedConfiguration = connections.length === 0;
    if (connectionsNeedConfiguration) {
      sectionsToOpen.push('connections');
    }

    const executorErrors = validateExecutor(executor);

    if (Object.keys(executorErrors).length > 0) {
      sectionsToOpen.push('executor');
      setValidationErrors(executorErrors);
    }

    setOpenSections(sectionsToOpen);
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
    createNewItem: () => ({ name: '', valueFrom: { secret: { name: '', key: '' } } }),
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



  useEffect(() => {
    if (executor.type === 'semaphore' && semaphoreParameters.length > 0) {
      const newParams = parametersWithIdToRecord(semaphoreParameters);
      const currentParams = executor.spec?.parameters as Record<string, string>;

      if (JSON.stringify(newParams) !== JSON.stringify(currentParams || {})) {
        updateExecutorField('parameters', newParams);
      }
    }
  }, [semaphoreParameters, executor.type, executor.spec?.parameters, parametersWithIdToRecord]);

  useEffect(() => {
    if (executor.type === 'github' && githubInputs.length > 0) {
      const newInputs = parametersWithIdToRecord(githubInputs);
      const currentInputs = executor.spec?.inputs as Record<string, string>;

      if (JSON.stringify(newInputs) !== JSON.stringify(currentInputs || {})) {
        updateExecutorField('inputs', newInputs);
      }
    }
  }, [githubInputs, executor.type, executor.spec?.inputs, parametersWithIdToRecord]);

  useEffect(() => {
    if (executor.type === 'http' && httpHeaders.length > 0) {
      const newHeaders = parametersWithIdToRecord(httpHeaders);
      const currentHeaders = executor.spec?.headers as Record<string, string>;

      if (JSON.stringify(newHeaders) !== JSON.stringify(currentHeaders || {})) {
        updateExecutorField('headers', newHeaders);
      }
    }
  }, [httpHeaders, executor.type, executor.spec?.headers, parametersWithIdToRecord]);

  // Sync component state with incoming data prop changes
  useEffect(() => {
    syncWithIncomingData(
      {
        name: data.name,
        description: data.description,
        inputs: data.inputs || [],
        outputs: data.outputs || [],
        connections: data.connections || [],
        secrets: data.secrets || [],
        conditions: data.conditions || [],
        executor: data.executor || { type: '', spec: {} },
        inputMappings: data.inputMappings || [],
        isValid: true
      },
      (incomingData) => {
        setInputs(incomingData.inputs);
        setOutputs(incomingData.outputs);
        setConnections(incomingData.connections);
        setSecrets(incomingData.secrets);
        setConditions(incomingData.conditions);
        setExecutor(incomingData.executor);
        setInputMappings(incomingData.inputMappings);
        setResponsePolicyStatusCodesDisplay(
          ((incomingData.executor?.spec?.responsePolicy as Record<string, unknown>)?.statusCodes as number[] || []).join(', ')
        );
        setPayloadDisplay(
          JSON.stringify(incomingData.executor?.spec?.payload || {}, null, 2)
        );
      }
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  // Auto-update input mappings when inputs change
  useEffect(() => {
    if (inputMappings.length > 0) {
      const updatedMappings = ensureAllInputsInMappings(inputMappings, inputs);
      // Only update if there are actual changes to prevent infinite loops
      const hasChanges = updatedMappings.some((mapping, index) =>
        mapping.values?.length !== inputMappings[index]?.values?.length
      );
      if (hasChanges) {
        setInputMappings(updatedMappings);
      }
    }
  }, [inputs, inputMappings, ensureAllInputsInMappings]);

  // Notify parent of data changes
  useEffect(() => {
    if (onDataChange) {
      handleDataChange({
        name: data.name,
        description: data.description,
        inputs,
        outputs,
        connections,
        executor,
        secrets,
        conditions,
        inputMappings
      });
    }// eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data.name, data.description, inputs, outputs, connections, executor, secrets, conditions, inputMappings, onDataChange]);

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
        setPayloadDisplay((originalData.executor?.spec as Record<string, unknown>)?.payload as string || '{}');
        break;
    }
  };

  // Helper functions for executor

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
    const newParam: ParameterWithId = {
      id: generateId(),
      key: `PARAM_${semaphoreParameters.length + 1}`,
      value: ''
    };
    setSemaphoreParameters(prev => [...prev, newParam]);
  };

  const updateExecutorParameter = (id: string, key: string, value: string) => {
    setSemaphoreParameters(prev =>
      prev.map(param =>
        param.id === id ? { ...param, key, value } : param
      )
    );
  };

  const removeExecutorParameter = (id: string) => {
    setSemaphoreParameters(prev => prev.filter(param => param.id !== id));
  };

  const addExecutorInput = () => {
    const newInput: ParameterWithId = {
      id: generateId(),
      key: `INPUT_${githubInputs.length + 1}`,
      value: ''
    };
    setGithubInputs(prev => [...prev, newInput]);
  };

  const updateExecutorInput = (id: string, key: string, value: string) => {
    setGithubInputs(prev =>
      prev.map(input =>
        input.id === id ? { ...input, key, value } : input
      )
    );
  };

  const removeExecutorInput = (id: string) => {
    setGithubInputs(prev => prev.filter(input => input.id !== id));
  };

  const addExecutorHeader = () => {
    const newHeader: ParameterWithId = {
      id: generateId(),
      key: `Header_${httpHeaders.length + 1}`,
      value: ''
    };
    setHttpHeaders(prev => [...prev, newHeader]);
  };

  const updateExecutorHeader = (id: string, key: string, value: string) => {
    setHttpHeaders(prev =>
      prev.map(header =>
        header.id === id ? { ...header, key, value } : header
      )
    );
  };

  const removeExecutorHeader = (id: string) => {
    setHttpHeaders(prev => prev.filter(header => header.id !== id));
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

  const availableIntegrations = getAllIntegrations();
  const semaphoreIntegrations = availableIntegrations.filter(int => int.spec?.type === 'semaphore');
  const githubIntegrations = availableIntegrations.filter(int => int.spec?.type === 'github');
  const requireIntegration = ['semaphore', 'github'].includes(executor?.type || '');
  const hasRequiredIntegrations = (executor.type === 'semaphore' && semaphoreIntegrations.length > 0) ||
    (executor.type === 'github' && githubIntegrations.length > 0) ||
    (!executor.type || (executor.type !== 'semaphore' && executor.type !== 'github'));

  const getZeroStateLabel = () => {
    switch (executor.type) {
      case 'semaphore':
        return 'Semaphore organizations';
      case 'github':
        return 'GitHub accounts';
      default:
        return 'integrations';
    }
  };

  const handleTriggerConnectionChange = (selectedConnection: string, actualMappingIndex: number, input: SuperplaneInputDefinition, mapping: SuperplaneInputMapping) => {
    const newMappings = [...inputMappings];
    const currentInputValue = mapping.values?.find(v => v.name === input.name);

    if (!currentInputValue) return;

    const existingMappingIndex = newMappings.findIndex((m, idx) =>
      idx !== actualMappingIndex &&
      m.when?.triggeredBy?.connection === selectedConnection &&
      selectedConnection !== ''
    );
    const mappingExists = existingMappingIndex !== -1 && selectedConnection !== '';

    if (mappingExists) {
      const existingValues = newMappings[existingMappingIndex].values || [];
      const valueExists = existingValues.some(v => v.name === input.name);

      if (!valueExists) {
        newMappings[existingMappingIndex].values = [...existingValues, currentInputValue];
      } else {
        newMappings[existingMappingIndex].values = existingValues.map(v =>
          v.name === input.name ? currentInputValue : v
        );
      }

      const updatedOriginalValues = (newMappings[actualMappingIndex].values || []).filter(v => v.name !== input.name);

      if (updatedOriginalValues.length === 0) {
        newMappings.splice(actualMappingIndex, 1);
      } else {
        newMappings[actualMappingIndex].values = updatedOriginalValues;
      }
      setInputMappings(newMappings);
      return;
    }


    if (selectedConnection === '') {
      newMappings[actualMappingIndex] = {
        ...newMappings[actualMappingIndex],
        when: {
          ...newMappings[actualMappingIndex].when,
          triggeredBy: { connection: selectedConnection }
        }
      };
    } else {
      newMappings[actualMappingIndex].when = {
        triggeredBy: { connection: selectedConnection }
      };

      newMappings[actualMappingIndex].values = inputs.map(input => ({
        name: input.name,
        value: ''
      }));
    }
    setInputMappings(newMappings);

  };

  const handleInputNameChange = (newName: string, index: number, input: SuperplaneInputDefinition) => {
    const oldName = input.name;
    inputsEditor.updateItem(index, 'name', newName);

    if (oldName && newName !== oldName) {
      const updatedMappings = inputMappings.map(mapping => ({
        ...mapping,
        values: mapping.values?.map(value =>
          value.name === oldName
            ? { ...value, name: newName }
            : value
        ) || []
      }));
      setInputMappings(updatedMappings);
    }
  };

  const handleValueModeChange = (mode: 'static' | 'eventData' | 'lastExecution', actualMappingIndex: number, input: SuperplaneInputDefinition) => {
    const newMappings = [...inputMappings];
    const values = [...(newMappings[actualMappingIndex].values || [])];
    const valueIndex = values.findIndex(v => v.name === input.name);

    if (valueIndex === -1) {
      return;
    }

    switch (mode) {
      case 'static':
        values[valueIndex] = {
          ...values[valueIndex],
          value: values[valueIndex]?.valueFrom?.eventData?.expression ||
            values[valueIndex]?.value || '',
          valueFrom: undefined
        };
        break;
      case 'eventData':
        values[valueIndex] = {
          ...values[valueIndex],
          value: undefined,
          valueFrom: {
            eventData: {
              connection: '',
              expression: values[valueIndex]?.value || ''
            }
          }
        };
        break;
      case 'lastExecution':
        values[valueIndex] = {
          ...values[valueIndex],
          value: undefined,
          valueFrom: {
            lastExecution: {
              results: []
            }
          }
        };
        break;
    }
    newMappings[actualMappingIndex].values = values;
    setInputMappings(newMappings);
  };

  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      <div className={twMerge('pb-0', requireIntegration && !hasRequiredIntegrations && 'pb-1')}>
        {/* Show zero state if executor type requires integrations but none are available */}
        {requireIntegration && !hasRequiredIntegrations && (
          <IntegrationZeroState
            integrationType={executor?.type || ''}
            label={getZeroStateLabel()}
            canvasId={canvasId}
            organizationId={organizationId}
          />
        )}

        {/* Form sections - only show if integrations are available or not required */}
        {hasRequiredIntegrations && (
          <>
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
              hasError={connections.length === 0 || Object.keys(validationErrors).some(key => key.startsWith('connection_'))}
            >
              {connections.map((connection, index) => (
                <div key={index}>
                  <InlineEditor
                    isEditing={connectionsEditor.editingIndex === index}
                    onSave={connectionsEditor.saveEdit}
                    onCancel={() => connectionsEditor.cancelEdit(index, (item) => {
                      if (!item.name || item.name.trim() === '') {
                        return true;
                      }

                      if (item.filters && item.filters.length > 0) {
                        const hasIncompleteFilters = item.filters.some(filter => {
                          if (filter.type === 'FILTER_TYPE_DATA') {
                            return !filter.data?.expression || filter.data.expression.trim() === '';
                          }
                          if (filter.type === 'FILTER_TYPE_HEADER') {
                            return !filter.header?.expression || filter.header.expression.trim() === '';
                          }
                          return false;
                        });
                        return hasIncompleteFilters;
                      }

                      return false;
                    })}
                    onEdit={() => connectionsEditor.startEdit(index)}
                    onDelete={() => connectionsEditor.removeItem(index)}
                    displayName={connection.name || `Connection ${index + 1}`}
                    badge={connection.type && (
                      <span className="text-xs bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-400 dark:text-zinc-300 px-2 py-0.5 rounded">
                        {connection.type?.replace?.('TYPE_', '').replace('_', ' ').toLowerCase()}
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
                        existingConnections={connections}
                      />
                    }
                  />
                </div>
              ))}
              <button
                onClick={connectionsEditor.addItem}
                className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              >
                <MaterialSymbol name="add" size="sm" />
                Add Connection
              </button>
              {validationErrors.connections && (
                <div className="text-xs text-red-600 mt-1">
                  {validationErrors.connections}
                </div>
              )}
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
              {inputs.map((input, index) => {
                // Get mappings for this specific input
                const inputMappingsForInput = inputMappings.filter(mapping =>
                  mapping.values?.some(value => value.name === input.name)
                );

                return (
                  <div key={`input-${currentStageId}-${index}`}>
                    <InlineEditor
                      isEditing={inputsEditor.editingIndex === index}
                      onSave={inputsEditor.saveEdit}
                      onCancel={() => inputsEditor.cancelEdit(index, (item) => {
                        if (!item.name || item.name.trim() === '') {
                          return true;
                        }

                        const isNewInput = index >= originalData.inputs.length ||
                          !originalData.inputs[index] ||
                          originalData.inputs[index].name !== item.name;

                        return isNewInput;
                      })}
                      onEdit={() => inputsEditor.startEdit(index)}
                      onDelete={() => {
                        const inputName = input.name;

                        inputsEditor.removeItem(index);

                        if (inputName) {
                          const updatedMappings = inputMappings.map(mapping => ({
                            ...mapping,
                            values: mapping.values?.filter(value => value.name !== inputName) || []
                          })).filter(mapping =>
                            // Remove mappings that have no values left
                            mapping.values && mapping.values.length > 0
                          );
                          setInputMappings(updatedMappings);
                        }
                      }}
                      displayName={input.name || `Input ${index + 1}`}
                      badge={
                        <div className="flex items-center gap-2">
                          {inputMappingsForInput.length > 0 && (
                            <span className="text-xs bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded">
                              {inputMappingsForInput.length} mapping{inputMappingsForInput.length !== 1 ? 's' : ''}
                            </span>
                          )}
                          {validationErrors[`input_${index}`] && (
                            <span
                              className="text-xs bg-red-100 dark:bg-red-800 text-red-800 dark:text-red-200 px-2 py-0.5 rounded flex items-center gap-1"
                              title={validationErrors[`input_${index}`]}
                            >
                              <MaterialSymbol name="error" size="sm" />
                              Mapping Error
                            </span>
                          )}
                        </div>
                      }
                      editForm={
                        <div className="space-y-4">
                          <ValidationField
                            label="Name"
                            error={validationErrors[`input_${index}`]}
                          >
                            <input
                              type="text"
                              value={input.name || ''}
                              onChange={(e) => handleInputNameChange(e.target.value, index, input)}
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

                          {/* Input Mappings for this specific input */}
                          <div className="border-t border-zinc-200 dark:border-zinc-700 pt-4">
                            <div className="flex justify-between items-center mb-3">
                              <label className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Input Mappings</label>

                            </div>

                            <div className="space-y-3">
                              {inputMappingsForInput.map((mapping) => {
                                const actualMappingIndex = inputMappings.findIndex(m => m === mapping);
                                const inputValue = mapping.values?.find(v => v.name === input.name);

                                return (
                                  <div key={actualMappingIndex} className="p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-600 rounded">
                                    <div className="space-y-3">
                                      {/* Trigger Connection */}
                                      <ValidationField
                                        label="Triggered by Connection"
                                        error={validationErrors[`inputMapping_${actualMappingIndex}`]}
                                      >
                                        <select
                                          value={mapping.when?.triggeredBy?.connection || ''}
                                          onChange={(e) => handleTriggerConnectionChange(e.target.value, actualMappingIndex, input, mapping)}
                                          className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`inputMapping_${actualMappingIndex}`]
                                            ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                                            : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                                            }`}
                                        >
                                          <option value="">Select trigger connection</option>
                                          {connections
                                            .filter(conn => mapping.when?.triggeredBy?.connection === conn.name || !inputMappings.some(mapping => mapping.when?.triggeredBy?.connection === conn.name))
                                            .map((conn, connIndex) => (
                                              <option key={connIndex} value={conn.name}>{conn.name}</option>
                                            ))}
                                        </select>
                                      </ValidationField>

                                      {/* Value Mode Toggle */}
                                      {
                                        mapping.when?.triggeredBy?.connection && (
                                          <>
                                            <div className="mb-3">
                                              <div className="space-y-2">
                                                <label className="flex items-center gap-2 text-sm">
                                                  <input
                                                    type="radio"
                                                    name={`value-mode-${actualMappingIndex}-${input.name}`}
                                                    checked={!inputValue?.valueFrom}
                                                    onChange={() => handleValueModeChange('static', actualMappingIndex, input)}
                                                    className="w-4 h-4"
                                                  />
                                                  Static Value
                                                </label>

                                                <label className="flex items-center gap-2 text-sm">
                                                  <input
                                                    type="radio"
                                                    name={`value-mode-${actualMappingIndex}-${input.name}`}
                                                    checked={!!inputValue?.valueFrom?.eventData}
                                                    onChange={() => handleValueModeChange('eventData', actualMappingIndex, input)}
                                                    className="w-4 h-4"
                                                  />
                                                  From Event Data
                                                </label>

                                                <label className="flex items-center gap-2 text-sm">
                                                  <input
                                                    type="radio"
                                                    name={`value-mode-${actualMappingIndex}-${input.name}`}
                                                    checked={!!inputValue?.valueFrom?.lastExecution}
                                                    onChange={() => handleValueModeChange('lastExecution', actualMappingIndex, input)}
                                                    className="w-4 h-4"
                                                  />
                                                  From Last Execution
                                                </label>
                                              </div>
                                            </div>
                                            {inputValue?.valueFrom?.eventData ? (
                                              /* Event Data Mode */
                                              <div className="space-y-2">
                                                <div>
                                                  <label className="block text-xs font-medium mb-1">Data Source Connection</label>
                                                  <select
                                                    value={inputValue.valueFrom.eventData.connection || ''}
                                                    onChange={(e) => inputMappingHandlers.handleEventDataConnectionChange(e.target.value, actualMappingIndex, input.name)}
                                                    className="w-full px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                                                  >
                                                    <option value="">Select data source</option>
                                                    {connections.map((conn, connIndex) => (
                                                      <option key={connIndex} value={conn.name}>{conn.name}</option>
                                                    ))}
                                                  </select>
                                                </div>
                                                <div>
                                                  <label className="block text-xs font-medium mb-1">Expression</label>
                                                  <input
                                                    value={inputValue.valueFrom.eventData.expression || ''}
                                                    onChange={(e) => inputMappingHandlers.handleEventDataExpressionChange(e.target.value, actualMappingIndex, input.name)}
                                                    placeholder="e.g., commit_sha[0:7], DEPLOY_URL"
                                                    className="w-full px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                                                  />
                                                </div>
                                              </div>
                                            ) : inputValue?.valueFrom?.lastExecution ? (
                                              /* Last Execution Mode */
                                              <div className="space-y-2">
                                                <div>
                                                  <label className="block text-xs font-medium mb-1">Required Execution Results</label>
                                                  <div className="space-y-2">
                                                    {(['RESULT_PASSED', 'RESULT_FAILED'] as const).map((result) => (
                                                      <label key={result} className="flex items-center gap-2 text-xs">
                                                        <input
                                                          type="checkbox"
                                                          checked={inputValue.valueFrom?.lastExecution?.results?.includes(result) || false}
                                                          onChange={(e) => inputMappingHandlers.handleLastExecutionChange(result, e.target.checked, actualMappingIndex, input.name)}
                                                          className="w-3 h-3"
                                                        />
                                                        {result.replace('RESULT_', '').toLowerCase()}
                                                      </label>
                                                    ))}
                                                  </div>
                                                </div>
                                              </div>
                                            ) : (
                                              /* Static Value Mode */
                                              <div>
                                                <label className="block text-xs font-medium mb-1">Static Value</label>
                                                <input
                                                  value={inputValue?.value || ''}
                                                  onChange={(e) => inputMappingHandlers.handleStaticValueChange(e.target.value, actualMappingIndex, input.name)}
                                                  placeholder="e.g., production, staging"
                                                  className="w-full px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                                                />
                                              </div>
                                            )}
                                          </>
                                        )
                                      }
                                      {/* Remove Input from Mapping Button */}
                                      <div className="flex justify-end">
                                        <button
                                          onClick={() => inputMappingHandlers.handleRemoveMapping(actualMappingIndex)}
                                          className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 text-xs"
                                        >
                                          Remove from Mapping
                                        </button>
                                      </div>
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          </div>
                          <button
                            onClick={() => inputMappingHandlers.handleAddMapping(input)}
                            className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                          >
                            <MaterialSymbol name="add" size="sm" />
                            Add Mapping
                          </button>
                        </div>
                      }
                    />
                  </div>
                );
              })}
              <button
                onClick={inputsEditor.addItem}
                className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              >
                <MaterialSymbol name="add" size="sm" />
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
                    onCancel={() => outputsEditor.cancelEdit(index, (item) => {
                      if (!item.name || item.name.trim() === '') {
                        return true;
                      }

                      const isNewOutput = index >= originalData.outputs.length ||
                        !originalData.outputs[index] ||
                        originalData.outputs[index].name !== item.name;

                      return isNewOutput;
                    })}
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
                            <label className="text-gray-900 dark:text-zinc-100" htmlFor={`required-${index}`}>Required</label>
                          </div>
                        </ValidationField>
                      </div>
                    }
                  />
                </div>
              ))}
              <button
                onClick={outputsEditor.addItem}
                className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              >
                <MaterialSymbol name="add" size="sm" />
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
                <div key={index}>
                  <InlineEditor
                    isEditing={conditionsEditor.editingIndex === index}
                    onSave={conditionsEditor.saveEdit}
                    onCancel={() => conditionsEditor.cancelEdit(index, (item) => {
                      // Remove if condition type is not set or has validation errors
                      if (!item.type || item.type === 'CONDITION_TYPE_UNKNOWN') {
                        return true;
                      }

                      const isNewCondition = index >= originalData.conditions.length ||
                        !originalData.conditions[index];

                      return isNewCondition;
                    })}
                    onEdit={() => conditionsEditor.startEdit(index)}
                    onDelete={() => conditionsEditor.removeItem(index)}
                    displayName={
                      condition.type === 'CONDITION_TYPE_APPROVAL'
                        ? `Approval (${condition.approval?.count || 1} required)`
                        : condition.type === 'CONDITION_TYPE_TIME_WINDOW'
                          ? `Time Window (${condition.timeWindow?.start || 'No start'} - ${condition.timeWindow?.end || 'No end'})`
                          : `Condition ${index + 1}`
                    }
                    badge={condition.type === 'CONDITION_TYPE_TIME_WINDOW' && condition.timeWindow?.weekDays && (
                      <span className="text-xs bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded">
                        {condition.timeWindow.weekDays.map(day => day.slice(0, 3)).join(', ')}
                      </span>
                    )}
                    editForm={
                      <div className="space-y-4">
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
                          <div className="space-y-4">
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
                                  <label key={day} className="flex flex-col items-center gap-1 p-2 border border-zinc-300 dark:border-zinc-600 rounded text-xs bg-white dark:bg-zinc-800">
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
                                    <span className="text-zinc-900 dark:text-zinc-100">{day.slice(0, 3)}</span>
                                  </label>
                                ))}
                              </div>
                            </ValidationField>
                          </div>
                        )}
                      </div>
                    }
                  />
                </div>
              ))}
              <button
                onClick={conditionsEditor.addItem}
                className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              >
                <MaterialSymbol name="add" size="sm" />
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
                    onCancel={() => secretsEditor.cancelEdit(index, (item) => {
                      if (!item.name || item.name.trim() === '') {
                        return true;
                      }

                      const isNewSecret = index >= originalData.secrets.length ||
                        !originalData.secrets[index];

                      return isNewSecret;
                    })}
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
                            placeholder="eg. MY_SECRET"
                            className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`secret_${index}`]
                              ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                              : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                              }`}
                          />
                        </ValidationField>

                        <div className="space-y-2">
                          <ValidationField label="Reference Secret">
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
                          </ValidationField>
                          <ValidationField label="Key">
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
                      </div>
                    }
                  />
                </div>
              ))}
              <button
                onClick={secretsEditor.addItem}
                className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              >
                <MaterialSymbol name="add" size="sm" />
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
              countLabel="executor"
              className={!openSections.includes('executor') ? 'rounded-b-2xl border-b-0' : ''}
              hasError={
                isExecutorMisconfigured() ||
                Object.keys(validationErrors).some(key => key.startsWith('executor')) ||
                Object.values(fieldErrors).some(Boolean)
              }
            >
              <div className="space-y-4">
                <ValidationField label="Executor name">
                  <input
                    type="text"
                    value={executor.name || ''}
                    onChange={(e) => setExecutor(prev => ({ ...prev, name: e.target.value }))}
                    placeholder="${{ inputs.VERSION }} deployment"
                    className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                  />

                </ValidationField>

                {executor.type === 'semaphore' && (
                  <div className="space-y-4">
                    <ValidationField
                      label="Integration"
                      error={validationErrors.executorIntegration}
                    >
                      <select
                        value={executor.integration?.name || ''}
                        onChange={(e) => updateExecutorIntegration(e.target.value)}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorIntegration
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
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
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                          No Semaphore integrations available. Create one in canvas settings.
                        </div>
                      )}
                    </ValidationField>

                    <ValidationField
                      label="Project Name"
                      error={validationErrors.executorProject || fieldErrors.project}
                    >
                      <input
                        type="text"
                        value={(executor.resource?.name as string) || ''}
                        onChange={(e) => {
                          if (executor.resource?.type !== 'project')
                            updateExecutorResource('type', 'project');

                          updateExecutorResource('name', e.target.value);
                          if (fieldErrors.project) {
                            setFieldErrors(prev => {
                              // eslint-disable-next-line @typescript-eslint/no-unused-vars
                              const { project, ...rest } = prev;
                              return rest;
                            });
                          }
                        }}
                        placeholder="my-semaphore-project"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${fieldErrors.project
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField label="Execution Type">
                      <ControlledTabs
                        className="text-left m-0 w-full"
                        buttonClasses='w-full'
                        tabs={[
                          { id: 'workflow', label: 'Workflow' },
                          { id: 'task', label: 'Task' },
                        ]}
                        variant="pills"
                        activeTab={semaphoreExecutionType}
                        onTabChange={(tabId) => updateSemaphoreExecutionType(tabId as 'workflow' | 'task')}
                      />
                    </ValidationField>

                    {semaphoreExecutionType === 'task' && (
                      <ValidationField label="Task">
                        <input
                          type="text"
                          value={(executor.spec?.task as string) || ''}
                          onChange={(e) => updateExecutorField('task', e.target.value)}
                          placeholder="my-task"
                          className="w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500"
                        />
                      </ValidationField>
                    )}

                    <ValidationField
                      label="Ref"
                      error={validationErrors.executorRef}
                    >
                      <input
                        type="text"
                        value={(executor.spec?.ref as string) || ''}
                        onChange={(e) => updateExecutorField('ref', e.target.value)}
                        placeholder="refs/heads/main"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorRef
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField
                      label="Pipeline File"
                      error={validationErrors.executorPipelineFile}
                    >
                      <input
                        type="text"
                        value={(executor.spec?.pipelineFile as string) || ''}
                        onChange={(e) => updateExecutorField('pipelineFile', e.target.value)}
                        placeholder=".semaphore/semaphore.yml"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorPipelineFile
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField label="Parameters">
                      <div className="space-y-2">
                        {semaphoreParameters.map((param) => (
                          <div key={param.id} className="w-full flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                            <input
                              type="text"
                              value={param.key}
                              onChange={(e) => updateExecutorParameter(param.id, e.target.value, param.value)}
                              placeholder="Parameter name"
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <input
                              type="text"
                              value={param.value}
                              onChange={(e) => updateExecutorParameter(param.id, param.key, e.target.value)}
                              placeholder="Parameter value"
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <button
                              onClick={() => removeExecutorParameter(param.id)}
                              className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:text-zinc-300"
                            >
                              <MaterialSymbol name="delete" size="sm" />
                            </button>
                          </div>
                        ))}
                        <button
                          onClick={addExecutorParameter}
                          className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                        >
                          <MaterialSymbol name="add" size="sm" />
                          Add Parameter
                        </button>
                      </div>
                    </ValidationField>
                  </div>
                )}

                {executor.type === 'github' && (
                  <div className="space-y-4">
                    <ValidationField
                      label="Integration"
                      error={validationErrors.executorIntegration}
                    >
                      <select
                        value={executor.integration?.name || ''}
                        onChange={(e) => updateExecutorIntegration(e.target.value)}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorIntegration
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      >
                        <option value="">Select an integration...</option>
                        {getAllIntegrations()
                          .filter(integration => integration.spec?.type === 'github')
                          .map((integration) => (
                            <option key={integration.metadata?.id} value={integration.metadata?.name}>
                              {integration.metadata?.name}
                            </option>
                          ))}
                      </select>
                      {getAllIntegrations().filter(int => int.spec?.type === 'github').length === 0 && (
                        <div className="text-xs text-zinc-500 mt-1">
                          No GitHub integrations available. Create one in canvas settings.
                        </div>
                      )}
                    </ValidationField>

                    <ValidationField
                      label="Repository Name"
                      error={validationErrors.executorRepository || fieldErrors.repository}
                    >
                      <input
                        type="text"
                        value={(executor.resource?.name as string) || ''}
                        onChange={(e) => {
                          if (executor.resource?.type !== 'repository')
                            updateExecutorResource('type', 'repository');

                          updateExecutorResource('name', e.target.value)

                          setFieldErrors(prev => ({ ...prev, repository: '' }));
                        }}
                        placeholder="my-repository"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorRepository
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField
                      label="Workflow"
                      error={validationErrors.executorWorkflow || fieldErrors.workflow}
                    >
                      <input
                        type="text"
                        value={(executor.spec?.workflow as string) || ''}
                        onChange={(e) => {
                          updateExecutorField('workflow', e.target.value);
                          setFieldErrors(prev => ({ ...prev, workflow: '' }));
                        }}
                        placeholder=".github/workflows/task.yml"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorWorkflow
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField
                      label="Ref"
                      error={validationErrors.executorRef}
                    >
                      <input
                        type="text"
                        value={(executor.spec?.ref as string) || ''}
                        onChange={(e) => updateExecutorField('ref', e.target.value)}
                        placeholder="main"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorRef
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField label="Inputs">
                      <div className="space-y-2">
                        {githubInputs.map((input) => (
                          <div key={input.id} className="w-full flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                            <input
                              type="text"
                              value={input.key}
                              onChange={(e) => updateExecutorInput(input.id, e.target.value, input.value)}
                              placeholder="Input name"
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <input
                              type="text"
                              value={input.value}
                              onChange={(e) => updateExecutorInput(input.id, input.key, e.target.value)}
                              placeholder="Input value"
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <button
                              onClick={() => removeExecutorInput(input.id)}
                              className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:text-zinc-300"
                            >
                              <MaterialSymbol name="delete" size="sm" />
                            </button>
                          </div>
                        ))}
                        <button
                          onClick={addExecutorInput}
                          className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                        >
                          <MaterialSymbol name="add" size="sm" />
                          Add Input
                        </button>
                      </div>
                    </ValidationField>
                  </div>
                )}

                {executor.type === 'http' && (
                  <div className="space-y-4">
                    <ValidationField
                      label="URL"
                      error={validationErrors.executorUrl}
                    >
                      <input
                        type="text"
                        value={(executor.spec?.url as string) || ''}
                        onChange={(e) => updateExecutorField('url', e.target.value)}
                        placeholder="https://api.example.com/endpoint"
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors.executorUrl
                          ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>

                    <ValidationField label="Payload (JSON)">
                      <textarea
                        value={payloadDisplay}
                        onChange={(e) => {
                          setPayloadDisplay(e.target.value);
                          try {
                            const parsed = JSON.parse(e.target.value);
                            updateExecutorField('payload', parsed);
                          } catch {
                            // Keep the display value but don't update payload until valid JSON
                          }
                        }}
                        placeholder='{\n  "key1": "value1",\n  "key2": "{{ inputs.KEY2 }}"\n}'
                        rows={6}
                        className="nodrag w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 border-zinc-300 dark:border-zinc-600 focus:ring-blue-500 font-mono"
                      />
                    </ValidationField>

                    <ValidationField label="Headers">
                      <div className="space-y-2">
                        {httpHeaders.map((header) => (
                          <div key={header.id} className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                            <input
                              type="text"
                              value={header.key}
                              onChange={(e) => updateExecutorHeader(header.id, e.target.value, header.value)}
                              placeholder="Header name"
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <input
                              type="text"
                              value={header.value}
                              onChange={(e) => updateExecutorHeader(header.id, header.key, e.target.value)}
                              placeholder="Header value"
                              className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700"
                            />
                            <button
                              onClick={() => removeExecutorHeader(header.id)}
                              className="text-zinc-600 dark:text-zinc-400 hover:text-zinc-700 dark:text-zinc-300"
                            >
                              <MaterialSymbol name="delete" size="sm" />
                            </button>
                          </div>
                        ))}
                        <button
                          onClick={addExecutorHeader}
                          className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                        >
                          <MaterialSymbol name="add" size="sm" />
                          Add Header
                        </button>
                      </div>
                    </ValidationField>

                    <ValidationField label="Response Policy - Success Status Codes">
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
                    </ValidationField>
                  </div>
                )}
              </div>

              <div className="flex justify-end gap-2 pt-2 pb-3">
                <Button
                  className='flex items-center border-0'
                  outline
                  onClick={() => {
                    setOpenSections(prev => prev.filter(section => section !== 'executor'));
                  }}
                >
                  <MaterialSymbol name="close" size="sm" data-slot="icon" />
                </Button>
                <Button
                  className='flex items-center'
                  color="white"
                  onClick={() => {
                    const executorErrors = validateExecutor(executor);
                    if (Object.keys(executorErrors).length === 0) {
                      setOpenSections(prev => prev.filter(section => section !== 'executor'));
                    } else {
                      setValidationErrors(prev => ({
                        ...prev,
                        ...executorErrors
                      }));
                    }
                  }}
                >
                  <MaterialSymbol name="check" size="sm" data-slot="icon" />
                </Button>
              </div>
            </EditableAccordionSection>
          </>
        )}
      </div>
    </div>
  );
}