import { useMemo, useState, useEffect } from 'react';
import type { NodeProps } from '@xyflow/react';
import CustomBarHandle from './handle';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { useCanvasStore } from '../../store/canvasStore';
import { useCreateEventSource, useUpdateEventSource, useDeleteEventSource } from '@/hooks/useCanvasData';
import { SuperplaneEventSource, SuperplaneEventSourceSpec, superplaneCreateEvent } from '@/api-client';
import { EventSourceEditModeContent } from '../EventSourceEditModeContent';
import { ConfirmDialog } from '../ConfirmDialog';
import { InlineEditable } from '../InlineEditable';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { NodeActionButtons } from '@/components/NodeActionButtons';
import { useParams } from 'react-router-dom';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';
import { twMerge } from 'tailwind-merge';
import { useIntegrations } from '../../hooks/useIntegrations';
import { EventStateItem, EventState } from '../EventStateItem';
import { EventSourceBadges } from '../EventSourceBadges';
import { EventSourceZeroState } from '../../../../components/EventSourceZeroState';
import { createEventSourceDuplicate, focusAndEditNode } from '../../utils/nodeDuplicationUtils';
import { EmitEventModal } from '@/components/EmitEventModal/EmitEventModal';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';

const EventSourceImageMap = {
  'webhook': <MaterialSymbol className='-mt-1 -mb-1' name="webhook" size="xl" />,
  'semaphore': <img src={SemaphoreLogo} alt="Semaphore" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />,
  'github': <img src={GithubLogo} alt="Github" className="w-6 h-6 object-contain dark:bg-white dark:rounded-lg" />,
  'scheduled': <MaterialSymbol className='-mt-1 -mb-1 text-gray-700 dark:text-gray-300' name="schedule" size="xl" />
}

