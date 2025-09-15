import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { SuperplaneEventSourceSpec, IntegrationsIntegrationRef, EventSourceEventType, SuperplaneFilter, SuperplaneFilterOperator } from '@/api-client/types.gen';
import { useIntegrations } from '../hooks/useIntegrations';
import { useEditModeState } from '../hooks/useEditModeState';
import { useResetEventSourceKey } from '@/hooks/useCanvasData';
import { EditableAccordionSection } from './shared/EditableAccordionSection';
import { ValidationField } from './shared/ValidationField';
import { Button } from '@/components/Button/button';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { ConfirmDialog } from './ConfirmDialog';
import IntegrationZeroState from '@/components/IntegrationZeroState';
import { useCanvasStore } from '../store/canvasStore';
import { showErrorToast } from '@/utils/toast';

interface EventSourceEditModeContentProps {
  data: EventSourceNodeType['data'];
  canvasId: string;
  organizationId: string;
  eventSourceType?: string;
  eventSourceKey?: string;
  onDataChange?: (data: {
    spec: SuperplaneEventSourceSpec
  }) => void;
  onDelete?: () => void;
  apiError?: string | null;
  shouldValidate?: boolean;
  onValidationResult?: (isValid: boolean) => void;
}

export function EventSourceEditModeContent({
  data,
  canvasId,
  organizationId,
  eventSourceType = 'webhook',
  eventSourceKey,
  onDataChange,
  onDelete,
  apiError,
  shouldValidate = false,
  onValidationResult
}: EventSourceEditModeContentProps) {
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationRef | null>(data.integration);
  const [resourceType, setResourceType] = useState(data.resource?.type);
  const [resourceName, setResourceName] = useState(data.resource?.name || '');
  const [eventTypes, setEventTypes] = useState<EventSourceEventType[]>(data.events || []);
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
      type: '',
      filters: [],
      filterOperator: 'FILTER_OPERATOR_AND'
    };
    setEventTypes(prev => [...prev, newEventType]);
  }, []);

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
      data: { expression: '' }
    };

    setEventTypes(prev => prev.map((eventType, i) =>
      i === eventTypeIndex ? {
        ...eventType,
        filters: [...(eventType.filters || []), newFilter]
      } : eventType
    ));
  }, []);

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

  const getEventTypePlaceholder = (eventSourceType: string): string => {
    switch (eventSourceType) {
      case 'github':
        return 'e.g., push, pull_request, deployment';
      case 'semaphore':
        return 'e.g., pipeline_done';
      default:
        return 'Event type';
    }
  };


  const parseApiError = (errorMessage: string) => {
    const errors: Record<string, string> = {};

    if (errorMessage.includes('not found')) {
      if (errorMessage.includes('project')) {
        errors.resourceName = 'Project not found. Please check the project name and ensure it exists in Semaphore.';
      } else if (errorMessage.includes('repository')) {
        errors.resourceName = 'Repository not found. Please check the repository name and ensure it exists.';
      }
    }

    if (errorMessage.includes('already exists')) {
      errors.resourceName = 'An Event Source for this resource already exists. Please choose a different name.';
    }

    return errors;
  };

  const validateAllFields = (setErrors?: (errors: Record<string, string>) => void) => {
    const errors: Record<string, string> = {};

    if (eventSourceType === 'semaphore' || eventSourceType === 'github') {
      if (!selectedIntegration) {
        errors.integration = 'Integration is required';
      }

      if (!resourceName || resourceName.trim() === '') {
        errors.resourceName = eventSourceType === 'semaphore' ? 'Project name is required' : 'Resource name is required';
      }
    }

    // Validate event types and filters
    eventTypes.forEach((eventType, index) => {
      if (!eventType.type || eventType.type.trim() === '') {
        errors[`eventType_${index}`] = 'Event type is required';
      }

      if (eventType.filters && eventType.filters.length > 0) {
        const emptyFilters: number[] = [];

        eventType.filters.forEach((filter, filterIndex) => {
          if (filter.type === 'FILTER_TYPE_DATA') {
            if (!filter.data?.expression || filter.data.expression.trim() === '') {
              emptyFilters.push(filterIndex + 1);
            }
          } else if (filter.type === 'FILTER_TYPE_HEADER') {
            if (!filter.header?.expression || filter.header.expression.trim() === '') {
              emptyFilters.push(filterIndex + 1);
            }
          }
        });

        if (emptyFilters.length > 0) {
          if (emptyFilters.length === 1) {
            errors[`eventType_${index}_filters`] = `Filter ${emptyFilters[0]} is incomplete - please fill all fields`;
          } else {
            errors[`eventType_${index}_filters`] = `Filters ${emptyFilters.join(', ')} are incomplete - please fill all fields`;
          }
        }
      }
    });

    if (setErrors) {
      setErrors(errors);
    }

    return Object.keys(errors).length === 0;
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
        events: data.events
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
  }, [shouldValidate, selectedIntegration, resourceName, eventSourceType, eventTypes, onValidationResult]);

  useEffect(() => {
    setOpenSections(['general', 'integration', 'webhook', 'filters']);
  }, [setOpenSections]);

  useEffect(() => {
    syncWithIncomingData(
      {
        spec: {
          integration: data.integration,
          resource: data.resource,
          events: data.events
        } as SuperplaneEventSourceSpec
      },
      (incomingData) => {
        setSelectedIntegration(incomingData.spec.integration || null);
        setResourceType(incomingData.spec.resource?.type || (eventSourceType === 'semaphore' ? 'project' : ''));
        setResourceName(incomingData.spec.resource?.name || '');
        setEventTypes(incomingData.spec.events || []);
      }
    );
  }, [data, eventSourceType, syncWithIncomingData]);

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
  const availableIntegrations = allIntegrations.filter(int => int.spec?.type === eventSourceType);
  const requireIntegration = ['semaphore', 'github'].includes(eventSourceType);
  const zeroStateLabel = useMemo(() => {
    switch (eventSourceType) {
      case 'semaphore':
        return 'Semaphore organizations';
      case 'github':
        return 'GitHub accounts';
      default:
        return `${eventSourceType} integrations`;
    }
  }, [eventSourceType]);

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
    if (onDataChange) {
      const spec: SuperplaneEventSourceSpec = {};

      if ((eventSourceType === 'semaphore' || eventSourceType === 'github') && selectedIntegration) {
        spec.integration = selectedIntegration;

        if (resourceType && resourceName) {
          spec.resource = {
            type: resourceType,
            name: resourceName
          };
        }
      }

      // Include event types with filters
      if (eventTypes.length > 0) {
        spec.events = eventTypes;
      }

      handleDataChange({
        spec
      });
    }
  }, [selectedIntegration, resourceType, resourceName, eventTypes, eventSourceType, onDataChange, handleDataChange]);

  // Auto-select first integration when integrations become available
  useEffect(() => {
    if (availableIntegrations.length > 0 && !selectedIntegration && requireIntegration) {
      const firstIntegration = availableIntegrations[0];
      setSelectedIntegration({
        name: firstIntegration.metadata?.name,
        domainType: firstIntegration.metadata?.domainType
      });

      // Set default resource type based on integration type
      if (firstIntegration.spec?.type === 'semaphore') {
        setResourceType('project');
      } else if (firstIntegration.spec?.type === 'github') {
        setResourceType('repository');
      }
    }
  }, [availableIntegrations, selectedIntegration, requireIntegration]);

  useEffect(() => {
    resourceNameRef.current?.focus();
  }, [selectedIntegration])

  const revertSection = (section: string) => {
    switch (section) {
      case 'integration':
        setSelectedIntegration(originalData.spec.integration || null);
        setResourceType(originalData.spec.resource?.type || (eventSourceType === 'semaphore' ? 'project' : ''));
        setResourceName(originalData.spec.resource?.name || '');
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

      if (integration.spec?.type === 'semaphore') {
        setResourceType('project');
      } else if (integration.spec?.type === 'github') {
        setResourceType('repository');
      }

      setValidationErrors(prev => ({
        ...prev,
        integration: ''
      }));
    }
  };

  return (
    <div className="w-full h-full text-left border-t border-gray-200 dark:border-gray-700" onClick={(e) => e.stopPropagation()}>
      <div className="">
        {requireIntegration && availableIntegrations.length === 0 && (
          <IntegrationZeroState
            integrationType={eventSourceType}
            label={zeroStateLabel}
            canvasId={canvasId}
            organizationId={organizationId}
          />
        )}


        {/* Configuration Section */}
        {availableIntegrations.length > 0 && eventSourceType === 'semaphore' && (
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
                    label="Project Name"
                    error={combinedErrors.resourceName}
                    required={true}
                  >
                    <input
                      ref={resourceNameRef}
                      type="text"
                      value={resourceName}
                      onChange={(e) => handleResourceNameChange(e.target.value)}
                      placeholder="my-semaphore-project"
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

        {availableIntegrations.length > 0 && eventSourceType === 'github' && (
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
                    label="Repository Name"
                    error={combinedErrors.resourceName}
                    required={true}
                  >
                    <input
                      ref={resourceNameRef}
                      type="text"
                      value={resourceName}
                      onChange={(e) => handleResourceNameChange(e.target.value)}
                      placeholder="my-repository"
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
        {eventSourceType === 'webhook' && (
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
                                onClick={() => {
                                  navigator.clipboard.writeText(webhookExampleCode);
                                }}
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

        {/* Event Types and Filters Section */}
        {(!requireIntegration || availableIntegrations.length > 0) && <EditableAccordionSection
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
                        placeholder={getEventTypePlaceholder(eventSourceType)}
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
                  <div className="flex justify-between items-center mb-2">
                    <label className="text-sm font-medium text-gray-900 dark:text-zinc-100">Filters</label>
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
    </div>
  );
}