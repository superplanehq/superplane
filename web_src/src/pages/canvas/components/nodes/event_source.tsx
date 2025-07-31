import { useState } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import type { EventSourceWithEvents } from '../../store/types';
import { useCreateEventSource } from '@/hooks/useCanvasData';
import { SuperplaneEventSourceSpec } from '@/api-client';
import { EventSourceEditModeContent } from '../EventSourceEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Dropdown, DropdownButton, DropdownItem, DropdownLabel, DropdownMenu } from '@/components/Dropdown/dropdown';

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

  // Get canvasId and organizationId from the store or current context
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const organizationId = ''; // This should come from context or props
  const createEventSourceMutation = useCreateEventSource(canvasId);

  // Edit mode handlers
  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingEventSource(props.id);
    // Initialize the editable values from current data
    setEventSourceName(props.data.name);
    setEventSourceDescription(props.data.description || '');
  };

  const handleSaveEventSource = async (saveAsDraft = false) => {
    if (!currentFormData || !currentEventSource) {
      return;
    }

    // Check if this is a new/draft event source
    const isTemporaryId = currentEventSource.metadata?.id && /^\d+$/.test(currentEventSource.metadata.id);
    const isNewEventSource = !currentEventSource.metadata?.id || isTemporaryId;

    try {
      if (isNewEventSource && !saveAsDraft) {
        // Create new event source (commit to backend)
        const result = await createEventSourceMutation.mutateAsync({
          name: eventSourceName,
          description: eventSourceDescription,
          spec: currentFormData.spec
        });

        // Update local store with the new event source data from API response
        const newEventSource = result.data?.eventSource;

        if (newEventSource) {
          const generatedKey = result.data?.key;
          updateEventSourceKey(newEventSource.metadata?.id || '', generatedKey || '');
          removeEventSource(props.id);
        }
      } else if (saveAsDraft) {
        // Save as draft (only update local store, don't commit to backend)
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
    // Reset to original values
    setEventSourceName(props.data.name);
    setEventSourceDescription(props.data.description || '');
  };

  // Check if this is a draft/new event source that can be discarded
  const isDraftEventSource = () => {
    if (!currentEventSource) return false;

    // Check if it has a temporary ID (timestamp strings)
    const isTemporaryId = currentEventSource.metadata?.id && /^\d+$/.test(currentEventSource.metadata.id);

    // Check if it doesn't have an ID yet
    const hasNoId = !currentEventSource.metadata?.id;

    return isTemporaryId || hasNoId;
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

  return (
    <div
      className={`bg-white rounded-lg shadow-lg border-2 ${props.selected ? 'border-blue-400' : 'border-gray-200'} relative`}
      style={{ width: '390px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
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
              <DropdownItem className='flex items-center gap-2' onClick={() => handleSaveEventSource(false)}>
                <DropdownLabel>Save & Commit</DropdownLabel>
              </DropdownItem>
              <DropdownItem className='flex items-center gap-2' onClick={() => handleSaveEventSource(true)}>
                <DropdownLabel>Save as Draft</DropdownLabel>
              </DropdownItem>
            </DropdownMenu>
          </Dropdown>

          <button
            onClick={handleCancelEdit}
            className="flex items-center gap-2 px-3 py-2 text-gray-600 hover:text-gray-800 hover:bg-gray-50 rounded-md transition-colors"
            title="Cancel changes"
          >
            <MaterialSymbol name="close" size="md" />
            Cancel
          </button>

          {isDraftEventSource() && (
            <button
              onClick={() => setShowDiscardConfirm(true)}
              className="flex items-center gap-2 px-3 py-2 text-red-600 hover:text-red-800 hover:bg-red-50 rounded-md transition-colors"
              title="Discard this event source"
            >
              <MaterialSymbol name="delete" size="md" />
              Discard
            </button>
          )}
        </div>
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
          data={props.data}
          canvasId={canvasId}
          organizationId={organizationId}
          eventSourceType={props.data.eventSourceType ? props.data.eventSourceType : (props.data.integration?.name ? "semaphore" : "webhook")}
          onDataChange={({ spec }) => { if (JSON.stringify(spec) !== JSON.stringify(currentFormData?.spec || {})) setCurrentFormData(prev => ({ ...prev!, spec })) }}
        />
      ) : (
        <>
          {
            eventSourceKey && (
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

      {/* Discard Confirmation Dialog */}
      <ConfirmDialog
        isOpen={showDiscardConfirm}
        title="Discard Event Source"
        message="Are you sure you want to discard this event source? This action cannot be undone."
        confirmText="Discard"
        cancelText="Cancel"
        confirmVariant="danger"
        onConfirm={handleDiscardEventSource}
        onCancel={() => setShowDiscardConfirm(false)}
      />
    </div>
  );
}