export default function EventSourceNode(props: NodeProps<EventSourceNodeType>) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const eventSourceKey = useCanvasStore(state => state.eventSourceKeys[props.id]);
  const canvasId = useCanvasStore(state => state.canvasId) || '';
  const createEventSourceMutation = useCreateEventSource(canvasId);
  const updateEventSourceMutation = useUpdateEventSource(canvasId);
  const deleteEventSourceMutation = useDeleteEventSource(canvasId);
  const focusedNodeId = useCanvasStore(state => state.focusedNodeId);
  const allEventSources = useCanvasStore(state => state.eventSources);
  const currentEventSource = useCanvasStore(state =>
    state.eventSources.find(es => es.metadata?.id === props.id)
  );
  const eventSourceId = currentEventSource?.metadata?.id;
  const isNewNode = props.id && /^\d+$/.test(props.id);
  const [isEditMode, setIsEditMode] = useState(Boolean(isNewNode));
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false);
  const [currentFormData, setCurrentFormData] = useState<{ name: string; description?: string; spec: SuperplaneEventSourceSpec } | null>({
    name: props.data.name || '',
    description: props.data.description || '',
    spec: {
      integration: props.data.integration ? { ...props.data.integration } : undefined,
      resource: props.data.resource ? { ...props.data.resource } : undefined,
      events: props.data.eventFilters ? [...props.data.eventFilters] : [],
    }
  });

  const [eventSourceName, setEventSourceName] = useState(props.data.name || '');
  const [eventSourceDescription, setEventSourceDescription] = useState(props.data.description || '');
  const [dirtyByUser, setDirtyByUser] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);
  const [validationPassed, setValidationPassed] = useState<boolean | null>(null);
  const [yamlUpdateCounter, setYamlUpdateCounter] = useState(0);
  const [integrationError, setIntegrationError] = useState(false);
  const [showEmitEventModal, setShowEmitEventModal] = useState(false);
  const { setEditingEventSource, removeEventSource, updateEventSource, updateEventSourceKey, resetEventSourceKey, selectEventSourceId, setNodes, setFocusedNodeId, addEventSource } = useCanvasStore();

  const { data: canvasIntegrations = [] } = useIntegrations(canvasId!, "DOMAIN_TYPE_CANVAS");

  const generateEventSourceName = (resourceName: string) => {
    if (!resourceName) return '';
    return `Listen to: ${resourceName}`;
  };

  const validateEventSourceName = (name: string) => {
    if (!name || name.trim() === '') {
      setNameError('Event source name is required');
      return false;
    }

    const isDuplicate = allEventSources.some(es =>
      es.metadata?.name?.toLowerCase() === name.toLowerCase() &&
      es.metadata?.id !== props.id
    );

    if (isDuplicate) {
      setNameError('An event source with this name already exists');
      return false;
    }

    setNameError(null);
    return true;
  };

  const validateIntegrationRequirement = () => {
    const requireIntegration = ['semaphore', 'github'].includes(eventSourceType);

    if (!requireIntegration) {
      return true; // Integration not required for webhook type
    }

    const typeIntegrations = canvasIntegrations.filter(int => int.spec?.type === eventSourceType);
    return typeIntegrations.length > 0;
  };

  const handleEditClick = () => {
    setIsEditMode(true);
    setEditingEventSource(props.id);
    setEventSourceName(props.data.name);
    setEventSourceDescription(props.data.description || '');

    // Initialize currentFormData with existing event source data for editing
    if (currentEventSource?.spec) {
      setCurrentFormData({
        name: props.data.name || '',
        description: props.data.description || '',
        spec: currentEventSource.spec
      });
    }
  };

  const handleSaveEventSource = async () => {
    if (!currentFormData || !currentEventSource) {
      return;
    }

    let basicValidationPassed = true;


    if (!validateEventSourceName(eventSourceName)) {
      basicValidationPassed = false;
    }

    if (!validateIntegrationRequirement()) {
      setIntegrationError(true);
      const integrationTypeLabel = eventSourceType;
      setApiError(`${integrationTypeLabel} integration is required but not configured. Please add a ${integrationTypeLabel} integration to continue.`);
      basicValidationPassed = false;
    }

    if ((eventSourceType === 'webhook' || validationPassed === true) && basicValidationPassed) {
      proceedWithSave();
    }
    setValidationPassed(null);
  };

  const proceedWithSave = async () => {
    if (!currentFormData || !currentEventSource) {
      return;
    }

    const isTemporaryId = currentEventSource.metadata?.id && /^\d+$/.test(currentEventSource.metadata.id);
    const isNewEventSource = !currentEventSource.metadata?.id || isTemporaryId;
    let success = false;

    try {
      setApiError(null);
      if (isNewEventSource) {
        // Keep the temporary node visible during the request
        createEventSourceMutation.mutateAsync({
          name: eventSourceName,
          description: eventSourceDescription,
          spec: currentFormData.spec
        }).then((result) => {
          const newEventSource = result.data?.eventSource;

          if (newEventSource) {
            const generatedKey = result.data?.key;
            updateEventSourceKey(newEventSource.metadata?.id || '', generatedKey || '');

            // Add the new event source to the canvas
            const newEventSourceWithEvents = {
              ...newEventSource,
              events: [],
              eventSourceType: currentFormData.spec.schedule ? 'scheduled' : undefined,
            };
            addEventSource(newEventSourceWithEvents);

            // Remove the temporary node after the new one is added
            removeEventSource(props.id);
            success = true;
          }
        }).catch((error) => {
          setApiError(((error as Error)?.message) || error?.toString() || 'An error occurred');
        });
      } else {
        await updateEventSourceMutation.mutateAsync({
          eventSourceId: currentEventSource.metadata?.id || '',
          name: eventSourceName,
          description: eventSourceDescription,
          spec: currentFormData.spec
        });

        updateEventSource({
          ...currentEventSource,
          metadata: {
            ...currentEventSource.metadata,
            name: eventSourceName,
            description: eventSourceDescription
          },
          spec: currentFormData.spec
        });

        props.data.name = eventSourceName;
        props.data.description = eventSourceDescription;
        success = true;
      }
      if (success) {
        setIsEditMode(false);
        setEditingEventSource(null);
        setCurrentFormData(null);
        setIntegrationError(false);
      }
    } catch (error) {
      setApiError(((error as Error)?.message) || error?.toString() || 'An error occurred');
    }
  };

  const handleCancelEdit = () => {
    setIsEditMode(false);
    setEditingEventSource(null);
    setCurrentFormData(null);
    setIntegrationError(false);

    setEventSourceName(props.data.name);
    setEventSourceDescription(props.data.description || '');

    if (eventSourceKey) {
      resetEventSourceKey(props.id);
    }
  };

  const handleDiscardEventSource = async () => {
    if (currentEventSource?.metadata?.id) {
      const isTemporaryId = /^\d+$/.test(currentEventSource.metadata.id);
      const isRealEventSource = !isTemporaryId;

      if (isRealEventSource) {
        try {
          await deleteEventSourceMutation.mutateAsync(currentEventSource.metadata.id);
        } catch (error) {
          console.error('Failed to delete event source:', error);
          return;
        }
      }
      removeEventSource(currentEventSource.metadata.id);
    }
    setShowDiscardConfirm(false);
  };

  const handleEventSourceNameChange = (newName: string) => {
    setEventSourceName(newName);
    validateEventSourceName(newName);
    if (currentFormData) {
      setCurrentFormData({
        ...currentFormData,
        name: newName
      });
    }

    if (newName === '') {
      setDirtyByUser(false);
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

  const handleDuplicateEventSource = () => {
    if (!currentEventSource) return;

    const duplicatedEventSource = createEventSourceDuplicate(currentEventSource, allEventSources);
    addEventSource(duplicatedEventSource, true);

    focusAndEditNode(
      duplicatedEventSource.metadata?.id || '',
      setFocusedNodeId,
      setEditingEventSource
    );
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
      // Force re-render of EventSourceEditModeContent by incrementing counter
      setYamlUpdateCounter(prev => prev + 1);
    }
  };

  const integration = useMemo(() => {
    const integrationName = props.data.integration?.name;
    return canvasIntegrations.find(integration => integration.metadata?.name === integrationName);
  }, [canvasIntegrations, props.data.integration?.name]);

  const eventSourceType = useMemo(() => {
    if (props.data.eventSourceType)
      return props.data.eventSourceType;

    // Check if this is a scheduled event source
    if (props.data.schedule) {
      return 'scheduled';
    }

    if (integration?.spec?.type) {
      return integration.spec.type;
    }
    return "webhook";
  }, [integration, props.data.eventSourceType, props.data.schedule]);

  // Auto-enter edit mode for webhook with key
  useEffect(() => {
    if (eventSourceKey && eventSourceType === 'webhook' && !isNewNode) {
      setIsEditMode(true);
      setEditingEventSource(props.id);
      setEventSourceName(props.data.name);
      setEventSourceDescription(props.data.description || '');
      setFocusedNodeId(props.id);

      // Initialize currentFormData with existing event source data
      if (currentEventSource?.spec) {
        setCurrentFormData({
          name: props.data.name || '',
          description: props.data.description || '',
          spec: currentEventSource.spec
        });
      }

      setTimeout(() => {
        const currentNodes = useCanvasStore.getState().nodes;
        const updatedNodes = currentNodes.map(node => ({
          ...node,
          selected: node.id === props.id
        }));
        setNodes(updatedNodes);
      }, 100);
    }
  }, [eventSourceKey, eventSourceType, isNewNode, props.id, currentEventSource, props.data.name, props.data.description, setEditingEventSource, setNodes, setFocusedNodeId]);

  const getBorderClass = () => {
    if (props.selected || focusedNodeId === props.id) {
      return 'border-blue-400';
    }
    return 'border-gray-200 dark:border-gray-700';
  };


  return (
    <div
      className={`bg-white dark:bg-zinc-800 rounded-lg shadow-lg border-2 ${getBorderClass()} relative cursor-pointer`}
      style={{ width: '340px', height: isEditMode ? 'auto' : 'auto', boxShadow: 'rgba(128, 128, 128, 0.2) 0px 4px 12px' }}
    >
      {(focusedNodeId === props.id || isEditMode) && (
        <NodeActionButtons
          isNewNode={!!isNewNode}
          onSave={handleSaveEventSource}
          onCancel={handleCancelEdit}
          onDiscard={() => setShowDiscardConfirm(true)}
          onEdit={handleEditClick}
          onDuplicate={!isNewNode ? handleDuplicateEventSource : undefined}
          onSend={eventSourceId ? () => setShowEmitEventModal(true) : undefined}
          onSelect={eventSourceId && !props.id.match(/^\d+$/) ? () => selectEventSourceId(eventSourceId) : undefined}
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
      <div className="px-4 py-4 justify-between items-start">
        <div className="flex items-start justify-between w-full">
        <div className="flex items-start flex-1 min-w-0">
          <div className='max-w-8 mt-1 flex items-center justify-center'>
            {EventSourceImageMap[eventSourceType as keyof typeof EventSourceImageMap]}
          </div>
          <div className="flex-1 min-w-0 ml-2">
            <div className="mb-1">
              <InlineEditable
                value={eventSourceName}
                onSave={handleEventSourceNameChange}
                placeholder="Event source name"
                className={twMerge(`font-bold text-gray-900 dark:text-gray-100 text-base text-left px-2 py-1`,
                  nameError && isEditMode ? 'border border-red-500 rounded-lg' : '',
                  isEditMode ? 'text-sm' : '')}
                onKeyDown={() => isNewNode && setDirtyByUser(true)}
                isEditMode={isEditMode}
                autoFocus={isEditMode && eventSourceType === "webhook"}
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
                value={eventSourceDescription}
                onSave={handleEventSourceDescriptionChange}
                placeholder={isEditMode ? "Add description..." : ""}
                className="text-gray-600 dark:text-gray-400 text-sm text-left px-2 py-1"
                isEditMode={isEditMode}
              />}
            </div>
          </div>
        </div>
        </div>
        {!isEditMode && (
          <>
            <div className="text-xs text-left text-gray-600 dark:text-gray-400 w-full mt-1">{eventSourceDescription || ''}</div>
            {/* Schedule Status */}
            {currentEventSource?.spec?.schedule && (
              <div className="w-full mt-3 space-y-2">
                {/* Schedule Type and Configuration */}
                <div className="flex items-center gap-2 text-xs">
                  <MaterialSymbol name="event_repeat" size="sm" className="text-purple-600 dark:text-purple-400" />
                  <span className="text-gray-700 dark:text-gray-300 font-medium">
                    {currentEventSource.spec.schedule.type === 'TYPE_HOURLY' &&
                      `Hourly, ${currentEventSource.spec.schedule.hourly?.minute || 0} minutes past the hour`
                    }
                    {currentEventSource.spec.schedule.type === 'TYPE_DAILY' &&
                      `Daily at ${currentEventSource.spec.schedule.daily?.time || '00:00'} UTC`
                    }
                    {currentEventSource.spec.schedule.type === 'TYPE_WEEKLY' &&
                      `Weekly on ${currentEventSource.spec.schedule.weekly?.weekDay?.replace('WEEK_DAY_', '').toLowerCase().replace(/^\w/, c => c.toUpperCase()) || 'Monday'} at ${currentEventSource.spec.schedule.weekly?.time || '00:00'} UTC`
                    }
                  </span>
                </div>

                {/* Timestamps - only show if status is available */}
                {currentEventSource?.status?.schedule && (
                  <>
                    <div className="flex items-center gap-2 text-xs">
                      <MaterialSymbol name="history" size="sm" className="text-green-600 dark:text-green-400" />
                      <span className="text-gray-500 dark:text-gray-400">Last:</span>
                      <span className="text-gray-700 dark:text-gray-300">
                        {currentEventSource.status.schedule.lastTrigger
                          ? new Date(currentEventSource.status.schedule.lastTrigger).toLocaleDateString('en-US', {
                            month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC'
                          }) + ' UTC'
                          : '-'
                        }
                      </span>
                    </div>
                    {currentEventSource.status.schedule.nextTrigger && (
                      <div className="flex items-center gap-2 text-xs">
                        <MaterialSymbol name="schedule" size="sm" className="text-blue-600 dark:text-blue-400" />
                        <span className="text-gray-500 dark:text-gray-400">Next:</span>
                        <span className="text-gray-700 dark:text-gray-300">
                          {new Date(currentEventSource.status.schedule.nextTrigger).toLocaleDateString('en-US', {
                            month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC'
                          })} UTC
                        </span>
                      </div>
                    )}
                  </>
                )}
              </div>
            )}
          </>
        )}

      </div>

      {!isEditMode && (
        <EventSourceBadges
          resourceName={props.data.resource?.name}
          currentEventSource={currentEventSource}
          eventSourceType={eventSourceType}
          integration={integration}
        />
      )}

      {isEditMode ? (
        <EventSourceEditModeContent
          key={yamlUpdateCounter}
          nodeId={props.id}
          data={{
            ...props.data,
            name: eventSourceName,
            description: eventSourceDescription,
            ...(currentFormData?.spec && {
              integration: currentFormData.spec.integration,
              resource: currentFormData.spec.resource,
              events: currentFormData.spec.events,
              schedule: currentFormData.spec.schedule,
            })
          }}
          canvasId={canvasId}
          organizationId={organizationId!}
          eventSourceType={eventSourceType}
          eventSourceKey={eventSourceKey}
          onDataChange={({ spec }) => {
            if (JSON.stringify(spec) !== JSON.stringify(currentFormData?.spec || {})) {
              setCurrentFormData(prev => ({ ...prev!, spec }));
              // Clear API errors when user makes changes
              setApiError(null);

              if (isNewNode && !dirtyByUser && !currentEventSource?.isDuplicate) {
                const autoGeneratedName = generateEventSourceName(spec.resource?.name || '');
                setEventSourceName(autoGeneratedName);
                if (currentFormData) {
                  setCurrentFormData(prevFormData => ({ ...prevFormData!, name: autoGeneratedName }));
                }
                validateEventSourceName(autoGeneratedName);
              }
            }
          }}
          onDelete={handleDiscardEventSource}
          apiError={apiError}
          shouldValidate={true}
          onValidationResult={setValidationPassed}
          integrationError={integrationError}
        />
      ) : (
        <>

          {currentEventSource?.events?.length ? (
            <div className="px-3 py-3 pt-2 w-full border-t border-gray-200 dark:border-gray-700">
              <div className="flex items-center w-full justify-between mb-2 py-2">
                <div className="text-sm  font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">Latest Events ({currentEventSource?.status?.history?.received || 0})</div>
              </div>

              <div className="space-y-1">
                {currentEventSource.events.slice(0, 3).map((event) => {
                  // Map event states to our EventState type
                  let eventState: EventState = 'pending';
                  if (event.state === 'STATE_REJECTED') {
                    eventState = 'rejected';
                  } else if (event.state === 'STATE_PROCESSED') {
                    eventState = 'processed';
                  }

                  return (
                    <EventStateItem
                      key={event.id}
                      state={eventState}
                      receivedAt={event.receivedAt}
                      eventType={event.type}
                    />
                  );
                })}
              </div>
            </div>
          ) : (
            // Only show EventSourceZeroState for non-scheduled event sources
            eventSourceType !== 'scheduled' && (
              <EventSourceZeroState
                eventSourceType={eventSourceType}
              />
            )
          )}

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

      {currentEventSource?.metadata?.id && (
        <EmitEventModal
          isOpen={showEmitEventModal}
          onClose={() => setShowEmitEventModal(false)}
          sourceName={currentEventSource.metadata.name || ''}
          loadLastEvent={async () => {
            // For event sources, return the latest event immediately
            return currentEventSource.events?.[0] || null;
          }}
          onSubmit={async (eventType: string, eventData: any) => {
            await superplaneCreateEvent(withOrganizationHeader({
              path: { canvasIdOrName: canvasId! },
              body: {
                sourceType: 'EVENT_SOURCE_TYPE_EVENT_SOURCE',
                sourceId: currentEventSource.metadata!.id,
                type: eventType,
                raw: eventData
              }
            }));
          }}
        />
      )}
    </div>
  );
}
