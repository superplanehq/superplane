import { useState, useMemo, useCallback, useRef } from 'react';
import { useParams } from 'react-router-dom';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { StageNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { useUpdateStage, useCreateStage, useDeleteStage } from '@/hooks/useCanvasData';
import { SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneValueDefinition, SuperplaneCondition, SuperplaneStage, SuperplaneInputMapping, superplaneListEvents, superplaneCreateEvent } from '@/api-client';
import { useIntegrations } from '../../hooks/useIntegrations';
import { StageEditModeContent } from '../StageEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Badge } from '@/components/Badge/badge';
import { NodeActionButtons } from '@/components/NodeActionButtons';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';

import { formatRelativeTime, formatExecutionDuration } from '../../utils/stageEventUtils';
import { IOTooltip } from './IOTooltip';
import { twMerge } from 'tailwind-merge';
import { StageQueueSection } from '../StageQueueSection';
import { EventTriggerBadge } from '../EventTriggerBadge';
import { createStageDuplicate, focusAndEditNode } from '../../utils/nodeDuplicationUtils';
import { showErrorToast } from '@/utils/toast';
import { EmitEventModal } from '@/components/EmitEventModal/EmitEventModal';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';

const StageImageMap = {
  'http': <MaterialSymbol className='-mt-1 -mb-1' name="rocket_launch" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />
}

export default function StageNode(props: NodeProps<StageNodeType>) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const isNewNode = Boolean(props.data.isDraft) || !!(props.id && /^\d+$/.test(props.id));
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [isHovered, setIsHovered] = useState(false);
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ name: string; description?: string; inputs: SuperplaneInputDefinition[]; outputs: SuperplaneOutputDefinition[]; connections: SuperplaneConnection[]; executor: SuperplaneExecutor; secrets: SuperplaneValueDefinition[]; conditions: SuperplaneCondition[]; inputMappings: SuperplaneInputMapping[]; dryRun: boolean; isValid: boolean } | null>(null);
  const [stageName, setStageName] = useState(props.data.name || '');
  const [stageDescription, setStageDescription] = useState(props.data.description || '');
  const [nameError, setNameError] = useState<string | null>(null);
  const [stageNameDirtyByUser, setStageNameDirtyByUser] = useState(false);
  const [integrationError, setIntegrationError] = useState(false);
  const [showEmitEventModal, setShowEmitEventModal] = useState(false);
  const triggerSectionValidationRef = useRef<(() => void) | null>(null);
  const setFieldErrorsRef = useRef<React.Dispatch<React.SetStateAction<Record<string, string>>> | null>(null);
  const { selectStageId, updateStage, setEditingStage, removeStage, approveStageEvent, addStage, setFocusedNodeId } = useCanvasStore();

  const parseApiErrorMessage = useCallback((errorMessage: string): { field: string; message: string } | null => {
    if (!errorMessage) return null;

    const repositoryNotFoundMatch = errorMessage.match(/repository\s+([^\s]+)\s+not\s+found/i);
    if (repositoryNotFoundMatch) {
      return {
        field: 'repository',
        message: `Repository "${repositoryNotFoundMatch[1]}" not found. Please check that the repository exists and that your Personal Access Token (PAT) has access to it.`
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

    const invalidStatusCodeMatch = errorMessage.match(/invalid\s+status\s+code/i);
    if (invalidStatusCodeMatch) {
      return {
        field: 'statusCodes',
        message: 'Invalid status code'
      };
    }

    return null;
  }, []);

  const handleApiError = useCallback((errorMessage: string) => {
    const parsedError = parseApiErrorMessage(errorMessage);
    if (parsedError && setFieldErrorsRef.current) {
      setFieldErrorsRef.current(prev => ({
        ...prev,
        [parsedError.field]: parsedError.message
      }));
      triggerSectionValidationRef?.current?.();
    }
    showErrorToast(errorMessage);
  }, [parseApiErrorMessage]);

  const allStages = useCanvasStore(state => state.stages);
  const nodes = useCanvasStore(state => state.nodes);
  const currentStage = useCanvasStore(state =>
    state.stages.find(stage => stage.metadata?.id === props.id)
  );

  const isPartiallyBroken = useMemo(() => {
    if (!currentStage || isNewNode)
      return false;

    const hasNoConnections = currentStage.spec?.connections?.length === 0

    const hasInvalidConnections = currentStage.spec?.connections?.some(connection => {
      return !nodes.some(node => node?.data?.name === connection.name)
    })

    const hasInvalidInputMappings = currentStage.spec?.inputMappings?.some(mapping => {
      return !currentStage.spec?.connections?.some(connection => connection.name === mapping.when?.triggeredBy?.connection)
    })

    return hasNoConnections || hasInvalidConnections || hasInvalidInputMappings
  }, [currentStage, isNewNode, nodes])

  const validateStageName = (name: string) => {
    if (!name || name.trim() === '') {
      setNameError('Stage name is required');
      return false;
    }

    const isDuplicate = allStages.some(stage =>
      stage.metadata?.name?.toLowerCase() === name.toLowerCase() &&
      stage.metadata?.id !== props.id
    );

    if (isDuplicate) {
      setNameError('A stage with this name already exists');
      return false;
    }

    setNameError(null);
    return true;
  };

  const validateIntegrationRequirement = () => {
    if (!currentFormData?.executor) {
      return true;
    }

    // Skip integration validation when DryRun mode is enabled
    if (currentFormData.dryRun) {
      return true;
    }

    const executorType = currentFormData.executor.type || '';
    const requireIntegration = ['semaphore', 'github'].includes(executorType);

    if (!requireIntegration) {
      return true;
    }

    const semaphoreIntegrations = availableIntegrations.filter(int => int.spec?.type === 'semaphore');
    const githubIntegrations = availableIntegrations.filter(int => int.spec?.type === 'github');

    const hasRequiredIntegrations = (executorType === 'semaphore' && semaphoreIntegrations.length > 0) ||
      (executorType === 'github' && githubIntegrations.length > 0);

    return hasRequiredIntegrations;
  };

  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const updateStageMutation = useUpdateStage(canvasId);
  const createStageMutation = useCreateStage(canvasId);
  const deleteStageMutation = useDeleteStage(canvasId);
  const focusedNodeId = useCanvasStore(state => state.focusedNodeId);
  const { data: availableIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");

  const pendingEvents = useMemo(() =>
    currentStage?.queue
      ?.filter(event => event.state === 'STATE_PENDING')
      ?.sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [currentStage?.queue]
  );
  const lastPendingEvent = useMemo(() =>
    pendingEvents.at(-1) || null,
    [pendingEvents]
  );

  const waitingEvents = useMemo(() =>
    currentStage?.queue
      ?.filter(event => event.state === 'STATE_WAITING')
      ?.sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [currentStage?.queue]
  );

  const lastWaitingEvent = useMemo(() => {
    const event = waitingEvents.at(-1);
    if (!event) {
      return null;
    }
    return event;
  },
    [waitingEvents]
  );

  const allExecutions = useMemo(() =>
    currentStage?.executions
      ?.sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [currentStage?.executions]
  );

  const allFinishedExecutions = useMemo(() =>
    allExecutions
      .filter(execution => execution?.state === 'STATE_FINISHED')
    , [allExecutions]
  );

  const runningExecution = useMemo(() =>
    allExecutions.find(execution => execution.state === 'STATE_STARTED'),
    [allExecutions]
  );

  // If there is a running execution, use it as the last execution
  const lastExecution = runningExecution || allFinishedExecutions.at(0);
  const lastExecutionEvent = lastExecution?.stageEvent;
  const lastInputsCount = lastExecutionEvent?.inputs?.length || 0;
  const lastOutputsCount = lastExecution?.outputs?.length || 0;

  const getStatusIcon = () => {
    const latestExecution = allExecutions.at(0);
    const status = latestExecution?.state;
    const result = latestExecution?.result;

    switch (status) {
      case 'STATE_STARTED':
        return (
          <MaterialSymbol name="sync" size="lg" className="text-blue-600 mr-2 animate-spin" />
        );
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return <MaterialSymbol name="check_circle" size="lg" className="text-green-600 mr-2" />;
        }
        if (result === 'RESULT_FAILED') {
          return <MaterialSymbol name="cancel" size="lg" className="text-red-600 mr-2" />;
        }
        if (result === 'RESULT_CANCELLED') {
          return <MaterialSymbol name="block" size="lg" className="text-gray-600 dark:text-gray-400 mr-2" />;
        }
        return <MaterialSymbol name="check_circle" size="lg" className="text-green-600 mr-2" />;
      case 'STATE_PENDING':
        return (
          <MaterialSymbol name="hourglass" size="lg" className="text-orange-600 mr-2 animate-spin" />
        );
      default:
        return (
          <span className="material-icons text-gray-600 dark:text-gray-400 text-2xl mr-2">help</span>
        );
    }
  };

  const isRunning = !!runningExecution || props.data.status?.toLowerCase() === 'running';
  const handleEditClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation?.();
    setIsEditMode(true);
    setEditingStage(props.id);

    setStageName(props.data.name);
    setStageDescription(props.data.description || '');
  };

  const handleSaveStage = async () => {
    if (!currentFormData || !currentStage) {
      return;
    }

    let basicValidationPassed = true;

    if (!validateStageName(stageName)) {
      basicValidationPassed = false;
    }

    if (!validateIntegrationRequirement()) {
      // Integration is required but not available, show error message
      setIntegrationError(true);
      const executorType = currentFormData.executor?.type;

      handleApiError(`${executorType} integration is required but not configured. Please add a ${executorType} integration to continue.`);
      return;
    }

    if (!currentFormData.isValid) {
      triggerSectionValidationRef?.current?.();
      basicValidationPassed = false;
    }

    if (!basicValidationPassed) {
      return;
    }

    const isTemporaryId = currentStage.metadata?.id && /^\d+$/.test(currentStage.metadata.id);
    const isNewStage = !currentStage.metadata?.id || currentStage.isDraft || isTemporaryId;

    try {
      if (isNewStage) {
        const createParams = {
          name: stageName,
          description: stageDescription,
          inputs: currentFormData.inputs,
          outputs: currentFormData.outputs,
          connections: currentFormData.connections,
          executor: currentFormData.executor,
          secrets: currentFormData.secrets,
          conditions: currentFormData.conditions,
          inputMappings: currentFormData.inputMappings,
          dryRun: currentFormData.dryRun
        };
        await createStageMutation.mutateAsync(createParams);
        removeStage(props.id);
      } else if (!isNewStage) {

        if (!currentStage.metadata?.id) {
          throw new Error('Stage ID is required for update');
        }

        const updateParams = {
          stageId: currentStage.metadata.id,
          name: stageName,
          description: stageDescription,
          inputs: currentFormData.inputs,
          outputs: currentFormData.outputs,
          connections: currentFormData.connections,
          executor: currentFormData.executor,
          secrets: currentFormData.secrets,
          conditions: currentFormData.conditions,
          inputMappings: currentFormData.inputMappings,
          dryRun: currentFormData.dryRun
        };
        await updateStageMutation.mutateAsync(updateParams);

        updateStage({
          ...currentStage,
          metadata: {
            ...currentStage.metadata,
            name: stageName,
            description: stageDescription
          },
          spec: {
            ...currentStage.spec!,
            inputs: currentFormData.inputs,
            outputs: currentFormData.outputs,
            connections: currentFormData.connections,
            conditions: currentFormData.conditions,
            executor: currentFormData.executor,
            secrets: currentFormData.secrets,
            inputMappings: currentFormData.inputMappings,
            dryRun: currentFormData.dryRun
          }
        });

        props.data.name = stageName;
        props.data.description = stageDescription;
        props.data.dryRun = currentFormData.dryRun;
      }
    } catch (error) {
      const apiError = error as Error;
      console.error(`Failed to ${isNewStage ? 'create' : 'update'} stage:`, apiError);
      console.error('API Error:', apiError);

      handleApiError(apiError.message);

      return;
    }

    setIsEditMode(false);
    setEditingStage(null);
    setCurrentFormData(null);
    setIntegrationError(false);
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingStage(null);
    setCurrentFormData(null);
    setIntegrationError(false);
    setStageName(props.data.name);
    setStageDescription(props.data.description || '');
  };

  const handleDiscardStage = async () => {
    if (currentStage?.metadata?.id) {
      const isTemporaryId = /^\d+$/.test(currentStage.metadata.id);
      const isRealStage = !isTemporaryId && !currentStage.isDraft;

      if (isRealStage) {
        try {
          await deleteStageMutation.mutateAsync(currentStage.metadata.id);
        } catch (error) {
          console.error('Failed to delete stage:', error);
          return;
        }
      }
      removeStage(currentStage.metadata.id);
    }
    setShowDiscardConfirm(false);
  };

  const handleStageNameChange = (newName: string) => {
    setStageName(newName);
    validateStageName(newName);
    handleStageNameUserModified();
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: newName
      });
    }
  };

  const handleStageNameUserModified = () => {
    setStageNameDirtyByUser(true);
  };

  const handleAutoGeneratedStageNameChange = (name: string) => {
    setStageName(name);
  };

  const handleStageDescriptionChange = (newDescription: string) => {
    setStageDescription(newDescription);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        description: newDescription
      });
    }
  };

  const handleDuplicateStage = () => {
    if (!currentStage) return;

    const duplicatedStage = createStageDuplicate(currentStage, allStages);
    addStage(duplicatedStage, true, true);

    focusAndEditNode(
      duplicatedStage.metadata?.id || '',
      setFocusedNodeId,
      setEditingStage
    );
  };

  const handleYamlApply = (updatedData: unknown) => {
    const yamlData = updatedData as SuperplaneStage;

    if (yamlData.metadata?.name) {
      setStageName(yamlData.metadata.name);
    }
    if (yamlData.metadata?.description) {
      setStageDescription(yamlData.metadata.description);
    }

    if (yamlData.spec && currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: yamlData.metadata?.name || stageName,
        description: yamlData.metadata?.description || stageDescription,
        inputs: yamlData.spec.inputs || currentFormData.inputs,
        outputs: yamlData.spec.outputs || currentFormData.outputs,
        connections: yamlData.spec.connections || currentFormData.connections,
        executor: yamlData.spec.executor || currentFormData.executor,
        secrets: yamlData.spec.secrets || currentFormData.secrets,
        conditions: yamlData.spec.conditions || currentFormData.conditions,
        inputMappings: yamlData.spec.inputMappings || currentFormData.inputMappings,
        isValid: currentFormData.isValid
      });
    }
  };


  const handleDataChange = useCallback((data: typeof currentFormData) => {
    setCurrentFormData(data);
  }, []);

  const handleFieldErrorsChange = useCallback((setFieldErrorsFunction: React.Dispatch<React.SetStateAction<Record<string, string>>>) => {
    setFieldErrorsRef.current = setFieldErrorsFunction;
  }, []);

  const getBackgroundColorClass = () => {
    const latestExecution = allExecutions.at(0);
    const status = latestExecution?.state;
    const result = latestExecution?.result;

    switch (status) {
      case 'STATE_STARTED':
        return 'bg-blue-50 dark:bg-blue-900/50 border-blue-200 dark:border-blue-700';
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return 'bg-green-50 dark:bg-green-900/50 border-green-200 dark:border-green-700';
        }
        if (result === 'RESULT_FAILED') {
          return 'bg-red-50 dark:bg-red-900/50 border-red-200 dark:border-red-700';
        }
        if (result === 'RESULT_CANCELLED') {
          return 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700';
        }
        return 'bg-green-50 dark:bg-green-900/50 border-green-200 dark:border-green-700';
      case 'STATE_PENDING':
        return 'bg-yellow-50 dark:bg-yellow-900/50 border-yellow-200 dark:border-yellow-700';
      default:
        return 'bg-gray-50 dark:bg-gray-800 border-gray-200 dark:border-gray-700';
    }
  };

  const eventsMoreCount = useMemo(() => {
    let total = (pendingEvents?.length || 0) + (waitingEvents?.length || 0)

    if (lastWaitingEvent && lastWaitingEvent.stateReason === "STATE_REASON_APPROVAL")
      total -= 1

    if (lastWaitingEvent && lastWaitingEvent.stateReason === "STATE_REASON_TIME_WINDOW")
      total -= 1

    if (lastPendingEvent)
      total -= 1

    return total
  }, [lastPendingEvent, lastWaitingEvent, pendingEvents?.length, waitingEvents?.length])

  const executorBadges = useMemo(() => {
    const badges: Array<{ icon: string; text: string }> = []

    if (props.data.executor?.type === 'semaphore') {
      const resourceName = (props.data.executor?.resource?.name as string)?.replace('.semaphore/', '')
      const pipelineFile = (props.data.executor?.spec?.['pipelineFile'] as string)?.replace('.semaphore/', '')
      const ref = props.data.executor?.spec?.['ref'] as string

      if (resourceName) badges.push({ icon: 'assignment', text: resourceName })
      if (pipelineFile) badges.push({ icon: 'code', text: pipelineFile })
      if (ref) badges.push({ icon: 'graph_1', text: ref })
    }

    if (props.data.executor?.type === 'github') {
      const resourceName = props.data.executor?.resource?.name as string
      const workflow = (props.data.executor?.spec?.['workflow'] as string)?.replace('.github/workflows/', '')
      const ref = props.data.executor?.spec?.['ref'] as string

      if (resourceName) badges.push({ icon: 'assignment', text: resourceName })
      if (workflow) badges.push({ icon: 'code', text: workflow })
      if (ref) badges.push({ icon: 'graph_1', text: ref })
    }

    return badges
  }, [props.data.executor])


  const borderColor = useMemo(() => {
    if (isPartiallyBroken) {
      return 'border-red-400 dark:border-red-200'
    }

    if (props.selected || focusedNodeId === props.id) {
      return 'border-blue-400 dark:border-gray-200'
    }
    return 'border-transparent dark:border-transparent'
  }, [props.selected, focusedNodeId, props.id, isPartiallyBroken])

  return (
    <div
      className="relative pt-14"
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div
        className={twMerge(`bg-transparent rounded-xl border-2 relative `, borderColor)}
        onClick={() => {
          if (!isEditMode && currentStage?.metadata?.id) {
            selectStageId(props.id);
          }
        }}
      >
        <div className="bg-white dark:bg-zinc-800 border-gray-200 dark:border-gray-700 rounded-xl"
          style={{ width: isEditMode ? '390px' : '320px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
        >

          {(isHovered || isEditMode) && (
            <NodeActionButtons
              isNewNode={!!isNewNode}
              onSave={handleSaveStage}
              onCancel={handleCancelEdit}
              onDiscard={() => setShowDiscardConfirm(true)}
              onEdit={() => handleEditClick({} as React.MouseEvent<HTMLButtonElement>)}
              onDuplicate={!isNewNode ? handleDuplicateStage : undefined}
              onSend={currentStage?.metadata?.id ? () => setShowEmitEventModal(true) : undefined}
              isEditMode={isEditMode}
              entityType="stage"
              entityData={currentFormData ? {
                metadata: {
                  name: stageName,
                  description: stageDescription
                },
                spec: {
                  inputs: currentFormData.inputs,
                  outputs: currentFormData.outputs,
                  connections: currentFormData.connections,
                  executor: currentFormData.executor,
                  secrets: currentFormData.secrets,
                  conditions: currentFormData.conditions,
                  inputMappings: currentFormData.inputMappings
                }
              } : (currentStage ? {
                metadata: {
                  name: currentStage.metadata?.name,
                  description: currentStage.metadata?.description
                },
                spec: {
                  inputs: currentStage.spec?.inputs || [],
                  outputs: currentStage.spec?.outputs || [],
                  connections: currentStage.spec?.connections || [],
                  executor: currentStage.spec?.executor || { type: '', spec: {} },
                  secrets: currentStage.spec?.secrets || [],
                  conditions: currentStage.spec?.conditions || [],
                  inputMappings: currentStage.spec?.inputMappings || []
                }
              } : null)}
              onYamlApply={handleYamlApply}
            />
          )}


          {/* Header Section */}
          <div className={twMerge('px-4 py-4 justify-between items-start border-gray-200 dark:border-gray-700', isEditMode ? 'border-b' : '')}>
            <div className="flex items-start justify-between w-full">
              <div className="flex items-start flex-1 min-w-0">
                <div className='max-w-8 mt-1 flex items-center justify-center'>
                  {StageImageMap[(props.data.executor?.type || 'http') as keyof typeof StageImageMap]}
                </div>
                <div className="flex-1 min-w-0 ml-2">
                  <div className="mb-1">
                    <InlineEditable
                      value={stageName}
                      onSave={handleStageNameChange}
                      placeholder="Stage name"
                      className={twMerge(`font-bold text-gray-900 dark:text-gray-100 text-base text-left px-2 py-1`,
                        nameError && isEditMode ? 'border border-red-500 rounded-lg' : '',
                        isEditMode ? 'text-sm' : '')}
                      isEditMode={isEditMode}
                      autoFocus={!!isNewNode && !props.data.executor?.resource?.type}
                      dataTestId="event-source-name-input"
                    />
                    {nameError && isEditMode && (
                      <div className="text-xs text-red-600 text-left mt-1 px-2">
                        {nameError}
                      </div>
                    )}
                  </div>
                  <div>
                    {isEditMode && <InlineEditable
                      value={stageDescription}
                      onSave={handleStageDescriptionChange}
                      placeholder={isEditMode ? "Add description..." : ""}
                      className="text-gray-600 dark:text-gray-400 text-sm text-left px-2 py-1"
                      isEditMode={isEditMode}
                    />}
                  </div>
                </div>
              </div>
            </div>
            {!isEditMode && (
              <div className="text-xs text-left text-gray-600 dark:text-gray-400 w-full mt-1">{stageDescription || ''}</div>
            )}
          </div>

          {isEditMode ? (
            <StageEditModeContent
              data={{
                ...props.data,
                name: stageName,
                description: stageDescription,
                ...(currentFormData && {
                  inputs: currentFormData.inputs,
                  outputs: currentFormData.outputs,
                  connections: currentFormData.connections,
                  executor: currentFormData.executor,
                  secrets: currentFormData.secrets,
                  conditions: currentFormData.conditions,
                  inputMappings: currentFormData.inputMappings,
                  dryRun: currentFormData.dryRun
                })
              }}
              currentStageId={props.id}
              canvasId={canvasId}
              organizationId={organizationId!}
              isNewStage={isNewNode}
              dirtyByUser={stageNameDirtyByUser}
              onDataChange={handleDataChange}
              onTriggerSectionValidation={triggerSectionValidationRef}
              onStageNameChange={handleAutoGeneratedStageNameChange}
              integrationError={integrationError}
              onFieldErrorsChange={handleFieldErrorsChange}
            />
          ) : (
            <>

              {(executorBadges.length > 0 || props.data.dryRun) && (
                <div className="flex flex-col w-full gap-2 px-4 font-semibold min-w-0 overflow-hidden">
                  {props.data.dryRun && (
                    <div className="flex items-center">
                      <Badge
                        color="yellow"
                        icon="science"
                        className="flex-shrink-0"
                        title="This stage is in dry run mode and will use the no-op executor"
                      >
                        Dry Run Mode
                      </Badge>
                    </div>
                  )}
                  {executorBadges.length > 0 && !props.data.dryRun && (
                    <div className="flex items-center gap-2 min-w-0 overflow-hidden">
                      {executorBadges.map((badge, index) => (
                        <Badge
                          key={`${badge.icon}-${index}`}
                          color="zinc"
                          icon={badge.icon}
                          truncate
                          className="flex-shrink min-w-0 max-w-full"
                          title={badge.text}
                        >
                          {badge.text}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
              )}
              {/* Last Run Section */}
              <div className={`mt-4 px-3 py-3 border-t-2 w-full ${getBackgroundColorClass()}`}>
                <div className="flex items-center w-full justify-between mb-2">
                  <div className="text-xs font-bold text-gray-900 dark:text-gray-100 uppercase tracking-wide">Last run</div>
                  <div className="text-xs text-gray-600 dark:text-gray-400">
                    {isRunning ? 'Running...' : lastExecution ? (
                      <div className="flex items-center gap-1 font-semibold text-gray-500 dark:text-gray-400">
                        <MaterialSymbol name="timer" size="md" />
                        <span>{formatExecutionDuration(lastExecution?.createdAt, lastExecution?.finishedAt)}</span>
                        <span>|</span>
                        <span>{formatRelativeTime(lastExecution?.finishedAt, true)}</span>
                      </div>
                    ) : 'No recent runs'}
                  </div>
                </div>

                {/* Current Execution Display */}
                <div>
                  <div className="flex items-center mb-1 py-2">
                    {getStatusIcon()}
                    <span
                      className="text-left w-full font-semibold text-sm truncate text-gray-900 dark:text-gray-100"
                    >
                      {lastExecutionEvent?.name || lastExecution?.id || 'No recent runs'}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 font-semibold">
                    {lastInputsCount > 0 && (
                      <IOTooltip
                        type="inputs"
                        data={lastExecutionEvent?.inputs?.map(input => ({ name: input.name, value: input.value })) || []}
                      >
                        <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                          <MaterialSymbol name="input" size="md" />
                          <span className="whitespace-nowrap">{lastInputsCount} {lastInputsCount === 1 ? 'input' : 'inputs'}</span>
                        </span>
                      </IOTooltip>
                    )}
                    <EventTriggerBadge
                      lastExecutionEvent={lastExecutionEvent}
                      lastExecution={lastExecution}
                      stageName={props.data.name}
                    />
                    {lastOutputsCount > 0 && (
                      <IOTooltip
                        type="outputs"
                        data={lastExecution?.outputs?.map(output => ({ name: output.name, value: output.value })) || []}
                      >
                        <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                          <MaterialSymbol name="output" size="md" />
                          <span className="whitespace-nowrap">{lastOutputsCount} {lastOutputsCount === 1 ? 'output' : 'outputs'}</span>
                        </span>
                      </IOTooltip>
                    )}

                  </div>
                </div>
              </div>
              {/* Queue Section */}
              <StageQueueSection
                lastWaitingEvent={lastWaitingEvent}
                lastPendingEvent={lastPendingEvent}
                eventsMoreCount={eventsMoreCount}
                onApproveEvent={approveStageEvent}
                stageId={currentStage?.metadata?.id || ''}
              />
            </>
          )}

          {/* Custom Handles */}
          <CustomBarHandle internalPadding={true} type="target" connections={props.data.connections} conditions={props.data.conditions} />
          <CustomBarHandle internalPadding={true} type="source" />

          <ConfirmDialog
            isOpen={showDiscardConfirm}
            title="Delete Stage"
            message="Are you sure you want to delete this stage? This action cannot be undone."
            confirmText="Delete"
            cancelText="Cancel"
            confirmVariant="danger"
            onConfirm={handleDiscardStage}
            onCancel={() => setShowDiscardConfirm(false)}
          />

          {currentStage?.metadata?.id && (
            <EmitEventModal
              nodeType="stage"
              isOpen={showEmitEventModal}
              onClose={() => setShowEmitEventModal(false)}
              sourceName={currentStage.metadata.name || ''}
              loadLastEvent={async () => {
                try {
                  const response = await superplaneListEvents(withOrganizationHeader({
                    path: { canvasIdOrName: canvasId! },
                    query: {
                      sourceType: 'EVENT_SOURCE_TYPE_STAGE' as const,
                      sourceId: currentStage.metadata!.id,
                      limit: 1
                    }
                  }));
                  return response.data?.events?.[0] || null;
                } catch (error) {
                  console.error('Failed to load last event for stage:', error);
                  return null;
                }
              }}
              onSubmit={async (eventType: string, eventData: any) => {
                await superplaneCreateEvent(withOrganizationHeader({
                  path: { canvasIdOrName: canvasId! },
                  body: {
                    sourceType: 'EVENT_SOURCE_TYPE_STAGE',
                    sourceId: currentStage.metadata!.id,
                    type: eventType,
                    raw: eventData
                  }
                }));
              }}
            />
          )}
        </div>
      </div>
    </div>
  );
}
