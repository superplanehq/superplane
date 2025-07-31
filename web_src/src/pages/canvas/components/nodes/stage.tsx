import { useState, useMemo } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { StageNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import type { StageWithEventQueue } from '../../store/types';
import { useUpdateStage, useCreateStage } from '@/hooks/useCanvasData';
import { SuperplaneExecution, SuperplaneInputDefinition, SuperplaneOutputDefinition, SuperplaneConnection, SuperplaneExecutor, SuperplaneValueDefinition, SuperplaneCondition, SuperplaneStage } from '@/api-client';
import { EditModeContent } from '../EditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { EditModeActionButtons } from '../EditModeActionButtons';

export default function StageNode(props: NodeProps<StageNodeType>) {
  const isNewNode = Boolean(props.data.isDraft) || (props.id && /^\d+$/.test(props.id));
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ label: string; description?: string; inputs: SuperplaneInputDefinition[]; outputs: SuperplaneOutputDefinition[]; connections: SuperplaneConnection[]; executor: SuperplaneExecutor; secrets: SuperplaneValueDefinition[]; conditions: SuperplaneCondition[]; isValid: boolean } | null>(null);
  const [apiError, setApiError] = useState<string | null>(null);
  const [stageName, setStageName] = useState(props.data.label);
  const [stageDescription, setStageDescription] = useState(props.data.description || '');
  const { selectStageId, updateStage, setEditingStage, removeStage } = useCanvasStore()
  const currentStage = useCanvasStore(state =>
    state.stages.find(stage => stage.metadata?.id === props.id)
  )
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const updateStageMutation = useUpdateStage(canvasId);
  const createStageMutation = useCreateStage(canvasId);
  const pendingEvents = useMemo(() =>
    props.data.queues?.filter(event => event.state === 'STATE_PENDING') || [],
    [props.data.queues]
  );
  const waitingEvents = useMemo(() =>
    props.data.queues?.filter(event => event.state === 'STATE_WAITING') || [],
    [props.data.queues]
  );
  const allExecutions = useMemo(() =>
    props.data.queues?.flatMap(event => event.execution as SuperplaneExecution)
      .filter(execution => execution)
      .sort((a, b) => new Date(b?.createdAt || '').getTime() - new Date(a?.createdAt || '').getTime()) || [],
    [props.data.queues]
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

  const outputs = useMemo(() => {
    const lastFinishedExecution = allFinishedExecutions.at(0);

    return props.data.outputs.map(output => {
      const executionOutput = lastFinishedExecution?.outputs?.find(
        executionOutput => executionOutput.name === output.name
      )
      return {
        key: output.name,
        value: executionOutput?.value || '—',
        required: !!output.required
      }
    })
  }, [props.data.outputs, allFinishedExecutions])

  const getStatusIcon = () => {
    const latestExecution = allExecutions.at(0);
    const status = latestExecution?.state;
    const result = latestExecution?.result;

    switch (status) {
      case 'STATE_STARTED':
        return (
          <span className="rounded-full bg-blue-500 w-[22px] h-[22px] border border-blue-200 text-center mr-2 flex items-center justify-center">
            <span className="text-white text-base job-log-working"></span>
          </span>
        );
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return <span className="material-icons text-green-600 text-2xl mr-2">check_circle</span>;
        }
        if (result === 'RESULT_FAILED') {
          return <span className="material-icons text-red-600 text-2xl mr-2">cancel</span>;
        }
        return <span className="material-icons text-green-600 text-2xl mr-2">check_circle</span>;
      case 'STATE_PENDING':
        return (
          <span className="rounded-full bg-orange-500 w-[22px] h-[22px] border border-orange-200 text-center mr-2 flex items-center justify-center">
            <span className="text-white text-xs job-log-pending"></span>
          </span>
        );
      default:
        return (
          <span className="material-icons text-gray-600 text-2xl mr-2">help</span>
        );
    }
  };

  const isRunning = executionRunning || props.data.status?.toLowerCase() === 'running';
  const handleEditClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    setIsEditMode(true);
    setEditingStage(props.id);

    setStageName(props.data.label);
    setStageDescription(props.data.description || '');
  };

  const handleSaveStage = async (saveAsDraft = false) => {
    if (!currentFormData || !currentStage) {
      return;
    }

    if (!currentFormData.isValid && !saveAsDraft) {
      setApiError('Please fix validation errors before saving.');
      return;
    }

    setApiError(null);

    const isTemporaryId = currentStage.metadata?.id && /^\d+$/.test(currentStage.metadata.id);
    const isNewStage = !currentStage.metadata?.id || currentStage.isDraft || isTemporaryId;

    try {
      if (isNewStage && !saveAsDraft) {
        await createStageMutation.mutateAsync({
          name: stageName,
          description: stageDescription,
          inputs: currentFormData.inputs,
          outputs: currentFormData.outputs,
          connections: currentFormData.connections,
          executor: currentFormData.executor,
          secrets: currentFormData.secrets,
          conditions: currentFormData.conditions
        });
        removeStage(props.id);
      } else if (!isNewStage && !saveAsDraft) {

        if (!currentStage.metadata?.id) {
          throw new Error('Stage ID is required for update');
        }

        await updateStageMutation.mutateAsync({
          stageId: currentStage.metadata.id,
          name: stageName,
          description: stageDescription,
          inputs: currentFormData.inputs,
          outputs: currentFormData.outputs,
          connections: currentFormData.connections,
          executor: currentFormData.executor,
          secrets: currentFormData.secrets,
          conditions: currentFormData.conditions
        });

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
            secrets: currentFormData.secrets
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
            executor: currentFormData.executor
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

      const errorMessage = apiError.message || 'An error occurred while saving the stage';
      setApiError(errorMessage);
      return;
    }

    setIsEditMode(false);
    setEditingStage(null);
    setCurrentFormData(null);
    setApiError(null);
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingStage(null);
    setCurrentFormData(null);
    setApiError(null);

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
        isValid: currentFormData.isValid
      });
    }
  };


  const getBackgroundColorClass = () => {
    const latestExecution = allExecutions.at(0);
    const status = latestExecution?.state;
    const result = latestExecution?.result;

    switch (status) {
      case 'STATE_STARTED':
        return 'bg-blue-50 border-blue-200';
      case 'STATE_FINISHED':
        if (result === 'RESULT_PASSED') {
          return 'bg-green-50 border-green-200';
        }
        if (result === 'RESULT_FAILED') {
          return 'bg-red-50 border-red-200';
        }
        return 'bg-green-50 border-green-200';
      case 'STATE_PENDING':
        return 'bg-yellow-50 border-yellow-200';
      default:
        return 'bg-gray-50 border-gray-200';
    }
  };

  return (
    <div
      onClick={!isEditMode ? () => selectStageId(props.id) : undefined}
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative `}
      style={{ width: '390px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {isEditMode && (
        <EditModeActionButtons
          onSave={handleSaveStage}
          onCancel={handleCancelEdit}
          onDiscard={() => setShowDiscardConfirm(true)}
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
              conditions: currentFormData.conditions
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
              conditions: currentStage.spec?.conditions || []
            }
          } : null)}
          onYamlApply={handleYamlApply}
        />
      )}

      {/* Header Section */}
      <div className="px-4 py-4 flex justify-between items-start">
        <div className="flex items-start flex-1 min-w-0">
          <span className="material-symbols-outlined mr-2 text-gray-700 mt-1">rocket_launch</span>
          <div className="flex-1 min-w-0">
            <div className="mb-1">
              <InlineEditable
                value={stageName}
                onSave={handleStageNameChange}
                placeholder="Stage name"
                className="font-bold text-gray-900 text-base text-left px-2 py-1"
                isEditMode={isEditMode}
              />
            </div>
            <div>
              <InlineEditable
                value={stageDescription}
                onSave={handleStageDescriptionChange}
                placeholder={isEditMode ? "Add description..." : "No description available"}
                className="text-gray-600 text-sm text-left px-2 py-1"
                isEditMode={isEditMode}
              />
            </div>
            {/* API Error Display */}
            {isEditMode && apiError && (
              <p className="text-left text-sm text-red-700 mt-1">{apiError}</p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 ml-2">
          {!isEditMode && (
            <button
              onClick={handleEditClick}
              className="p-1 text-gray-500 hover:text-gray-700 transition-colors"
              title="Edit stage"
            >
              <MaterialSymbol name="edit" size="md" />
            </button>
          )}
          {props.data.isDraft && (
            <span className="text-black bg-gray-200 px-2 py-1 rounded-md text-xs">Draft</span>
          )}
        </div>
      </div>

      {isEditMode ? (
        <EditModeContent
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
              conditions: currentFormData.conditions
            })
          }}
          currentStageId={props.id}
          onDataChange={setCurrentFormData}
        />
      ) : (
        <>
          {/* Last Run Section */}
          <div className={`px-3 py-3 border-t w-full ${getBackgroundColorClass()}`}>
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-xs font-medium text-gray-700 uppercase tracking-wide">Last run</div>
              <div className="text-xs text-gray-600">
                {isRunning ? 'Running...' : props.data.timestamp || 'No recent runs'}
              </div>
            </div>

            {/* Current Execution Display */}
            <div>
              <div className="flex items-center mb-1">
                {getStatusIcon()}
                <a
                  href="#"
                  className="min-w-0 font-semibold text-sm flex items-center hover:underline truncate text-gray-900"
                  onClick={() => selectStageId(props.id)}
                >
                  {props.data.label || 'Stage execution'}
                </a>
              </div>

              {/* Output Tags */}
              <div className="flex flex-wrap gap-1 mt-2">
                {outputs.slice(0, 4).map((output, index) => (
                  <span
                    key={index}
                    className={`text-xs px-2 py-1 rounded-full ${output.required
                      ? 'bg-gray-200 text-gray-800 border border-gray-300 font-medium'
                      : 'bg-gray-100 text-gray-700'
                      }`}
                  >
                    {output.key}: {output.value}
                  </span>
                ))}
              </div>
            </div>
          </div>

          {/* Queue Section */}
          <div className="px-3 pt-2 pb-0 w-full">
            <div className="w-full text-left text-xs font-medium text-gray-700 uppercase tracking-wide mb-1">Queue</div>

            <div className="w-full pt-1 pb-6">
              {/* Pending Events */}
              {pendingEvents.length > 0 && (
                <div className="flex items-center w-full p-2 bg-gray-100 rounded-lg mt-1">
                  <div className="rounded-full bg-[var(--lightest-orange)] text-[var(--dark-orange)] w-6 h-6 mr-2 flex items-center justify-center">
                    <span className="material-symbols-outlined" style={{ fontSize: '19px' }}>how_to_reg</span>
                  </div>
                  <a
                    href="#"
                    className="min-w-0 font-semibold text-sm flex items-center hover:underline"
                  >
                    <div className="truncate">Pending ({pendingEvents.length})</div>
                  </a>
                </div>
              )}

              {/* Waiting Events */}
              {waitingEvents.length > 0 && (
                <div className="flex items-center w-full p-2 bg-gray-100 rounded-lg mt-1">
                  <div className="rounded-full bg-[var(--lightest-orange)] text-[var(--dark-orange)] w-6 h-6 mr-2 flex items-center justify-center">
                    <span className="material-symbols-outlined" style={{ fontSize: '19px' }}>how_to_reg</span>
                  </div>
                  <a
                    href="#"
                    className="min-w-0 font-semibold text-sm flex items-center hover:underline"
                  >
                    <div className="truncate">Waiting Approval ({waitingEvents.length})</div>
                  </a>
                </div>
              )}

              {/* Show empty state when no queue items */}
              {!pendingEvents.length && !waitingEvents.length && (
                <div className="text-sm text-gray-500 italic py-2">No queue activity</div>
              )}
            </div>
          </div>
        </>
      )}

      {/* Custom Handles */}
      <CustomBarHandle type="target" connections={props.data.connections} conditions={props.data.conditions} />
      <CustomBarHandle type="source" />

      {/* Discard Confirmation Dialog */}
      <ConfirmDialog
        isOpen={showDiscardConfirm}
        title="Discard Stage"
        message="Are you sure you want to discard this stage? This action cannot be undone."
        confirmText="Discard"
        cancelText="Cancel"
        confirmVariant="danger"
        onConfirm={handleDiscardStage}
        onCancel={() => setShowDiscardConfirm(false)}
      />
    </div>
  );
};

