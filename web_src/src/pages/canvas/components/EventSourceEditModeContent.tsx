import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { SuperplaneEventSourceSpec, IntegrationsIntegrationRef } from '@/api-client/types.gen';
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
  const [apiValidationErrors, setApiValidationErrors] = useState<Record<string, string>>({});
  const [isKeyRevealed, setIsKeyRevealed] = useState(false);
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

    if (setErrors) {
      setErrors(errors);
    }

    return Object.keys(errors).length === 0;
  };

  useEffect(() => {
    if (apiError) {
      const parsedErrors = parseApiError(apiError);
      setApiValidationErrors(parsedErrors);
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
        resource: data.resource
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
  }, [shouldValidate, selectedIntegration, resourceName, eventSourceType, onValidationResult]);

  useEffect(() => {
    setOpenSections(['general', 'integration', 'webhook']);
  }, [setOpenSections]);

  useEffect(() => {
    syncWithIncomingData(
      {
        spec: {
          integration: data.integration,
          resource: data.resource
        } as SuperplaneEventSourceSpec
      },
      (incomingData) => {
        setSelectedIntegration(incomingData.spec.integration || null);
        setResourceType(incomingData.spec.resource?.type || (eventSourceType === 'semaphore' ? 'project' : ''));
        setResourceName(incomingData.spec.resource?.name || '');
      }
    );
  }, [data, eventSourceType, syncWithIncomingData]);

  const { data: canvasIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

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

      handleDataChange({
        spec
      });
    }
  }, [selectedIntegration, resourceType, resourceName, eventSourceType, onDataChange, handleDataChange]);

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
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
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
                          value={`https://superplane.sxmoon.com/api/v1/sources/${data.id}`}
                          readOnly
                          className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-l-md bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm focus:outline-none"
                        />
                        <Button
                          outline
                          onClick={() => navigator.clipboard.writeText(`https://superplane.sxmoon.com/api/v1/sources/${data.id}`)}
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