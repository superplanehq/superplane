import { useState, useMemo } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { StageNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { SuperplaneExecution, SuperplaneInputDefinition, SuperplaneOutputDefinition } from '@/api-client';
import { EditModeContent } from '../EditModeContent';
import { OverlayModal } from '../OverlayModal';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Dropdown, DropdownButton, DropdownItem, DropdownLabel, DropdownMenu } from '@/components/Dropdown/dropdown';


// Define the data type for the deployment card
// Using Record<string, unknown> to satisfy ReactFlow's Node constraint
export default function StageNode(props: NodeProps<StageNodeType>) {
  const [showOverlay, setShowOverlay] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);
  const { selectStageId, updateStage, setEditingStage } = useCanvasStore()
  const currentStage = useCanvasStore(state =>
    state.stages.find(stage => stage.metadata?.id === props.id)
  )

  // Filter events by their state
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

  // Edit mode handlers
  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingStage(props.id);
  };

  const handleSaveAndEdit = (editedData: { label: string; inputs: SuperplaneInputDefinition[]; outputs: SuperplaneOutputDefinition[] }) => {
    if (currentStage) {
      // Save changes using canvas store
      updateStage({
        ...currentStage,
        metadata: {
          ...currentStage.metadata!,
          name: editedData.label
        },
        spec: {
          ...currentStage.spec!,
          inputs: editedData.inputs,
          outputs: editedData.outputs
        }
      });
    }
    setIsEditMode(false);
    setEditingStage(null);
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingStage(null);
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
      style={{ width: isEditMode ? '600px' : '320px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {isEditMode && (
        <div
          className="action-buttons absolute z-50 -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white shadow-lg rounded-lg px-2 py-1 border border-gray-200 z-50"
          onClick={(e) => e.stopPropagation()}
        >
          <Dropdown>
            <DropdownButton plain className='flex items-center gap-2'>
              <MaterialSymbol name="save" size="md" />
              Save
              <MaterialSymbol name="expand_more" size="md" />
            </DropdownButton>
            <DropdownMenu anchor="bottom start">
              <DropdownItem className='flex items-center gap-2'><DropdownLabel>Save & Commit</DropdownLabel></DropdownItem>
              <DropdownItem className='flex items-center gap-2'><DropdownLabel>Save as Draft</DropdownLabel></DropdownItem>
            </DropdownMenu>
          </Dropdown>

        </div>
      )}
      {/* Modal overlay for View Code */}
      <OverlayModal open={showOverlay} onClose={() => setShowOverlay(false)}>
        <h2 style={{ fontSize: 22, fontWeight: 700, marginBottom: 16 }}>Stage Code</h2>
        <div style={{ color: '#444', fontSize: 16, lineHeight: 1.7 }}>
          Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse et urna fringilla, tincidunt nulla nec, dictum erat. Etiam euismod, justo id facilisis dictum, urna massa dictum erat, eget dictum urna massa id justo. Praesent nec facilisis urna. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas.
        </div>
      </OverlayModal>


      {/* Header Section */}
      <div className="px-4 py-4 flex justify-between items-center">
        <div className="flex items-center">
          <span className="material-symbols-outlined mr-2 text-gray-700">rocket_launch</span>
          <p className="mb-0 font-bold ml-1 text-gray-900">{props.data.label}</p>
        </div>
        <div className="flex items-center gap-2">
          {!isEditMode && (
            <button
              onClick={handleEditClick}
              className="p-1 text-gray-500 hover:text-gray-700 transition-colors"
              title="Edit stage"
            >
              <span className="material-symbols-outlined text-base">edit</span>
            </button>
          )}
          {props.data.isDraft && (
            <span className="text-black bg-gray-200 px-2 py-1 rounded-md text-xs">Draft</span>
          )}
        </div>
      </div>

      {isEditMode ? (
        <EditModeContent
          data={props.data}
          onSave={handleSaveAndEdit}
          onCancel={handleCancelEdit}
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
    </div>
  );
};

