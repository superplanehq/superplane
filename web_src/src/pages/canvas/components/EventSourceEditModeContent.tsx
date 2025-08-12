import { useState, useEffect, useCallback } from 'react';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { SuperplaneEventSourceSpec, IntegrationsIntegrationRef } from '@/api-client/types.gen';
import { Link } from '@/components/Link/link';
import { useIntegrations } from '../hooks/useIntegrations';
import { useEditModeState } from '../hooks/useEditModeState';
import { EditableAccordionSection } from './shared/EditableAccordionSection';
import { ValidationField } from './shared/ValidationField';

interface EventSourceEditModeContentProps {
  data: EventSourceNodeType['data'];
  canvasId: string;
  organizationId: string;
  eventSourceType?: string;
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
  onDataChange,
  onDelete,
  apiError,
  shouldValidate = false,
  onValidationResult
}: EventSourceEditModeContentProps) {
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationRef | null>(data.integration);
  const [resourceType, setResourceType] = useState(data.resource?.type);
  const [resourceName, setResourceName] = useState(data.resource?.name || '');
  const [integrationConfig, setIntegrationConfig] = useState<Record<string, string | boolean>>({});
  const [apiValidationErrors, setApiValidationErrors] = useState<Record<string, string>>({});

  const parseApiError = (errorMessage: string) => {
    const errors: Record<string, string> = {};

    if (errorMessage.includes('not found')) {
      if (errorMessage.includes('project')) {
        errors.resourceName = 'Project not found. Please check the project name and ensure it exists in Semaphore.';
      } else if (errorMessage.includes('repository')) {
        errors.resourceName = 'Repository not found. Please check the repository name and ensure it exists.';
      }
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

  const revertSection = (section: string) => {
    switch (section) {
      case 'integration':
        setSelectedIntegration(originalData.spec.integration || null);
        setResourceType(originalData.spec.resource?.type || (eventSourceType === 'semaphore' ? 'project' : ''));
        setResourceName(originalData.spec.resource?.name || '');
        setIntegrationConfig({});
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

  const updateIntegrationConfig = (key: string, value: string | boolean) => {
    setIntegrationConfig(prev => ({
      ...prev,
      [key]: value
    }));
  };

  const renderIntegrationSpecificFields = () => {
    if (!selectedIntegration) return null;

    const integration = availableIntegrations.find(
      int => int.metadata?.name === selectedIntegration.name
    );

    if (!integration) return null;

    // Render fields based on integration type
    switch (integration.spec?.type) {
      case 'TYPE_SEMAPHORE':
        return (
          <div className="space-y-3">
            <ValidationField label="Project Name">
              <input
                type="text"
                value={String(integrationConfig.project || '')}
                onChange={(e) => updateIntegrationConfig('project', e.target.value)}
                placeholder="my-semaphore-project"
                className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </ValidationField>
          </div>
        );

      case 'TYPE_GITHUB':
        return (
          <div className="space-y-3">
            <ValidationField label="Repository">
              <input
                type="text"
                value={String(integrationConfig.repository || '')}
                onChange={(e) => updateIntegrationConfig('repository', e.target.value)}
                placeholder="owner/repository-name"
                className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </ValidationField>
            <ValidationField label="Events">
              <select
                value={String(integrationConfig.events || 'push')}
                onChange={(e) => updateIntegrationConfig('events', e.target.value)}
                className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="push">Push</option>
                <option value="pull_request">Pull Request</option>
                <option value="issues">Issues</option>
                <option value="release">Release</option>
              </select>
            </ValidationField>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      <div className="">
        {/* Configuration Section */}
        {eventSourceType === 'semaphore' && (
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

              {availableIntegrations.length === 0 && (
                <div className="text-sm text-zinc-500 dark:text-zinc-400 bg-zinc-50 dark:bg-zinc-800 p-3 rounded-md">
                  No Semaphore integrations available. Create one first in the &nbsp;
                  <Link className="text-blue-600 hover:underline" href={`/organization/${organizationId}/canvas/${canvasId}#integrations`}>canvas settings</Link>.
                </div>
              )}

              {(selectedIntegration || combinedErrors.resourceName) && (
                <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                  <ValidationField
                    label="Project Name"
                    error={combinedErrors.resourceName}
                    required={true}
                  >
                    <input
                      type="text"
                      value={resourceName}
                      onChange={(e) => {
                        setResourceName(e.target.value);
                        if (e.target.value.trim() !== '') {
                          setValidationErrors(prev => ({
                            ...prev,
                            resourceName: ''
                          }));
                        }
                      }}
                      placeholder="my-semaphore-project"
                      className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors.resourceName
                        ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                    />
                  </ValidationField>

                  {/* Integration-specific configuration fields */}
                  {renderIntegrationSpecificFields()}
                </div>
              )}
            </div>
          </EditableAccordionSection>
        )}

        {eventSourceType === 'github' && (
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

              {availableIntegrations.length === 0 && (
                <div className="text-sm text-zinc-500 bg-zinc-50 dark:bg-zinc-800 p-3 rounded-md">
                  No GitHub integrations available. Create one first in the &nbsp;
                  <Link className="text-blue-600 hover:underline" href={`/organization/${organizationId}/canvas/${canvasId}#integrations`}>canvas settings</Link>.
                </div>
              )}

              {(selectedIntegration || combinedErrors.resourceName) && (
                <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                  <ValidationField
                    label="Repository Name"
                    error={combinedErrors.resourceName}
                    required={true}
                  >
                    <input
                      type="text"
                      value={resourceName}
                      onChange={(e) => {
                        setResourceName(e.target.value);
                        if (e.target.value.trim() !== '') {
                          setValidationErrors(prev => ({
                            ...prev,
                            resourceName: ''
                          }));
                        }
                      }}
                      placeholder="my-repository"
                      className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${combinedErrors.resourceName
                        ? 'border-red-500 dark:border-red-400 focus:ring-red-500'
                        : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
                        }`}
                    />
                  </ValidationField>

                  {/* Integration-specific configuration fields */}
                  {renderIntegrationSpecificFields()}
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
            <div className="space-y-3">
              {!Number.isNaN(Number(data.id)) ? (
                <div className="text-sm text-amber-600 bg-amber-50 dark:bg-amber-900/20 p-3 rounded-md">
                  Save this event source to generate the webhook endpoint and signing key.
                </div>
              ) : (
                <div className="text-sm text-amber-600 bg-amber-50 dark:bg-amber-900/20 p-3 rounded-md">
                  This event source has been saved. Register the webhook at:
                  <input
                    type="text"
                    value={`https://superplane.io/api/v1/sources/${data.id}/${data.name}`}
                    readOnly
                    className="w-full px-3 py-2 bg-zinc-50 dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              )}
            </div>
          </EditableAccordionSection>
        )}
      </div>
    </div>
  );
}