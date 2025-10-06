import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { SuperplaneEventSourceSpec, IntegrationsIntegrationRef, EventSourceEventType, SuperplaneFilter, SuperplaneFilterOperator, SuperplaneEventSourceSchedule } from '@/api-client/types.gen';
import { useIntegrations } from '../hooks/useIntegrations';
import { useEditModeState } from '../hooks/useEditModeState';
import { useResetEventSourceKey } from '@/hooks/useCanvasData';
import { EditableAccordionSection } from './shared/EditableAccordionSection';
import { ValidationField } from '../../../components/ValidationField';
import { Button } from '@/components/Button/button';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { ConfirmDialog } from './ConfirmDialog';
import IntegrationZeroState from '@/components/IntegrationZeroState';
import { useCanvasStore } from '../store/canvasStore';
import { showErrorToast } from '@/utils/toast';
import { EventSourceFilterTooltip } from '@/components/Tooltip/EventSourceFilterTooltip';
import { NodeContentWrapper } from './shared/NodeContentWrapper';
import { ScheduleConfiguration } from '../../../components/ScheduleConfiguration';
import { getDefaultEventType, getDefaultFilterExpression, getResourceLabel, getResourcePlaceholder, getResourceType, getIntegrationLabel, getEventTypePlaceholder, isRegularEventSource } from '@/utils/components';

interface EventSourceEditModeContentProps {
  data: EventSourceNodeType['data'];
  canvasId: string;
  organizationId: string;
  sourceType: string;
  eventSourceKey?: string;
  nodeId?: string;
  onDataChange?: (data: {
    spec: SuperplaneEventSourceSpec
  }) => void;
  onDelete?: () => void;
  apiError?: string | null;
  shouldValidate?: boolean;
  onValidationResult?: (isValid: boolean) => void;
  integrationError?: boolean;
}

