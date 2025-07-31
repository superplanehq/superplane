import { useState, useEffect } from 'react';
import { EventSourceNodeType } from '@/canvas/types/flow';
import { SuperplaneEventSourceSpec, IntegrationsIntegrationRef } from '@/api-client/types.gen';
import { AccordionItem } from './AccordionItem';
import { Label } from './Label';
import { Field } from './Field';
import { useIntegrations } from '../hooks/useIntegrations';

interface EventSourceEditModeContentProps {
  data: EventSourceNodeType['data'];
  canvasId: string;
  organizationId: string;
  eventSourceType?: string;
  onDataChange?: (data: {
    spec: SuperplaneEventSourceSpec
  }) => void;
}

export function EventSourceEditModeContent({
  data,
  canvasId,
  organizationId,
  eventSourceType = 'webhook',
  onDataChange
}: EventSourceEditModeContentProps) {
  const [openSections, setOpenSections] = useState<string[]>(['general', 'integration', 'webhook']);
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationRef | null>(data.integration);
  const [resourceType, setResourceType] = useState(data.resource?.type || (eventSourceType === 'semaphore' ? 'project' : ''));
  const [resourceName, setResourceName] = useState(data.resource?.name || '');
  const [integrationConfig, setIntegrationConfig] = useState<Record<string, string | boolean>>({});

  // Fetch available integrations
  const { data: canvasIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  // Combine canvas and organization integrations and filter by event source type
  const allIntegrations = [...canvasIntegrations, ...orgIntegrations];
  const availableIntegrations = eventSourceType === 'semaphore'
    ? allIntegrations.filter(int => int.spec?.type === 'semaphore')
    : allIntegrations;

  // Notify parent of data changes
  useEffect(() => {
    if (onDataChange) {
      const spec: SuperplaneEventSourceSpec = {};

      // For semaphore event sources, integration is required
      if (eventSourceType === 'semaphore' && selectedIntegration) {
        spec.integration = selectedIntegration;

        if (resourceType && resourceName) {
          spec.resource = {
            type: resourceType,
            name: resourceName
          };
        }
      }
      // For webhook event sources, no integration by default

      onDataChange({
        spec
      });
    }
  }, [selectedIntegration, resourceType, resourceName, eventSourceType, onDataChange]);

  const handleAccordionToggle = (sectionId: string) => {
    setOpenSections(prev => {
      return prev.includes(sectionId)
        ? prev.filter(id => id !== sectionId)
        : [...prev, sectionId];
    });
  };


  const handleIntegrationChange = (integrationName: string) => {
    const integration = availableIntegrations.find(int => int.metadata?.name === integrationName);
    if (integration) {
      setSelectedIntegration({
        name: integration.metadata?.name,
        domainType: integration.metadata?.domainType
      });

      // Set default resource type based on integration type
      if (integration.spec?.type === 'TYPE_SEMAPHORE') {
        setResourceType('project');
      } else if (integration.spec?.type === 'TYPE_GITHUB') {
        setResourceType('repository');
      }
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
            <Field>
              <Label>Project Name</Label>
              <input
                type="text"
                value={String(integrationConfig.project || '')}
                onChange={(e) => updateIntegrationConfig('project', e.target.value)}
                placeholder="my-semaphore-project"
                className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </Field>
          </div>
        );

      case 'TYPE_GITHUB':
        return (
          <div className="space-y-3">
            <Field>
              <Label>Repository</Label>
              <input
                type="text"
                value={String(integrationConfig.repository || '')}
                onChange={(e) => updateIntegrationConfig('repository', e.target.value)}
                placeholder="owner/repository-name"
                className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </Field>
            <Field>
              <Label>Events</Label>
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
            </Field>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div className="w-full h-full text-left" onClick={(e) => e.stopPropagation()}>
      {/* Accordion Sections */}
      <div className="">

        {/* Configuration Section */}
        {eventSourceType === 'semaphore' && (
          <AccordionItem
            id="integration"
            title={
              <div className="flex items-center justify-between w-full">
                <span>Semaphore Configuration</span>
                <span className="text-xs text-blue-600 font-medium">Required</span>
              </div>
            }
            isOpen={openSections.includes('integration')}
            onToggle={handleAccordionToggle}
          >
            <div className="space-y-3">
              <Field>
                <Label>Select Integration</Label>
                <select
                  value={selectedIntegration?.name || ''}
                  onChange={(e) => handleIntegrationChange(e.target.value)}
                  className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">Select a Semaphore integration...</option>
                  {availableIntegrations.map((integration) => (
                    <option key={integration.metadata?.id} value={integration.metadata?.name}>
                      {integration.metadata?.name}
                    </option>
                  ))}
                </select>
              </Field>

              {availableIntegrations.length === 0 && (
                <div className="text-sm text-zinc-500 bg-zinc-50 dark:bg-zinc-800 p-3 rounded-md">
                  No Semaphore integrations available. Create one first in the canvas settings.
                </div>
              )}

              {selectedIntegration && (
                <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                  <Label className="text-sm font-medium mb-2 block">Project Configuration</Label>

                  <Field>
                    <Label>Resource Type</Label>
                    <select
                      value={resourceType}
                      onChange={(e) => setResourceType(e.target.value)}
                      className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="">Select resource type...</option>
                      <option value="project">Project</option>
                    </select>
                  </Field>

                  <Field>
                    <Label>Resource Name</Label>
                    <input
                      type="text"
                      value={resourceName}
                      onChange={(e) => setResourceName(e.target.value)}
                      placeholder="my-semaphore-project"
                      className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </Field>

                  {/* Integration-specific configuration fields */}
                  {renderIntegrationSpecificFields()}
                </div>
              )}
            </div>
          </AccordionItem>
        )}

        {/* Webhook Configuration Section */}
        {eventSourceType === 'webhook' && (
          <AccordionItem
            id="webhook"
            title="Webhook Configuration"
            isOpen={openSections.includes('webhook')}
            onToggle={handleAccordionToggle}
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
          </AccordionItem>
        )}
      </div>
    </div>
  );
}