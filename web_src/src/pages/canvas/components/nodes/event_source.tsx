import { useState } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import type { EventSourceWithEvents } from '../../store/types';
import { useCreateEventSource } from '@/hooks/useCanvasData';
import { SuperplaneEventSource, SuperplaneEventSourceSpec } from '@/api-client';
import { EventSourceEditModeContent } from '../EventSourceEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { EditModeActionButtons } from '../EditModeActionButtons';
import { useParams } from 'react-router-dom';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';

const EventSourceImageMap = {
  'webhook': <MaterialSymbol className='w-6 h-5 -mt-2' name="webhook" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" />
}

export default function EventSourceNode(props: NodeProps<EventSourceNodeType>) {
  const { orgId } = useParams<{ orgId: string }>();
  const isNewNode = props.id && /^\d+$/.test(props.id);
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ name: string; description?: string; spec: SuperplaneEventSourceSpec } | null>({
    name: props.data.name || '',
    description: props.data.description || '',
    spec: {}
  });
  const [eventSourceName, setEventSourceName] = useState(props.data.name);
  const [eventSourceDescription, setEventSourceDescription] = useState(props.data.description || '');
  const { updateEventSource, setEditingEventSource, removeEventSource, updateEventSourceKey, resetEventSourceKey } = useCanvasStore();
  const currentEventSource = useCanvasStore(state =>
    state.eventSources.find(es => es.metadata?.id === props.id)
  );
  const eventSourceKey = useCanvasStore(state => state.eventSourceKeys[props.id]);
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const organizationId = orgId || '';
  const createEventSourceMutation = useCreateEventSource(canvasId);
  const focusedNodeId = useCanvasStore(state => state.focusedNodeId);

  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingEventSource(props.id);
    setEventSourceName(props.data.name);
    setEventSourceDescription(props.data.description || '');
  };
  const handleSaveEventSource = async (saveAsDraft = false) => {
    if (!currentFormData || !currentEventSource) {
      return;
    }

    const isTemporaryId = currentEventSource.metadata?.id && /^\d+$/.test(currentEventSource.metadata.id);
    const isNewEventSource = !currentEventSource.metadata?.id || isTemporaryId;

    try {
      if (isNewEventSource && !saveAsDraft) {

        const result = await createEventSourceMutation.mutateAsync({
          name: eventSourceName,
          description: eventSourceDescription,
          spec: currentFormData.spec
        });


        const newEventSource = result.data?.eventSource;

        if (newEventSource) {
          const generatedKey = result.data?.key;
          updateEventSourceKey(newEventSource.metadata?.id || '', generatedKey || '');
          removeEventSource(props.id);
        }
      } else if (saveAsDraft) {

        const draftEventSource: EventSourceWithEvents = {
          ...currentEventSource,
          metadata: {
            ...currentEventSource.metadata,
            name: eventSourceName,
            description: eventSourceDescription,
          },
          spec: currentFormData.spec
        };
        updateEventSource(draftEventSource);
      }
      setIsEditMode(false);
      setEditingEventSource(null);
      setCurrentFormData(null);
    } catch (error) {
      console.error(`Failed to ${isNewEventSource ? 'create' : 'update'} event source:`, error);
    }
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingEventSource(null);
    setCurrentFormData(null);

    setEventSourceName(props.data.name);
    setEventSourceDescription(props.data.description || '');
  };

  const handleDiscardEventSource = () => {
    if (currentEventSource?.metadata?.id) {
      removeEventSource(currentEventSource.metadata.id);
    }
    setShowDiscardConfirm(false);
  };

  const handleEventSourceNameChange = (newName: string) => {
    setEventSourceName(newName);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: newName
      });
    }
  };

  const handleEventSourceDescriptionChange = (newDescription: string) => {
    setEventSourceDescription(newDescription);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        description: newDescription
      });
    }
  };

  const handleYamlApply = (updatedData: unknown) => {
    // Handle YAML data application for event source
    const yamlData = updatedData as SuperplaneEventSource;

    if (yamlData.metadata?.name) {
      setEventSourceName(yamlData.metadata.name);
    }
    if (yamlData.metadata?.description) {
      setEventSourceDescription(yamlData.metadata.description);
    }

    // Update form data if available
    if (yamlData.spec && currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: yamlData.metadata?.name || eventSourceName,
        description: yamlData.metadata?.description || eventSourceDescription,
        spec: yamlData.spec
      });
    }
  };

  const eventSourceType = props.data.eventSourceType ? props.data.eventSourceType : (props.data.integration?.name ? "semaphore" : "webhook");

  return (
    <div
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative`}
      style={{ width: '360px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {focusedNodeId === props.id && (
        <EditModeActionButtons
          onSave={handleSaveEventSource}
          onCancel={handleCancelEdit}
          onDiscard={() => setShowDiscardConfirm(true)}
          onEdit={handleEditClick}
          isEditMode={isEditMode}
          entityType="event source"
          entityData={currentFormData ? {
            metadata: {
              name: eventSourceName,
              description: eventSourceDescription
            },
            spec: currentFormData.spec
          } : (currentEventSource ? {
            metadata: {
              name: currentEventSource.metadata?.name,
              description: currentEventSource.metadata?.description
            },
            spec: currentEventSource.spec || {}
          } : null)}
          onYamlApply={handleYamlApply}
        />
      )}

      {/* Header Section */}
      <div className="px-4 py-4 flex justify-between items-start">
        <div className="flex items-start flex-1 min-w-0">
          <div className='max-w-8 mt-2'>
            {EventSourceImageMap[eventSourceType as keyof typeof EventSourceImageMap]}
          </div>
          <div className="flex-1 min-w-0">
            <div className="mb-1">
              <InlineEditable
                value={eventSourceName}
                onSave={handleEventSourceNameChange}
                placeholder="Event source name"
                className="font-bold text-gray-900 text-base text-left px-2 py-1"
                isEditMode={isEditMode}
              />
            </div>
            <div>
              <InlineEditable
                value={eventSourceDescription}
                onSave={handleEventSourceDescriptionChange}
                placeholder={isEditMode ? "Add description..." : "No description available"}
                className="text-gray-600 text-sm text-left px-2 py-1"
                isEditMode={isEditMode}
              />
            </div>
          </div>
        </div>
      </div>

      {isEditMode ? (
        <EventSourceEditModeContent
          data={{
            ...props.data,
            name: eventSourceName,
            description: eventSourceDescription,
            ...(currentFormData && {
              spec: currentFormData.spec
            })
          }}
          canvasId={canvasId}
          organizationId={organizationId}
          eventSourceType={eventSourceType}
          onDataChange={({ spec }) => { if (JSON.stringify(spec) !== JSON.stringify(currentFormData?.spec || {})) setCurrentFormData(prev => ({ ...prev!, spec })) }}
        />
      ) : (
        <>
          {
            eventSourceKey && eventSourceType === "webhook" && (
              <div className="px-3 py-3 border-t w-full text-left bg-amber-50">
                <p className="text-sm text-amber-600">The Webhook Event Source has been created. Save this webhook signature, it will be displayed only once:</p>
                <div className="flex items-center justify-between gap-2 mt-2">
                  <input type="text" value={eventSourceKey} readOnly className="w-full p-2 border border-gray-200 rounded bg-white" />
                  <button className='font-bold bg-gray-100 text-gray-700 p-2 rounded' onClick={() => resetEventSourceKey(props.id)}>
                    Dismiss
                  </button>
                </div>
              </div>
            )}

          <div className="px-3 py-3 border-t border-gray-200 w-full">
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-sm my-2 font-semibold text-gray-500 uppercase tracking-wide">Events</div>
            </div>

            <div className="space-y-2">
              {props.data.events?.length ? (
                props.data.events.map((event) => (
                  <div key={event.id} className="bg-gray-50 rounded-xl p-2">
                    <div className="flex justify-start items-center gap-3 overflow-hidden">
                      <span className="text-sm text-gray-600">
                        <MaterialSymbol name="bolt" size="md" />
                      </span>
                      <span className="truncate">{event.id!}</span>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-sm text-gray-500 italic py-2">No events received</div>
              )}
            </div>
          </div>
        </>
      )}

      <CustomBarHandle type="source" />

      <ConfirmDialog
        isOpen={showDiscardConfirm}
        title="Delete Event Source"
        message="Are you sure you want to delete this event source? This action cannot be undone."
        confirmText="Delete"
        cancelText="Cancel"
        confirmVariant="danger"
        onConfirm={handleDiscardEventSource}
        onCancel={() => setShowDiscardConfirm(false)}
      />
    </div>
  );
}
