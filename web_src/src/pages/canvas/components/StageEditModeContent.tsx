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
import { ValidationField } from '../../../components/ValidationField';
import { ConnectionSelector } from './shared/ConnectionSelector';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import IntegrationZeroState from '@/components/IntegrationZeroState';
import { createInputMappingHandlers } from '../utils/inputMappingHandlers';
import { twMerge } from 'tailwind-merge';
import { OutputsTooltip } from '@/components/Tooltip/outputs-tooltip';
import { OutputsHelpTooltip } from '@/components/Tooltip/outputs-help-tooltip';
import { ConnectionsTooltip } from '@/components/Tooltip/connections-tooltip';
import { InputsTooltip } from '@/components/Tooltip/inputs-tooltip';
import { ConditionsTooltip } from '@/components/Tooltip/conditions-tooltip';
import { SecretsTooltip } from '@/components/Tooltip/secrets-tooltip';
import { ExecutorTooltip } from '@/components/Tooltip/executor-tooltip';
import { InputMappingsTooltip } from '@/components/Tooltip/input-mappings-tooltip';
import { StaticValueTooltip } from '@/components/Tooltip/static-value-tooltip';
import { ExpressionTooltip } from '@/components/Tooltip/expression-tooltip';
import { DryRunTooltip } from '@/components/Tooltip/dry-run-tooltip';
import { RequiredExecutionResultsTooltip } from '@/components/Tooltip/required-execution-results-tooltip';
import { NodeContentWrapper } from './shared/NodeContentWrapper';
import { Switch } from '@/components/Switch/switch';
import { ExecutorFormSection } from './stage/ExecutorFormSection';

interface StageEditModeContentProps {
  data: StageNodeType['data'];
  currentStageId?: string;
  canvasId: string;
  organizationId: string;
  isNewStage?: boolean;
  dirtyByUser?: boolean;
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
    dryRun: boolean;
    isValid: boolean
  }) => void;
  onTriggerSectionValidation?: { current: ((hasFieldErrors?: boolean) => void) | null };
  onStageNameChange?: (name: string) => void;
  integrationError?: boolean;
  onFieldErrorsChange?: (setFieldErrors: React.Dispatch<React.SetStateAction<Record<string, string>>>) => void;
}