export function EventSourceEditModeContent({
  data,
  canvasId,
  organizationId,
  sourceType,
  eventSourceKey,
  nodeId,
  onDataChange,
  onDelete,
  apiError,
  shouldValidate = false,
  onValidationResult,
  integrationError = false
}: EventSourceEditModeContentProps) {
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationRef | null>(data.integration);
  const [resourceType, setResourceType] = useState(data.resource?.type);
  const [resourceName, setResourceName] = useState(data.resource?.name || '');
  const [eventTypes, setEventTypes] = useState<EventSourceEventType[]>(data.events || []);
  const [schedule, setSchedule] = useState<SuperplaneEventSourceSchedule | null>(data.schedule || null);
  const [apiValidationErrors, setApiValidationErrors] = useState<Record<string, string>>({});
  const [isKeyRevealed, setIsKeyRevealed] = useState(false);
  const [showWebhookInstructions, setShowWebhookInstructions] = useState(false);
  const [showRegenerateConfirm, setShowRegenerateConfirm] = useState(false);
  const resourceNameRef = useRef<HTMLInputElement | null>(null);
  const updateEventSourceKey = useCanvasStore(state => state.updateEventSourceKey)

  const resetKeyMutation = useResetEventSourceKey(canvasId);

  const handleCopyKey = async () => {
    if (eventSourceKey) {
      await navigator.clipboard.writeText(eventSourceKey);
    }
  };

  const handleRegenerateKey = async () => {
    if (data.id) {
      try {
        const { data: { key } } = await resetKeyMutation.mutateAsync(data.id);
        setShowRegenerateConfirm(false);
        updateEventSourceKey(data.id, key!);
      } catch (error) {
        console.error('Failed to regenerate key:', error);
      }
    }
  };

  const handleResourceNameChange = (value: string) => {
    setResourceName(value);

    if (value.trim() !== '') {
      setValidationErrors(prev => ({
        ...prev,
        resourceName: ''
      }));
    }
  };

  // Filter management functions
  const addEventType = useCallback(() => {
    const newEventType: EventSourceEventType = {
      type: getDefaultEventType(sourceType),
      filters: [],
      filterOperator: 'FILTER_OPERATOR_AND'
    };
    setEventTypes(prev => [...prev, newEventType]);
  }, [sourceType]);

  const updateEventType = useCallback((index: number, updates: Partial<EventSourceEventType>) => {
    setEventTypes(prev => prev.map((eventType, i) =>
      i === index ? { ...eventType, ...updates } : eventType
    ));
  }, []);

  const removeEventType = useCallback((index: number) => {
    setEventTypes(prev => prev.filter((_, i) => i !== index));
  }, []);

  const addFilter = useCallback((eventTypeIndex: number) => {
    const newFilter: SuperplaneFilter = {
      type: 'FILTER_TYPE_DATA',
      data: { expression: getDefaultFilterExpression(sourceType) }
    };

    setEventTypes(prev => prev.map((eventType, i) =>
      i === eventTypeIndex ? {
        ...eventType,
        filters: [...(eventType.filters || []), newFilter]
      } : eventType
    ));
  }, [sourceType]);

  const updateFilter = useCallback((eventTypeIndex: number, filterIndex: number, updates: Partial<SuperplaneFilter>) => {
    setEventTypes(prev => prev.map((eventType, i) =>
      i === eventTypeIndex ? {
        ...eventType,
        filters: eventType.filters?.map((filter, j) =>
          j === filterIndex ? { ...filter, ...updates } : filter
        )
      } : eventType
    ));
  }, []);

  const removeFilter = useCallback((eventTypeIndex: number, filterIndex: number) => {
    setEventTypes(prev => prev.map((eventType, i) =>
      i === eventTypeIndex ? {
        ...eventType,
        filters: eventType.filters?.filter((_, j) => j !== filterIndex)
      } : eventType
    ));
  }, []);

  const toggleFilterOperator = useCallback((eventTypeIndex: number) => {
    const current = eventTypes[eventTypeIndex]?.filterOperator || 'FILTER_OPERATOR_AND';
    const newOperator: SuperplaneFilterOperator =
      current === 'FILTER_OPERATOR_AND' ? 'FILTER_OPERATOR_OR' : 'FILTER_OPERATOR_AND';

    updateEventType(eventTypeIndex, { filterOperator: newOperator });
  }, [eventTypes, updateEventType]);

  const parseApiError = (errorMessage: string) => {
    const errors: Record<string, string> = {};

    if (errorMessage.includes('not found')) {
      if (errorMessage.includes('project')) {
        errors.resourceName = 'Project not found. Please check the project name and ensure it exists in Semaphore.';
      } else if (errorMessage.includes('repository')) {
        errors.resourceName = 'Repository not found. Please check the repository name and ensure your Personal Access Token (PAT) has access to it.';
      }
    }

    return errors;
  };

  const validateAllFields = (setErrors?: (errors: Record<string, string>) => void) => {
    const errors: Record<string, string> = {};

    // Handle scheduled event sources separately
    if (sourceType === 'scheduled') {
      validateScheduledEventSource(errors);
      if (setErrors) setErrors(errors);
      return Object.keys(errors).length === 0;
    }

    // Validate integration requirements for semaphore and github
    validateIntegrationRequirements(errors);

    // Validate event types and their filters
    validateEventTypesAndFilters(errors);

    if (setErrors) setErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const validateScheduledEventSource = (errors: Record<string, string>) => {
    if (!schedule) {
      errors.schedule = 'Schedule is required for scheduled event sources';
      return;
    }

    if (!schedule.type || schedule.type === 'TYPE_UNKNOWN') {
      errors.scheduleType = 'Schedule type is required';
      return;
    }

    if (schedule.type === 'TYPE_DAILY' && !schedule.daily?.time) {
      errors.time = 'Time is required for daily schedule';
      return;
    }

    if (schedule.type === 'TYPE_WEEKLY') {
      if (!schedule.weekly?.weekDay || schedule.weekly.weekDay === 'WEEK_DAY_UNKNOWN') {
        errors.weekDay = 'Day of week is required for weekly schedule';
      }
      if (!schedule.weekly?.time) {
        errors.time = 'Time is required for weekly schedule';
      }
    }
  };

  const validateIntegrationRequirements = (errors: Record<string, string>) => {
    if (isRegularEventSource(sourceType)) {
      return;
    }

    if (!selectedIntegration) {
      errors.integration = 'Integration is required';
    }

    if (!resourceName || resourceName.trim() === '') {
      const label = getResourceLabel(sourceType);
      errors.resourceName = label ?
        `${label} is required` :
        'Resource name is required';
    }
  };

  const validateEventTypesAndFilters = (errors: Record<string, string>) => {
    eventTypes.forEach((eventType, index) => {
      if (!eventType.type || eventType.type.trim() === '') {
        errors[`eventType_${index}`] = 'Event type is required';
      }

      if (!eventType.filters || eventType.filters.length === 0) {
        return;
      }

      const emptyFilters = getEmptyFilterIndices(eventType.filters);
      if (emptyFilters.length === 0) {
        return;
      }

      const filterErrorMessage = emptyFilters.length === 1
        ? `Filter ${emptyFilters[0]} is incomplete - please fill all fields`
        : `Filters ${emptyFilters.join(', ')} are incomplete - please fill all fields`;

      errors[`eventType_${index}_filters`] = filterErrorMessage;
    });
  };

  const getEmptyFilterIndices = (filters: SuperplaneFilter[]): number[] => {
    const emptyFilters: number[] = [];

    filters.forEach((filter, filterIndex) => {
      const isEmpty = filter.type === 'FILTER_TYPE_DATA'
        ? !filter.data?.expression || filter.data.expression.trim() === ''
        : !filter.header?.expression || filter.header.expression.trim() === '';

      if (isEmpty) {
        emptyFilters.push(filterIndex + 1);
      }
    });

    return emptyFilters;
  };

  useEffect(() => {
    if (apiError) {
      const parsedErrors = parseApiError(apiError);
      setApiValidationErrors(parsedErrors);

      showErrorToast(apiError);
    } else {
      setApiValidationErrors({});
    }
  }, [apiError]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape' && onDelete) {
      onDelete();
    }
  }, [onDelete]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown]);

  const {
    openSections,
    setOpenSections,
    originalData,
    validationErrors,
    setValidationErrors,
    handleAccordionToggle,
    handleDataChange,
    syncWithIncomingData
  } = useEditModeState({
    initialData: {
      spec: {
        integration: data.integration,
        resource: data.resource,
        events: data.events,
        schedule: data.schedule
      } as SuperplaneEventSourceSpec
    },
    onDataChange,
    validateAllFields
  });

  const combinedErrors = { ...validationErrors, ...apiValidationErrors };

  useEffect(() => {
    if (shouldValidate) {
      const isValid = validateAllFields(setValidationErrors);
      if (onValidationResult) {
        onValidationResult(isValid);
      }
    }
  }, [shouldValidate, selectedIntegration, resourceName, sourceType, eventTypes, schedule, onValidationResult]);

  useEffect(() => {
    const sections = ['general'];

    if (sourceType === 'scheduled') {
      sections.push('schedule');
      setOpenSections(sections);
      return;
    }

    if (sourceType === 'webhook') {
      sections.push('webhook');
      sections.push('filters');
      setOpenSections(sections);
      return;
    }

    if (sourceType === 'manual') {
      setOpenSections(sections);
      return;
    }

    sections.push('integration');
    sections.push('filters');
    setOpenSections(sections);
  }, [setOpenSections, sourceType]);

  useEffect(() => {
    syncWithIncomingData(
      {
        spec: {
          integration: data.integration,
          resource: data.resource,
          events: data.events,
          schedule: data.schedule
        } as SuperplaneEventSourceSpec
      },
      (incomingData) => {
        setSelectedIntegration(incomingData.spec.integration || null);
        const integrationData = [...canvasIntegrations, ...orgIntegrations].find(
          int => int.metadata?.name === incomingData.spec.integration?.name
        );
        setResourceType(incomingData.spec.resource?.type || getResourceType(integrationData?.spec?.type || ''));
        setResourceName(incomingData.spec.resource?.name || '');
        setEventTypes(incomingData.spec.events || []);
        setSchedule(incomingData.spec.schedule || null);
      }
    );
  }, [data, sourceType, syncWithIncomingData]);

  const { data: canvasIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  const webhookUrl = useMemo(() => {
    const baseUrl = import.meta.env.VITE_BASE_URL || 'https://app.superplane.com';
    return `${baseUrl}/api/v1/sources/${data.id}`;
  }, [data.id]);

  const webhookExampleCode = useMemo(() => {
    return `export SOURCE_KEY="${eventSourceKey || 'SOURCE_KEY'}"
export URL="${webhookUrl || 'WEBHOOK_URL'}"
export EVENT='{"ref":"v1.0","ref_type":"tag"}'
export SIGNATURE=$(echo -n "$EVENT" | openssl dgst -sha256 -hmac "$SOURCE_KEY" | awk '{print $2}')

curl -X POST \\
  -H "X-Signature-256: sha256=$SIGNATURE" \\
  -H "Content-Type: application/json" \\
  --data "$EVENT" \\
  $URL`;
  }, [eventSourceKey, webhookUrl]);

  const allIntegrations = [...canvasIntegrations, ...orgIntegrations];
  const availableIntegrations = !isRegularEventSource(sourceType)
    ? allIntegrations.filter(integration => integration.spec?.type === sourceType)
    : [];
  const requireIntegration = !isRegularEventSource(sourceType);
  const zeroStateLabel = useMemo(() => {
    return getIntegrationLabel(sourceType) || `${sourceType} integrations`;
  }, [sourceType]);

  const normalizeSpecForComparison = (spec: SuperplaneEventSourceSpec) => {
    return {
      integration: spec.integration || null,
      resource: spec.resource || null
    };
  };

  const hasActualChanges = () => {
    const currentSpec = {
      integration: selectedIntegration,
      resource: resourceType && resourceName ? { type: resourceType, name: resourceName } : null
    };

    const normalizedOriginal = normalizeSpecForComparison(originalData.spec);
    const normalizedCurrent = normalizeSpecForComparison(currentSpec as SuperplaneEventSourceSpec);

    return JSON.stringify(normalizedOriginal) !== JSON.stringify(normalizedCurrent);
  };

  useEffect(() => {
    if (!onDataChange) return;

    const spec: SuperplaneEventSourceSpec = {
      type: sourceType
    };

    if (sourceType === 'scheduled') {
      if (schedule) {
        spec.schedule = schedule;
      }
      handleDataChange({ spec });
      return;
    }

    if (!isRegularEventSource(sourceType) && selectedIntegration) {
      spec.integration = selectedIntegration;

      if (resourceType && resourceName) {
        spec.resource = {
          type: resourceType,
          name: resourceName
        };
      }
    }

    if (eventTypes.length > 0) {
      spec.events = eventTypes;
    }

    handleDataChange({ spec });
  }, [selectedIntegration, resourceType, resourceName, eventTypes, schedule, sourceType, onDataChange, handleDataChange]);

  // Auto-select first integration when integrations become available
  useEffect(() => {
    if (availableIntegrations.length > 0 && !selectedIntegration && requireIntegration) {
      const firstIntegration = availableIntegrations[0];
      setSelectedIntegration({
        name: firstIntegration.metadata?.name,
        domainType: firstIntegration.metadata?.domainType
      });

      // Set default resource type based on integration type
      setResourceType(getResourceType(firstIntegration.spec?.type || ''));
    }
  }, [availableIntegrations, selectedIntegration, requireIntegration]);

  useEffect(() => {
    resourceNameRef.current?.focus();
  }, [selectedIntegration])

  const revertSection = (section: string) => {
    switch (section) {
      case 'integration':
        setSelectedIntegration(originalData.spec.integration || null);
        const integrationData = allIntegrations.find(
          int => int.metadata?.name === originalData.spec.integration?.name
        );
        setResourceType(originalData.spec.resource?.type || getResourceType(integrationData?.spec?.type || ''));
        setResourceName(originalData.spec.resource?.name || '');
        break;
      case 'schedule':
        setSchedule(originalData.spec.schedule || null);
        break;
      case 'filters':
        setEventTypes(originalData.spec.events || []);
        break;
    }
  };

  const handleIntegrationChange = (integrationName: string) => {
    const integration = availableIntegrations.find(int => int.metadata?.name === integrationName);
    if (integration) {
      setSelectedIntegration({
        name: integration.metadata?.name,
        domainType: integration.metadata?.domainType
      });

      setResourceType(getResourceType(integration.spec?.type || ''));

      setValidationErrors(prev => ({
        ...prev,
        integration: ''
      }));
    }
  };

  return (
    <NodeContentWrapper nodeId={nodeId} className="border-t border-gray-200 dark:border-gray-700">
      <div className="">
        {requireIntegration && availableIntegrations.length === 0 && (
          <IntegrationZeroState
            integrationType={sourceType}
            label={zeroStateLabel}
            canvasId={canvasId}
            organizationId={organizationId}
            hasError={integrationError}
          />
        )}


        {/* Configuration Section */}
        {availableIntegrations.length > 0 && sourceType === 'semaphore' && (
          <EditableAccordionSection
            id="integration"
            title="Semaphore Configuration"
            isOpen={openSections.includes('integration')}
            onToggle={handleAccordionToggle}
            isModified={hasActualChanges()}
            onRevert={revertSection}
            requiredBadge={true}
          >
            <div className="space-y-3">
              <ValidationField
                label="Semaphore integration"
                error={combinedErrors.integration}
                required={true}
              >
                <select
                  value={selectedIntegration?.name || ''}
                  onChange={(e) => handleIntegrationChange(e.target.value)}
                  className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors.integration
                    ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                    : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                    }`}
                >
                  <option value="">Select a Semaphore integration...</option>
                  {availableIntegrations.map((integration) => (
                    <option key={integration.metadata?.id} value={integration.metadata?.name}>
                      {integration.metadata?.name}
                    </option>
                  ))}
                </select>
              </ValidationField>

              {(selectedIntegration || combinedErrors.resourceName) && (
                <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                  <ValidationField
                    label={`${getResourceLabel(sourceType)} Name`}
                    error={combinedErrors.resourceName}
                    required={true}
                  >
                    <input
                      ref={resourceNameRef}
                      type="text"
                      value={resourceName}
                      onChange={(e) => handleResourceNameChange(e.target.value)}
                      placeholder={getResourcePlaceholder(sourceType)}
                      className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors.resourceName
                        ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                    />
                  </ValidationField>
                </div>
              )}
            </div>
          </EditableAccordionSection>
        )}

        {availableIntegrations.length > 0 && sourceType === 'github' && (
          <EditableAccordionSection
            id="integration"
            title="GitHub Configuration"
            isOpen={openSections.includes('integration')}
            onToggle={handleAccordionToggle}
            isModified={hasActualChanges()}
            onRevert={revertSection}
            requiredBadge={true}
          >
            <div className="space-y-3">
              <ValidationField
                label="GitHub integration"
                error={combinedErrors.integration}
                required={true}
              >
                <select
                  value={selectedIntegration?.name || ''}
                  onChange={(e) => handleIntegrationChange(e.target.value)}
                  className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors.integration
                    ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                    : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                    }`}
                >
                  <option value="">Select a GitHub integration...</option>
                  {availableIntegrations.map((integration) => (
                    <option key={integration.metadata?.id} value={integration.metadata?.name}>
                      {integration.metadata?.name}
                    </option>
                  ))}
                </select>
              </ValidationField>

              {(selectedIntegration || combinedErrors.resourceName) && (
                <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                  <ValidationField
                    label={`${getResourceLabel(sourceType)} Name`}
                    error={combinedErrors.resourceName}
                    required={true}
                  >
                    <input
                      ref={resourceNameRef}
                      type="text"
                      value={resourceName}
                      onChange={(e) => handleResourceNameChange(e.target.value)}
                      placeholder={getResourcePlaceholder(sourceType)}
                      className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors.resourceName
                        ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                    />
                  </ValidationField>
                </div>
              )}
            </div>
          </EditableAccordionSection>
        )}

        {/* Webhook Configuration Section */}
        {sourceType === 'webhook' && (
          <EditableAccordionSection
            id="webhook"
            title="Webhook Configuration"
            isOpen={openSections.includes('webhook')}
            onToggle={handleAccordionToggle}
            isModified={false}
            onRevert={revertSection}
          >
            <div className="space-y-4">
              {!Number.isNaN(Number(data.id)) ? (
                <div className="text-sm text-amber-600 bg-amber-50 dark:bg-amber-900/20 p-3 rounded-md">
                  Save the component to get the webhook parameters (URL and Key)
                </div>
              ) : (
                <>
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        Webhook URL
                      </label>
                      <div className="flex">
                        <input
                          type="text"
                          value={webhookUrl}
                          readOnly
                          className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-l-md bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm focus:outline-none"
                        />
                        <Button
                          outline
                          onClick={() => navigator.clipboard.writeText(webhookUrl)}
                          className="rounded-l-none border-l-0 px-3 py-2 text-sm flex items-center"
                        >
                          <MaterialSymbol name="content_copy" size="sm" />
                        </Button>
                      </div>
                    </div>

                    {eventSourceKey && (
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                          Signing Key
                        </label>
                        <div className="flex">
                          <input
                            type={isKeyRevealed ? "text" : "password"}
                            value={eventSourceKey}
                            readOnly
                            className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-l-md bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm focus:outline-none"
                          />
                          <Button
                            outline
                            onClick={() => setIsKeyRevealed(!isKeyRevealed)}
                            className="rounded-none border-l-0 border-r-0 px-3 py-2 text-sm flex items-center"
                          >
                            <MaterialSymbol name={isKeyRevealed ? "visibility_off" : "visibility"} size="sm" />
                          </Button>
                          <Button
                            outline
                            onClick={handleCopyKey}
                            className="rounded-l-none border-l-0 px-3 py-2 text-sm flex items-center"
                          >
                            <MaterialSymbol name="content_copy" size="sm" />
                          </Button>
                        </div>


                      </div>
                    )}

                    <div className="nodrag rounded-md border border-gray-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 p-4 relative">
                      <div className="flex items-start gap-3 min-w-0">
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-gray-900 dark:text-white">You will need to sign the webhook payload for events to be delivered.</div>
                          <button
                            type="button"
                            className="mt-2 text-sm text-blue-600 dark:text-blue-300 hover:underline"
                            onClick={() => setShowWebhookInstructions(!showWebhookInstructions)}
                          >
                            See how to use this
                          </button>
                          {showWebhookInstructions && (
                            <div className="relative mt-3">
                              <textarea
                                readOnly
                                value={webhookExampleCode}
                                className="nodrag w-full text-xs text-zinc-700 dark:text-zinc-300 bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-600 p-2 pr-10 rounded font-mono resize-none focus:outline-none overflow-scroll"
                                style={{ height: '370px' }}
                              />
                              <button
                                onClick={() => navigator.clipboard.writeText(webhookExampleCode)}
                                className="absolute top-2 right-2 p-1 text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-200 bg-white dark:bg-zinc-700 rounded border border-gray-200 dark:border-gray-600 h-6 w-6 flex items-center"
                              >
                                <MaterialSymbol name="content_copy" size="sm" />
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="mt-3">
                      <Button
                        outline
                        onClick={() => setShowRegenerateConfirm(true)}
                        className="text-orange-600 border-orange-300 hover:bg-orange-50 dark:text-orange-400 dark:border-orange-600 dark:hover:bg-orange-900/20 px-3 py-2 text-sm flex items-center"
                      >
                        <MaterialSymbol name="refresh" size="sm" className="mr-2" />
                        Regenerate Signing Key
                      </Button>
                    </div>
                  </div>
                </>
              )}
            </div>
          </EditableAccordionSection>
        )}

        {/* Schedule Section - Only for scheduled event sources */}
        {sourceType === 'scheduled' && (
          <EditableAccordionSection
            id="schedule"
            title="Schedule"
            isOpen={openSections.includes('schedule')}
            onToggle={handleAccordionToggle}
            isModified={JSON.stringify(schedule) !== JSON.stringify(originalData.spec.schedule || null)}
            onRevert={revertSection}
            requiredBadge={true}
          >
            <ScheduleConfiguration
              schedule={schedule}
              onScheduleChange={setSchedule}
              errors={combinedErrors}
            />
          </EditableAccordionSection>
        )}

        {/* Event Types and Filters Section - Only for webhook and event sources with integrations */}
        {(sourceType === 'webhook' || (!isRegularEventSource(sourceType) && availableIntegrations.length > 0)) && <EditableAccordionSection
          id="filters"
          title="Filters"
          isOpen={openSections.includes('filters')}
          onToggle={handleAccordionToggle}
          isModified={JSON.stringify(eventTypes) !== JSON.stringify(originalData.spec.events || [])}
          onRevert={revertSection}
          count={eventTypes.length}
          countLabel="event types"
        >
          <div className="space-y-4">
            {eventTypes.map((eventType, eventTypeIndex) => (
              <div key={eventTypeIndex} className="border border-zinc-200 dark:border-zinc-600 rounded-lg p-4 space-y-4">
                <div className="flex justify-between items-start">
                  <div className="flex-1 space-y-3">
                    <ValidationField
                      label="Event Type"
                      error={combinedErrors[`eventType_${eventTypeIndex}`]}
                      required={true}
                    >
                      <input
                        type="text"
                        value={eventType.type || ''}
                        onChange={(e) => updateEventType(eventTypeIndex, { type: e.target.value })}
                        placeholder={getEventTypePlaceholder(sourceType)}
                        className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors[`eventType_${eventTypeIndex}`]
                          ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                          : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                          }`}
                      />
                    </ValidationField>
                  </div>
                  <button
                    onClick={() => removeEventType(eventTypeIndex)}
                    className="ml-4 text-gray-600 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
                  >
                    <MaterialSymbol name="close" size="sm" />
                  </button>
                </div>

                {/* Filters Section */}
                <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                  <div className="flex justify-start items-center mb-2">
                    <label className="text-sm font-medium text-gray-900 dark:text-zinc-100">Filters</label>
                    <EventSourceFilterTooltip>
                      <div className="flex items-center ml-2">
                        <MaterialSymbol name="help" size="sm" className="text-zinc-400 hover:text-zinc-600 cursor-help" />
                      </div>
                    </EventSourceFilterTooltip>
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    Pro tip: Use <a href="https://expr-lang.org/docs/language-definition" target="_blank" rel="noopener noreferrer" className="text-blue-600 dark:text-blue-400 hover:underline">Expr</a> to parse payload data
                  </div>
                  <div className="space-y-2">
                    {(eventType.filters || []).map((filter, filterIndex) => (
                      <div key={filterIndex}>
                        <div className="flex gap-2 items-center bg-zinc-50 dark:bg-zinc-800 p-2 rounded">
                          <select
                            value={filter.type || 'FILTER_TYPE_DATA'}
                            onChange={(e) => {
                              const type = e.target.value as SuperplaneFilter['type'];
                              const updates: Partial<SuperplaneFilter> = { type };
                              if (type === 'FILTER_TYPE_DATA') {
                                updates.data = { expression: filter.data?.expression || '' };
                                updates.header = undefined;
                              } else {
                                updates.header = { expression: filter.header?.expression || '' };
                                updates.data = undefined;
                              }
                              updateFilter(eventTypeIndex, filterIndex, updates);
                            }}
                            className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100"
                          >
                            <option value="FILTER_TYPE_DATA">Data</option>
                            <option value="FILTER_TYPE_HEADER">Header</option>
                          </select>
                          <input
                            type="text"
                            value={
                              filter.type === 'FILTER_TYPE_HEADER'
                                ? filter.header?.expression || ''
                                : filter.data?.expression || ''
                            }
                            onChange={(e) => {
                              const expression = e.target.value;
                              const updates: Partial<SuperplaneFilter> = {};
                              if (filter.type === 'FILTER_TYPE_HEADER') {
                                updates.header = { expression };
                              } else {
                                updates.data = { expression };
                              }
                              updateFilter(eventTypeIndex, filterIndex, updates);
                            }}
                            placeholder="Filter expression"
                            className="w-1/2 px-2 py-1 border border-zinc-300 dark:border-zinc-600 rounded text-sm bg-white dark:bg-zinc-700 text-gray-900 dark:text-zinc-100"
                          />
                          <button
                            onClick={() => removeFilter(eventTypeIndex, filterIndex)}
                            className="text-gray-600 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
                          >
                            <MaterialSymbol name="close" size="sm" />
                          </button>
                        </div>
                        {/* OR/AND toggle between filters */}
                        {filterIndex < (eventType.filters?.length || 0) - 1 && (
                          <div className="flex justify-center py-1">
                            <button
                              onClick={() => toggleFilterOperator(eventTypeIndex)}
                              className="px-3 py-1 text-xs bg-zinc-200 dark:bg-zinc-700 text-gray-900 dark:text-zinc-100 rounded-full hover:bg-zinc-300 dark:hover:bg-zinc-600"
                            >
                              {eventType.filterOperator === 'FILTER_OPERATOR_OR' ? 'OR' : 'AND'}
                            </button>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                  <button
                    onClick={() => addFilter(eventTypeIndex)}
                    className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200 mt-2"
                  >
                    <MaterialSymbol name="add" size="sm" />
                    Add Filter
                  </button>
                  {combinedErrors[`eventType_${eventTypeIndex}_filters`] && (
                    <div className="text-xs text-red-600 mt-1">
                      {combinedErrors[`eventType_${eventTypeIndex}_filters`]}
                    </div>
                  )}
                </div>
              </div>
            ))}
            <button
              onClick={addEventType}
              className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
            >
              <MaterialSymbol name="add" size="sm" />
              Add Event Type
            </button>
          </div>
        </EditableAccordionSection>}
      </div>

      <ConfirmDialog
        isOpen={showRegenerateConfirm}
        title="Regenerate Signing Key"
        message="Are you sure you want to regenerate the signing key? This will invalidate the current key and any webhooks using it will need to be updated."
        confirmText="Regenerate Key"
        cancelText="Cancel"
        confirmVariant="danger"
        onConfirm={handleRegenerateKey}
        onCancel={() => setShowRegenerateConfirm(false)}
      />
    </NodeContentWrapper>
  );
}