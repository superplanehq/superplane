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

export default function EventSourceNode(props: NodeProps<EventSourceNodeType>) {
  const isNewNode = props.id && /^\d+$/.test(props.id);
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ name: string; description?: string; spec: SuperplaneEventSourceSpec } | null>(null);
  const [eventSourceName, setEventSourceName] = useState(props.data.name);
  const [eventSourceDescription, setEventSourceDescription] = useState(props.data.description || '');
  const { updateEventSource, setEditingEventSource, removeEventSource, updateEventSourceKey, resetEventSourceKey } = useCanvasStore();
  const currentEventSource = useCanvasStore(state =>
    state.eventSources.find(es => es.metadata?.id === props.id)
  );
  const eventSourceKey = useCanvasStore(state => state.eventSourceKeys[props.id]);
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const organizationId = '';
  const createEventSourceMutation = useCreateEventSource(canvasId);

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

  return (
    <div
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative`}
      style={{ width: '390px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {isEditMode && (
        <EditModeActionButtons
          onSave={handleSaveEventSource}
          onCancel={handleCancelEdit}
          onDiscard={() => setShowDiscardConfirm(true)}
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
          <span className="material-symbols-outlined mr-2 text-gray-700 mt-1">bolt</span>
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
        <div className="flex items-center gap-2 ml-2">
          {!isEditMode && (
            <button
              onClick={handleEditClick}
              className="p-1 text-gray-500 hover:text-gray-700 transition-colors"
              title="Edit event source"
            >
              <MaterialSymbol name="edit" size="md" />
            </button>
          )}
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
          eventSourceType={props.data.eventSourceType ? props.data.eventSourceType : (props.data.integration?.name ? "semaphore" : "webhook")}
          onDataChange={({ spec }) => { if (JSON.stringify(spec) !== JSON.stringify(currentFormData?.spec || {})) setCurrentFormData(prev => ({ ...prev!, spec })) }}
        />
      ) : (
        <>
          {
            eventSourceKey && props.data.eventSourceType === "webhook" && (
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

          <div className="px-3 py-3 border-t w-full">
            <div className="flex items-center w-full justify-between mb-2">
              <div className="text-xs font-medium text-gray-700 uppercase tracking-wide">Events</div>
              <div className="text-xs text-gray-600">
                {props.data.events?.length || 0} events
              </div>
            </div>

            <div className="space-y-1">
              {props.data.events?.length ? (
                props.data.events.map((event) => (
                  <div key={event.id} className="bg-gray-100 rounded p-2">
                    <div className="flex justify-start items-center gap-3 overflow-hidden">
                      <span className="text-sm text-gray-600">
                        <i className="material-icons f3 fill-black rounded-full bg-[var(--washed-green)] black-60 p-1">bolt</i>
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
