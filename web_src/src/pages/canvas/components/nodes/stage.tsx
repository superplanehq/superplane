import { useState, useMemo, useCallback } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { StageNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import type { StageWithEventQueue } from '../../store/types';
import { useUpdateStage, useCreateStage } from '@/hooks/useCanvasData';
import { SuperplaneExecution, SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneValueDefinition, SuperplaneCondition, SuperplaneStage, SuperplaneInputMapping } from '@/api-client';
import { StageEditModeContent } from '../StageEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { EditModeActionButtons } from '../EditModeActionButtons';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';

import { formatRelativeTime } from '../../utils/stageEventUtils';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import { twMerge } from 'tailwind-merge';

const StageImageMap = {
  'webhook': <MaterialSymbol className='-mt-1 -mb-1' name="webhook" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />
}

export default function StageNode(props: NodeProps<StageNodeType>) {
  const isNewNode = Boolean(props.data.isDraft) || (props.id && /^\d+$/.test(props.id));
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ label: string; description?: string; inputs: SuperplaneInputDefinition[]; outputs: SuperplaneOutputDefinition[]; connections: SuperplaneConnection[]; executor: SuperplaneExecutor; secrets: SuperplaneValueDefinition[]; conditions: SuperplaneCondition[]; inputMappings: SuperplaneInputMapping[]; isValid: boolean } | null>(null);
  const [stageName, setStageName] = useState(props.data.label || '');
  const [stageDescription, setStageDescription] = useState(props.data.description || '');
  const [nameError, setNameError] = useState<string | null>(null);
  const { selectStageId, updateStage, setEditingStage, removeStage } = useCanvasStore();
  const allStages = useCanvasStore(state => state.stages);
  const currentStage = useCanvasStore(state =>
    state.stages.find(stage => stage.metadata?.id === props.id)
  );

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
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const updateStageMutation = useUpdateStage(canvasId);
  const createStageMutation = useCreateStage(canvasId);
  const focusedNodeId = useCanvasStore(state => state.focusedNodeId);

  const pendingEvents = useMemo(() =>
    currentStage?.queue
      ?.filter(event => event.state === 'STATE_PENDING')
      ?.sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [currentStage?.queue]
  );
  const lastPendingEvent = useMemo(() =>
    pendingEvents.at(-1),
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
    if (!event || event.stateReason !== 'STATE_REASON_APPROVAL') {
      return null;
    }
    return event;
  },
    [waitingEvents]
  );

  const allExecutions = useMemo(() =>
    currentStage?.queue?.flatMap(event => event.execution as SuperplaneExecution)
      .filter(execution => execution)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [currentStage?.queue]
  );

  const allFinishedExecutions = useMemo(() =>
    allExecutions
      .filter(execution => execution?.finishedAt)
    , [allExecutions]
  );

  const executionRunning = useMemo(() =>
    allExecutions.some(execution => execution.state === 'STATE_STARTED'),
    [allExecutions]
  );

  const lastFinishedExecution = allFinishedExecutions.at(0);
  const lastExecutionEvent = currentStage?.queue?.find(event => event.execution?.id === lastFinishedExecution?.id);
  const lastInputsCount = lastExecutionEvent?.inputs?.length || 0;
  const lastOutputsCount = lastFinishedExecution?.outputs?.length || 0;

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
        return <span className="material-icons text-green-600 text-2xl mr-2">check_circle</span>;
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

  const isRunning = executionRunning || props.data.status?.toLowerCase() === 'running';
  const handleEditClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation?.();
    setIsEditMode(true);
    setEditingStage(props.id);

    setStageName(props.data.label);
    setStageDescription(props.data.description || '');
  };

  const handleSaveStage = async (saveAsDraft = false) => {
    if (!currentFormData || !currentStage) {
      return;
    }

    if (!validateStageName(stageName)) {
      return;
    }

    if (!currentFormData.isValid && !saveAsDraft) {
      return;
    }

    const isTemporaryId = currentStage.metadata?.id && /^\d+$/.test(currentStage.metadata.id);
    const isNewStage = !currentStage.metadata?.id || currentStage.isDraft || isTemporaryId;

    try {
      if (isNewStage && !saveAsDraft) {
        const createParams = {
          name: stageName,
          description: stageDescription,
          inputs: currentFormData.inputs,
          outputs: currentFormData.outputs,
          connections: currentFormData.connections,
          executor: currentFormData.executor,
          secrets: currentFormData.secrets,
          conditions: currentFormData.conditions,
          inputMappings: currentFormData.inputMappings
        };
        await createStageMutation.mutateAsync(createParams);
        removeStage(props.id);
      } else if (!isNewStage && !saveAsDraft) {

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
          inputMappings: currentFormData.inputMappings
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
            executor: currentFormData.executor,
            secrets: currentFormData.secrets,
            inputMappings: currentFormData.inputMappings
          }
        });

        props.data.label = stageName;
        props.data.description = stageDescription;
      } else if (saveAsDraft) {

        const draftStage: StageWithEventQueue = {
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
            executor: currentFormData.executor,
            inputMappings: currentFormData.inputMappings
          },
          isDraft: true
        };
        updateStage(draftStage);

        props.data.label = stageName;
        props.data.description = stageDescription;
      }
    } catch (error) {
      const apiError = error as Error;
      console.error(`Failed to ${isNewStage ? 'create' : 'update'} stage:`, apiError);

      console.error('API Error:', apiError);

      // Call the API error handler if available
      const handleStageApiError = (window as { handleStageApiError?: (errorMessage: string) => void }).handleStageApiError;
      if (handleStageApiError) {
        handleStageApiError(apiError.message);
      }

      return;
    }

    setIsEditMode(false);
    setEditingStage(null);
    setCurrentFormData(null);
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingStage(null);
    setCurrentFormData(null);
    setStageName(props.data.label);
    setStageDescription(props.data.description || '');
  };

  const handleDiscardStage = () => {
    if (currentStage?.metadata?.id) {
      removeStage(currentStage.metadata.id);
    }
    setShowDiscardConfirm(false);
  };

  const handleStageNameChange = (newName: string) => {
    setStageName(newName);
    validateStageName(newName);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        label: newName
      });
    }
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
        label: yamlData.metadata?.name || stageName,
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

    if (lastPendingEvent)
      total -= 1

    return total
  }, [lastPendingEvent, lastWaitingEvent, pendingEvents?.length, waitingEvents?.length])

  return (
    <div
      onClick={!isEditMode ? () => selectStageId(props.id) : undefined}
      className={`p-[2px] bg-transparent rounded-xl border-2 ${props.selected ? 'border-blue-400 dark:border-gray-200' : 'border-transparent dark:border-transparent'} relative `}
    >
      <div className="bg-white dark:bg-zinc-800 border border-gray-200 dark:border-gray-700 rounded-xl"
        style={{ width: isEditMode ? '390px' : '320px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
      >

        {focusedNodeId === props.id && (
          <EditModeActionButtons
            isNewNode={!!isNewNode}
            onSave={handleSaveStage}
            onCancel={handleCancelEdit}
            onDiscard={() => setShowDiscardConfirm(true)}
            onEdit={() => handleEditClick({} as React.MouseEvent<HTMLButtonElement>)}
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
        <div className="mt-1 px-4 py-4 justify-between items-start border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-start flex-1 min-w-0">
            <div className='max-w-8 mt-1 flex items-center justify-center'>
              {StageImageMap[(props.data.executor?.type || 'http') as keyof typeof StageImageMap]}
            </div>
            <div className="flex-1 min-w-0 ml-2">
              <div className="mb-1">
                <InlineEditable
                  value={stageName}
                  onSave={handleStageNameChange}
                  placeholder="Event source name"
                  className={twMerge(`font-bold text-gray-900 dark:text-gray-100 text-base text-left px-2 py-1`,
                    nameError && isEditMode ? 'border border-red-500 rounded-lg' : '',
                    isEditMode ? 'text-sm' : '')}
                  isEditMode={isEditMode}
                  autoFocus={!!isNewNode}
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
                  placeholder={isEditMode ? "Add description..." : "No description available"}
                  className="text-gray-600 dark:text-gray-400 text-sm text-left px-2 py-1"
                  isEditMode={isEditMode}
                />}
              </div>
            </div>
          </div>
          {!isEditMode && (
            <div className="text-xs text-left text-gray-600 dark:text-gray-400 w-full mt-1">{stageDescription || 'No description available'}</div>
          )}
        </div>

        {isEditMode ? (
          <StageEditModeContent
            data={{
              ...props.data,
              label: stageName,
              description: stageDescription,
              ...(currentFormData && {
                inputs: currentFormData.inputs,
                outputs: currentFormData.outputs,
                connections: currentFormData.connections,
                executor: currentFormData.executor,
                secrets: currentFormData.secrets,
                conditions: currentFormData.conditions,
                inputMappings: currentFormData.inputMappings
              })
            }}
            currentStageId={props.id}
            onDataChange={handleDataChange}
          />
        ) : (
          <>

            {props.data.executor?.type === 'semaphore' && (
              <div className="flex items-center w-full gap-2 mx-4 font-semibold">
                <div className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                  <MaterialSymbol name="assignment" size="md" />
                  <span>{(props.data.executor?.resource?.name as string)?.replace('.semaphore/', '')}</span>
                </div>
                <div className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">
                  <MaterialSymbol name="code" size="md" />
                  <span>{(props.data.executor?.spec?.['pipelineFile'] as string)?.replace('.semaphore/', '')}</span>
                </div>
              </div>
            )}
            {/* Last Run Section */}
            <div className={`mt-4 px-3 py-3 border-t-2 w-full ${getBackgroundColorClass()}`}>
              <div className="flex items-center w-full justify-between mb-2">
                <div className="text-xs font-bold text-gray-900 dark:text-gray-100 uppercase tracking-wide">Last run</div>
                <div className="text-xs text-gray-600 dark:text-gray-400">
                  {isRunning ? 'Running...' : lastFinishedExecution ? formatRelativeTime(lastFinishedExecution?.finishedAt) : 'No recent runs'}
                </div>
              </div>

              {/* Current Execution Display */}
              <div>
                <div className="flex items-center mb-1">
                  {getStatusIcon()}
                  <a
                    href="#"
                    className="min-w-0 font-semibold text-sm flex items-center hover:underline truncate text-gray-900 dark:text-gray-100"
                    onClick={() => selectStageId(props.id)}
                  >
                    {props.data.label || 'Stage execution'}
                  </a>
                </div>
                <div className="flex items-center gap-2 font-semibold">
                  {lastInputsCount > 0 && <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">{lastInputsCount} {lastInputsCount === 1 ? 'input' : 'inputs'}</span>}
                  {lastOutputsCount > 0 && <span className="inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline bg-zinc-600/10 text-zinc-700 group-data-hover:bg-zinc-600/20 dark:bg-white/5 dark:text-zinc-400 dark:group-data-hover:bg-white/10">{lastOutputsCount} {lastOutputsCount === 1 ? 'output' : 'outputs'}</span>}
                </div>
              </div>
            </div>

            {/* Queue Section */}
            <div className="px-3 pt-2 pb-2 w-full">
              <div className="w-full text-left flex justify-between text-xs font-bold text-gray-900 dark:text-gray-100 uppercase tracking-wide mb-1">
                Next in queue
                {
                  eventsMoreCount > 0 && (
                    <span className="text-xs text-gray-400 font-medium">+{eventsMoreCount} more</span>
                  )
                }
              </div>

              {
                lastWaitingEvent && lastWaitingEvent.stateReason === "STATE_REASON_APPROVAL" && (
                  <div className="flex justify-between w-full px-2 py-3 border-1 rounded border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 mb-2">
                    <span className='font-semibold text-gray-900 dark:text-gray-100 text-sm truncate mt-[2px]'>{lastWaitingEvent?.id}</span>
                    <Tippy content="Manual approval required" placement="top">
                      <MaterialSymbol name="how_to_reg" size="md" className='text-orange-700' />
                    </Tippy>
                  </div>
                )
              }

              {
                lastPendingEvent && (
                  <div className="flex justify-between w-full px-2 py-3 border-1 rounded border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                    <span className='font-semibold text-gray-900 dark:text-gray-100 text-sm truncate mt-[2px]'>{lastPendingEvent?.id}</span>
                    <Tippy content="Waiting For the current execution to finish" placement="top">
                      <MaterialSymbol name="timer" size="md" className='text-orange-700' />
                    </Tippy>
                  </div>
                )
              }

              {
                !lastPendingEvent && !lastWaitingEvent && (
                  <div className="flex justify-between w-full mb-2 px-2 py-3 border-1 rounded border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                    <span className='font-semibold text-gray-500 dark:text-gray-400 text-sm truncate mt-[2px]'>No events in queue..</span>
                  </div>
                )
              }

            </div>
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
      </div>

    </div>
  );
};