export function StageEditModeContent({ data, currentStageId, canvasId, organizationId, isNewStage, onDataChange, onTriggerSectionValidation, integrationError = false, onFieldErrorsChange }: StageEditModeContentProps) {
  // Component-specific state
  const [inputs, setInputs] = useState<SuperplaneInputDefinition[]>(data.inputs || []);
  const [outputs, setOutputs] = useState<SuperplaneOutputDefinition[]>(data.outputs || []);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [connectionFilterErrors, setConnectionFilterErrors] = useState<Record<number, number[]>>({});
  const [connections, setConnections] = useState<SuperplaneConnection[]>(data.connections || []);
  const [secrets, setSecrets] = useState<SuperplaneValueDefinition[]>(data.secrets || []);
  const [conditions, setConditions] = useState<SuperplaneCondition[]>(data.conditions || []);
  const [executor, setExecutor] = useState<SuperplaneExecutor>({
    type: data.executor?.type || '',
    spec: data.executor?.spec || {},
    resource: data.executor?.resource || {},
    integration: data.executor?.integration || {},
  });
  const [inputMappings, setInputMappings] = useState<SuperplaneInputMapping[]>(data.inputMappings || []);
  const [dryRun, setDryRun] = useState<boolean>(data.dryRun || false);

  // Pass setFieldErrors to parent component
  useEffect(() => {
    if (onFieldErrorsChange) {
      onFieldErrorsChange(setFieldErrors);
    }
  }, [onFieldErrorsChange]);

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

  // Helper functions
  const hasErrorsWithPrefix = useCallback((errors: Record<string, string>, prefixes: string | string[]) => {
    const prefixArray = Array.isArray(prefixes) ? prefixes : [prefixes];
    return Object.keys(errors).some(key =>
      prefixArray.some(prefix => key.startsWith(prefix))
    );
  }, []);

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

    // Skip executor spec validation if in dry run mode
    if (dryRun) {
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
    } else if (executor.type === 'noop') {
      // NoOp executor doesn't require any configuration
    }

    return errors;
  }, [dryRun]);

  const validateInputMappings = useCallback((currentErrors: Record<string, string>) => {
    const inputMappingErrors = new Map<string, string[]>();
    const specificMappingErrors = new Map<string, string>();

    inputMappings.forEach((mapping, mappingIndex) => {
      const mappingErrors: string[] = [];
      if (!mapping.when?.triggeredBy?.connection) {
        mappingErrors.push('Trigger connection is required');
      }

      if (inputs.length > 0) {
        const mappingInputNames = mapping.values?.map(v => v.name) || [];
        const missingInputs = inputs
          .map(input => input.name)
          .filter(inputName => inputName && !mappingInputNames.includes(inputName));

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

      if (mapping.values) {
        mapping.values.forEach((value) => {
          const hasStaticValue = value.value !== undefined && value.value !== '';
          const hasEventDataValue = value.valueFrom?.eventData?.expression !== undefined && value.valueFrom.eventData.expression !== '';
          const hasLastExecutionValue = value.valueFrom?.lastExecution?.results !== undefined && value.valueFrom.lastExecution.results.length > 0;

          if (!hasStaticValue && !hasEventDataValue && !hasLastExecutionValue) {
            const inputIndex = inputs.findIndex(inp => inp.name === value.name);
            if (inputIndex !== -1) {
              const errorKey = `mapping_${mappingIndex}_input_${inputIndex}`;
              specificMappingErrors.set(errorKey, 'No value defined for this input');

              const currentErrors = inputMappingErrors.get(`input_${inputIndex}`) || [];
              currentErrors.push(`No value defined in mapping for connection "${mapping.when?.triggeredBy?.connection || 'Unknown'}"`);
              inputMappingErrors.set(`input_${inputIndex}`, currentErrors);
            }
          }
        });
      }

      if (mappingErrors.length > 0) {
        currentErrors[`inputMapping_${mappingIndex}`] = mappingErrors.join(', ');
      }
    });

    specificMappingErrors.forEach((error, key) => {
      currentErrors[key] = error;
    });

    return inputMappingErrors;
  }, [inputMappings, inputs]);

  const validateConnections = useCallback((connections: SuperplaneConnection[], errors: Record<string, string>) => {
    connections.forEach((connection, index) => {
      const connectionErrors = connectionManager.validateConnection(connection);
      if (connectionErrors.length > 0) {
        errors[`connection_${index}`] = connectionErrors.join(', ');
      }
    });


    if (connections.length === 0) {
      errors.connections = 'At least one connection is required';
    } else {
      delete errors.connections;
    }

    return errors;
  }, []);

  const validateAllFields = useCallback((showUiErrors: boolean = true) => {
    let errors: Record<string, string> = {};

    // Validate all arrays
    inputs.forEach((input, index) => {
      const inputErrors = validateInput(input, index);
      if (inputErrors.length > 0) {
        errors[`input_name_${index}`] = inputErrors.join(', ');
      }
    });

    outputs.forEach((output, index) => {
      const outputErrors = validateOutput(output, index);
      if (outputErrors.length > 0) {
        errors[`output_${index}`] = outputErrors.join(', ');
      }
    });

    errors = validateConnections(connections, errors);


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

    const inputMappingErrors = validateInputMappings(errors);

    inputMappingErrors.forEach((errorList, inputKey) => {
      const mappingKey = inputKey.replace('input_', 'input_mapping_');
      errors[mappingKey] = errorList.join(', ');
    });

    // Comprehensive executor validation
    const executorErrors = validateExecutor(executor);
    errors = {
      ...errors,
      ...executorErrors
    };

    if (showUiErrors) {
      setValidationErrors(errors);
    }

    return errors;
  }, [inputs, outputs, connections, secrets, conditions, validateInput, validateOutput, validateSecret, validateCondition, validateInputMappings, validateExecutor, executor, validateConnections]);



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
      dryRun: data.dryRun || false,
      isValid: true
    },
    onDataChange,
    validateAllFields: () => {
      const executorErrors = validateExecutor(executor);

      if (Object.keys(executorErrors).length > 0) {
        setOpenSections(prev => [...prev, 'executor']);
      }
      return Object.keys(validateAllFields(false)).length === 0;
    },
  });

  // Auto-select first available integration for new stages
  useEffect(() => {
    if (isNewStage && executor.type && !executor.integration?.name && validationErrors.executorIntegration) {
      const availableIntegrations = [...canvasIntegrations, ...orgIntegrations];
      const firstIntegration = availableIntegrations.find(int => int.spec?.type === executor.type);

      if (firstIntegration?.metadata?.name) {
        setExecutor(prev => ({
          ...prev,
          integration: {
            name: firstIntegration.metadata?.name,
            domainType: firstIntegration.metadata?.domainType
          }
        }));
        setValidationErrors(prev => ({
          ...prev,
          executorIntegration: ''
        }));
      }
    }
  }, [isNewStage, executor.type, executor.integration?.name, canvasIntegrations, orgIntegrations, validationErrors.executorIntegration, setValidationErrors]);

  // Clear executor validation errors when dry run mode is enabled
  useEffect(() => {
    if (dryRun) {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        // Remove all executor-related validation errors
        Object.keys(newErrors).forEach(key => {
          if (key.startsWith('executor')) {
            delete newErrors[key];
          }
        });
        return newErrors;
      });
    }
  }, [dryRun, setValidationErrors]);

  const triggerSectionValidation = useCallback((hasFieldErrors: boolean = false) => {
    const errors = validateAllFields();

    const sectionsToOpen = [...openSections];

    const connectionsNeedConfiguration = connections.length === 0;
    if (connectionsNeedConfiguration && !sectionsToOpen.includes('connections')) {
      sectionsToOpen.push('connections');
    }

    const inputErrors = hasErrorsWithPrefix(errors, ['input_name_', 'input_mapping_']);
    if (inputErrors && !sectionsToOpen.includes('inputs')) {
      sectionsToOpen.push('inputs');
    }

    const secretErrors = hasErrorsWithPrefix(errors, 'secret_');
    if (secretErrors && !sectionsToOpen.includes('secrets')) {
      sectionsToOpen.push('secrets');
    }

    const outputErrors = hasErrorsWithPrefix(errors, 'output_');
    if (outputErrors && !sectionsToOpen.includes('outputs')) {
      sectionsToOpen.push('outputs');
    }

    const executorErrors = validateExecutor(executor);
    if (Object.keys(executorErrors).length > 0 || hasFieldErrors) {
      if (!sectionsToOpen.includes('executor')) {
        sectionsToOpen.push('executor');
      }
      setValidationErrors(prev => ({
        ...prev,
        ...errors,
        ...executorErrors
      }));
    }

    setOpenSections(sectionsToOpen);
  }, [connections.length, executor, openSections, setOpenSections, setValidationErrors, validateAllFields, validateExecutor, hasErrorsWithPrefix]);

  // Expose the trigger function to parent
  useEffect(() => {
    if (onTriggerSectionValidation) {
      onTriggerSectionValidation.current = triggerSectionValidation;
    }
  }, [triggerSectionValidation, onTriggerSectionValidation]);

  // Initialize open sections
  useEffect(() => {
    const sectionsToOpen = ['general'];
    let newValidationErrors: Record<string, string> = {};

    const connectionErrors = validateConnections(connections, {});
    if (Object.keys(connectionErrors).length > 0) {
      sectionsToOpen.push('connections');
      newValidationErrors = {
        ...connectionErrors
      };
    }

    const inputErrors = hasErrorsWithPrefix(validationErrors, ['input_name_', 'input_mapping_']);
    if (inputErrors) {
      sectionsToOpen.push('inputs');
    }

    const executorErrors = validateExecutor(executor);

    if (Object.keys(executorErrors).length > 0 || (isNewStage && executor.resource?.type && executor.type !== 'noop')) {
      sectionsToOpen.push('executor');
      newValidationErrors = {
        ...newValidationErrors,
        ...executorErrors
      };
    }
    setValidationErrors(newValidationErrors);

    setOpenSections(sectionsToOpen);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [setOpenSections, isNewStage, executor.type]);

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
    createNewItem: () => ({ name: `INPUT_${inputs.length + 1}`, description: '' }),
    validateItem: validateInput,
    setValidationErrors,
    errorPrefix: 'input'
  });

  const outputsEditor = useArrayEditor({
    items: outputs,
    setItems: setOutputs,
    createNewItem: () => ({ name: `OUTPUT_${outputs.length + 1}`, description: '', required: false }),
    validateItem: () => [],
    setValidationErrors: () => { },
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
        dryRun: data.dryRun || false,
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
        setDryRun(incomingData.dryRun || false);
      }
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

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
        inputMappings,
        dryRun
      });
    }// eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data.name, data.description, inputs, outputs, connections, executor, secrets, conditions, inputMappings, dryRun, onDataChange]);

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


  /**
   * Problem: When users type input names, temporary name conflicts can occur (e.g., typing "input1"
   * might temporarily show "input" which conflicts with another existing input). This causes input
   * mappings to get mixed up between different inputs.
   *
   * Solution: Uses an invisible Unicode character (Zero-Width Non-Joiner U+200C) as a prefix to
   * temporarily disambiguate conflicting names. This character is completely invisible to users
   * but makes names technically unique for the system.
   * This ensures input mappings never get mixed up during typing while keeping the UX seamless.
   */
  const handleInputNameChange = (newName: string, index: number, input: SuperplaneInputDefinition) => {
    const oldName = input.name;

    let hasNameConflict = inputs.some((otherInput, otherIndex) =>
      otherIndex !== index && otherInput.name === newName && newName !== ''
    );

    if (newName.startsWith('\u200C')) {
      const unprefixName = newName.slice(1);
      const unprefixNameConflict = inputs.some((otherInput, otherIndex) =>
        otherIndex !== index && otherInput.name === unprefixName && unprefixName !== ''
      );

      if (!unprefixNameConflict) {
        hasNameConflict = false;
        newName = unprefixName;
      }
    }

    const finalName = hasNameConflict ? `\u200C${newName}` : newName;

    inputsEditor.updateItem(index, 'name', finalName);

    if (finalName !== oldName) {
      const updatedMappings = inputMappings.map(mapping => ({
        ...mapping,
        values: mapping.values?.map(value =>
          value.name === oldName
            ? { ...value, name: finalName }
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
              connection: newMappings[actualMappingIndex].when?.triggeredBy?.connection,
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
              results: [
                'RESULT_PASSED',
                'RESULT_FAILED'
              ]
            }
          }
        };
        break;
    }
    newMappings[actualMappingIndex].values = values;
    setInputMappings(newMappings);
  };

  const handleConnectionSave = useCallback(() => {
    const savedConnectionIndex = connectionsEditor.editingIndex;
    const savedConnection = savedConnectionIndex !== null ?
      connections[savedConnectionIndex] : null;

    if (savedConnection && savedConnectionIndex !== null) {
      const connectionErrors = connectionManager.validateConnection(savedConnection);
      if (connectionErrors.length > 0) {
        setValidationErrors(prev => ({
          ...prev,
          [`connection_${savedConnectionIndex}`]: connectionErrors.join(', ')
        }));
        setConnectionFilterErrors(prev => ({
          ...prev,
          [savedConnectionIndex]: connectionManager.getConnectionFilterErrors(savedConnection)
        }));
        return;
      } else {
        setValidationErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors[`connection_${savedConnectionIndex}`];
          return newErrors;
        });

        setConnectionFilterErrors(prev => {
          const newErrors = { ...prev };
          delete newErrors[savedConnectionIndex];
          return newErrors;
        });
      }
    }

    connectionsEditor.saveEdit();

    // Clear the general "connections" error if we now have at least one connection
    if (connections.length > 0) {
      setValidationErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors.connections;
        return newErrors;
      });
    }

    if (savedConnection?.name && inputs.length > 0) {
      const existingMapping = inputMappings.find(mapping =>
        mapping.when?.triggeredBy?.connection === savedConnection.name
      );

      if (!existingMapping) {
        const newMapping = {
          when: { triggeredBy: { connection: savedConnection.name } },
          values: inputs.map(input => ({
            name: input.name,
            value: ''
          }))
        };
        setInputMappings(prev => [...prev, newMapping]);
      }
    }
  }, [connectionsEditor, connections, inputs, inputMappings, setInputMappings, connectionManager, setValidationErrors]);

  const handleConnectionDelete = useCallback((index: number, connection: SuperplaneConnection) => {
    const connectionName = connection.name;

    connectionsEditor.removeItem(index);

    if (connectionName) {
      const updatedMappings = inputMappings.filter(mapping =>
        mapping.when?.triggeredBy?.connection !== connectionName
      );
      setInputMappings(updatedMappings);
    }
  }, [connectionsEditor, inputMappings, setInputMappings]);

  const handleInputDelete = useCallback((index: number, input: SuperplaneInputDefinition) => {
    const inputName = input.name;

    inputsEditor.removeItem(index);

    if (inputName) {
      const updatedMappings = inputMappings.map(mapping => ({
        ...mapping,
        values: mapping.values?.filter(value => value.name !== inputName && value.name !== '') || []
      }))
        .filter(mapping => mapping.values?.length > 0);
      setInputMappings(updatedMappings);
    }
  }, [inputsEditor, inputMappings, setInputMappings]);

  const handleInputAdd = useCallback(() => {
    inputsEditor.addItem();

    if (connections.length > 0) {
      const inputsCount = inputs.length;
      const newInputName = `INPUT_${inputsCount + 1}`;


      if (inputMappings.length === 0) {
        const newMappings = connections.map(connection => ({
          when: { triggeredBy: { connection: connection.name } },
          values: [{ name: newInputName, value: '' }]
        }));
        setInputMappings(newMappings);
        return;
      }

      const updatedMappings = inputMappings.map(mapping => ({
        ...mapping,
        values: [
          ...(mapping.values || []),
          { name: newInputName, value: '' }
        ]
      }));
      setInputMappings(updatedMappings);
    }
  }, [inputsEditor, connections, inputMappings, setInputMappings, inputs]);

  return (
    <NodeContentWrapper nodeId={currentStageId}>
      <div className={twMerge('pb-0', requireIntegration && !hasRequiredIntegrations && !dryRun && 'pb-1')}>

        {/* DryRun Toggle - hidden for noop stages */}
        {executor.type != 'noop' && (
          <div className="mb-4 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <MaterialSymbol name="science" size="sm" className="text-amber-600 dark:text-amber-400" />
                <span className="text-sm font-medium text-amber-800 dark:text-amber-200">
                  Dry Run Mode
                </span>
                <DryRunTooltip className="flex items-center" />
              </div>
              <Switch
                checked={dryRun}
                onChange={checked => {
                  setDryRun(checked);
                  setFieldErrors(() => { return {}});
                }}
                color="indigo"
                aria-label="Toggle dry run mode"
              />
            </div>
          </div>
        )}

        {/* Show zero state if executor type requires integrations but none are available, except in dry-run mode */}
        {requireIntegration && !hasRequiredIntegrations && !dryRun && (
          <IntegrationZeroState
            integrationType={executor?.type || ''}
            label={getZeroStateLabel()}
            canvasId={canvasId}
            organizationId={organizationId}
            hasError={integrationError}
          />
        )}

        {/* Form sections - only show if integrations are available or not required, or in dry-run mode */}
        {(hasRequiredIntegrations || dryRun) && (
          <>

            {/* Connections Section */}
            <EditableAccordionSection
              id="connections"
              title={
                <div className="flex items-center gap-2">
                  Connections
                  <ConnectionsTooltip />
                </div>
              }
              isOpen={openSections.includes('connections')}
              onToggle={handleAccordionToggle}
              isModified={isSectionModified(connections, 'connections')}
              onRevert={revertSection}
              count={connections.length}
              countLabel="connections"
              hasError={connections.length === 0 || hasErrorsWithPrefix(validationErrors, 'connection_')}
            >
              {connections.map((connection, index) => (
                <div key={index}>
                  <InlineEditor
                    isEditing={connectionsEditor.editingIndex === index}
                    onSave={handleConnectionSave}
                    onCancel={() => connectionsEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                    onEdit={() => connectionsEditor.startEdit(index)}
                    onDelete={() => handleConnectionDelete(index, connection)}
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
                        onConnectionUpdate={(index, type, name) => {
                          connectionManager.updateConnection(index, type, name);
                          setInputMappings(prev => {
                            const updatedMappings = prev.map(mapping => {
                              if (mapping.when?.triggeredBy?.connection === connection.name) {
                                const values = mapping.values?.map(value => {
                                  if (value.valueFrom && value.valueFrom?.eventData?.connection === connection.name) {
                                    return { ...value, valueFrom: { ...value.valueFrom, eventData: { ...value.valueFrom.eventData, connection: name } } };
                                  }
                                  return value;
                                });
                                return {
                                  ...mapping,
                                  when: { triggeredBy: { connection: name } },
                                  values
                                };
                              }
                              return mapping;
                            });
                            return updatedMappings;
                          });
                        }}
                        onFilterAdd={connectionManager.addFilter}
                        onFilterUpdate={connectionManager.updateFilter}
                        onFilterRemove={connectionManager.removeFilter}
                        onFilterOperatorToggle={connectionManager.toggleFilterOperator}
                        currentEntityId={currentStageId}
                        validationError={validationErrors[`connection_${index}`]}
                        filterErrors={connectionFilterErrors[index] || []}
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
              title={
                <div className="flex items-center gap-2">
                  Inputs
                  <InputsTooltip />
                </div>
              }
              isOpen={openSections.includes('inputs')}
              onToggle={handleAccordionToggle}
              isModified={isSectionModified(inputs, 'inputs')}
              onRevert={revertSection}
              count={inputs.length}
              countLabel="inputs"
              hasError={hasErrorsWithPrefix(validationErrors, ['input_name_', 'input_mapping_'])}
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
                      onSave={() => {
                        const errors = validateAllFields();

                        if (errors[`input_name_${index}`] || errors[`input_mapping_${index}`]) {
                          return;
                        }

                        inputsEditor.saveEdit()
                      }}
                      onCancel={() => inputsEditor.cancelEdit(index, (item) => !item.name || item.name.trim() === '')}
                      onEdit={() => inputsEditor.startEdit(index)}
                      onDelete={() => handleInputDelete(index, input)}
                      displayName={input.name || `Input ${index + 1}`}
                      badge={
                        <div className="flex items-center gap-2">
                          {inputMappingsForInput.length > 0 && (
                            <span className="text-xs bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200 px-2 py-0.5 rounded">
                              {inputMappingsForInput.length} mapping{inputMappingsForInput.length !== 1 ? 's' : ''}
                            </span>
                          )}
                          {validationErrors[`input_mapping_${index}`] && (
                            <span
                              className="text-xs bg-red-100 dark:bg-red-800 text-red-800 dark:text-red-200 px-2 py-0.5 rounded flex items-center gap-1"
                              title={validationErrors[`input_mapping_${index}`]}
                            >
                              <MaterialSymbol name="error" size="sm" />
                              Mapping Error
                            </span>
                          )}
                          {validationErrors[`input_name_${index}`] && (
                            <span
                              className="text-xs bg-red-100 dark:bg-red-800 text-red-800 dark:text-red-200 px-2 py-0.5 rounded flex items-center gap-1"
                              title={validationErrors[`input_name_${index}`]}
                            >
                              <MaterialSymbol name="error" size="sm" />
                              Name Error
                            </span>
                          )}
                        </div>
                      }
                      editForm={
                        <div className="space-y-4">
                          <ValidationField label="Name" error={validationErrors[`input_name_${index}`]} required>
                            <input
                              type="text"
                              value={input.name || ''}
                              onChange={(e) => {
                                handleInputNameChange(e.target.value, index, input)
                              }}
                              placeholder="Input name"
                              className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${validationErrors[`input_name_${index}`]
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
                            <ValidationField
                              label={
                                <div className="flex items-center gap-2">
                                  Input Mappings
                                  <InputMappingsTooltip />
                                </div>
                              }
                              error={validationErrors[`input_mapping_${index}`]}
                            >
                              <div className="space-y-3">
                                {inputMappingsForInput.map((mapping) => {
                                  const actualMappingIndex = inputMappings.findIndex(m => m === mapping);
                                  const inputValue = mapping.values?.find(v => v.name === input.name);
                                  const mappingErrorKey = `mapping_${actualMappingIndex}_input_${index}`;
                                  const hasMappingError = validationErrors[mappingErrorKey];

                                  return (
                                    <div key={actualMappingIndex} className={`p-3 bg-zinc-50 dark:bg-zinc-800 border rounded ${hasMappingError
                                      ? 'border-red-300 dark:border-red-600 bg-red-50 dark:bg-red-900/20'
                                      : 'border-zinc-200 dark:border-zinc-600'}`}>
                                      <div className="space-y-3">
                                        {/* Trigger Connection */}
                                        <div className="space-y-1">
                                          <label className="block text-sm font-medium text-zinc-600 dark:text-zinc-400">
                                            Triggered by Connection
                                          </label>
                                          <div className="px-3 py-2 bg-zinc-100 dark:bg-zinc-700 border border-zinc-300 dark:border-zinc-600 rounded-md text-zinc-900 dark:text-zinc-100 text-sm">
                                            {mapping.when?.triggeredBy?.connection || 'No connection assigned'}
                                          </div>
                                        </div>

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
                                                    <div className="flex items-center gap-1">
                                                      Static Value
                                                      <StaticValueTooltip>
                                                        <div className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors cursor-help">
                                                          <MaterialSymbol name="help" size="sm" />
                                                        </div>
                                                      </StaticValueTooltip>
                                                    </div>
                                                  </label>

                                                  <label className="flex items-center gap-2 text-sm">
                                                    <input
                                                      type="radio"
                                                      name={`value-mode-${actualMappingIndex}-${input.name}`}
                                                      checked={!!inputValue?.valueFrom?.eventData}
                                                      onChange={() => handleValueModeChange('eventData', actualMappingIndex, input)}
                                                      className="w-4 h-4"
                                                    />
                                                    <div className="flex items-center gap-1">
                                                      From Event Data
                                                      <ExpressionTooltip>
                                                        <div className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors cursor-help">
                                                          <MaterialSymbol name="help" size="sm" />
                                                        </div>
                                                      </ExpressionTooltip>
                                                    </div>
                                                  </label>

                                                  <label className="flex items-center gap-2 text-sm">
                                                    <input
                                                      type="radio"
                                                      name={`value-mode-${actualMappingIndex}-${input.name}`}
                                                      checked={!!inputValue?.valueFrom?.lastExecution}
                                                      onChange={() => handleValueModeChange('lastExecution', actualMappingIndex, input)}
                                                      className="w-4 h-4"
                                                    />
                                                    <div className="flex items-center gap-1">
                                                      Inherit value from last execution
                                                      <RequiredExecutionResultsTooltip />
                                                    </div>
                                                  </label>
                                                </div>
                                              </div>
                                              {inputValue?.valueFrom?.eventData ? (
                                                /* Event Data Mode */
                                                <div>
                                                  <label className="block text-xs font-medium mb-1">Expression</label>
                                                  <input
                                                    value={inputValue.valueFrom.eventData.expression || ''}
                                                    onChange={(e) => inputMappingHandlers.handleEventDataExpressionChange(e.target.value, actualMappingIndex, input.name)}
                                                    placeholder="eg. $.commit[0].message"
                                                    className="w-full px-2 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md text-xs bg-white dark:bg-zinc-700"
                                                  />
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

                                        {/* Display specific error message for this mapping */}
                                        {hasMappingError && (
                                          <div className="mt-2 p-2 bg-red-100 dark:bg-red-800/30 border border-red-200 dark:border-red-700 rounded">
                                            <div className="flex items-center gap-2 text-sm text-red-700 dark:text-red-300">
                                              <MaterialSymbol name="error" size="sm" />
                                              {validationErrors[mappingErrorKey]}
                                            </div>
                                          </div>
                                        )}
                                      </div>
                                    </div>
                                  );
                                })}
                              </div>
                            </ValidationField>
                          </div>
                        </div>
                      }
                    />
                  </div>
                );
              })}
              <button
                onClick={handleInputAdd}
                className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
              >
                <MaterialSymbol name="add" size="sm" />
                Add Input
              </button>
            </EditableAccordionSection>


            {/* Outputs Section */}
            <EditableAccordionSection
              id="outputs"
              title={
                <div className="flex items-center">
                  <span>Outputs</span>
                  <OutputsTooltip className="ml-2" />
                </div>
              }
              isOpen={openSections.includes('outputs')}
              onToggle={handleAccordionToggle}
              isModified={isSectionModified(outputs, 'outputs')}
              onRevert={revertSection}
              count={outputs.length}
              countLabel="outputs"
              hasError={hasErrorsWithPrefix(validationErrors, 'output_')}
            >
              {outputs.map((output, index) => (
                <div key={index}>
                  <InlineEditor
                    isEditing={outputsEditor.editingIndex === index}
                    onSave={() => {
                      const errors = validateAllFields();

                      if (errors[`output_${index}`]) {
                        return;
                      }

                      outputsEditor.saveEdit()
                    }}
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
                          label={
                            <div className="flex items-center">
                              <span>Name</span>
                              <OutputsHelpTooltip className="ml-2" executorType={executor.type} />
                            </div>
                          }
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
              title={
                <div className="flex items-center gap-2">
                  Conditions
                  <ConditionsTooltip />
                </div>
              }
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
              title={
                <div className="flex items-center gap-2">
                  Secrets Management
                  <SecretsTooltip />
                </div>
              }
              isOpen={openSections.includes('secrets')}
              onToggle={handleAccordionToggle}
              isModified={isSectionModified(secrets, 'secrets')}
              onRevert={revertSection}
              count={secrets.length}
              countLabel="secrets"
              hasError={hasErrorsWithPrefix(validationErrors, 'secret_')}
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
              title={
                <div className="flex items-center gap-2">
                  Executor Configuration
                  <ExecutorTooltip />
                </div>
              }
              isOpen={openSections.includes('executor')}
              onToggle={handleAccordionToggle}
              isModified={isSectionModified(executor, 'executor')}
              onRevert={revertSection}
              countLabel="executor"
              className={!openSections.includes('executor') ? 'rounded-b-2xl border-b-0' : ''}
              hasError={
                isExecutorMisconfigured() ||
                hasErrorsWithPrefix(validationErrors, 'executor') ||
                Object.values(fieldErrors || {}).some(Boolean)
              }
            >
              <ExecutorFormSection
                executor={executor}
                dryRun={dryRun}
                availableIntegrations={getAllIntegrations()}
                validationErrors={validationErrors}
                fieldErrors={fieldErrors}
                onExecutorChange={(updates) => {
                  setExecutor(prev => ({ ...prev, ...updates }));
                }}
                onFieldErrorChange={(field, error) => {
                  setFieldErrors(prev => ({ ...prev, [field]: error }));
                }}
                organizationId={organizationId}
                canvasId={canvasId}
              />
            </EditableAccordionSection>
          </>
        )}
      </div>
    </NodeContentWrapper>
  );
}
