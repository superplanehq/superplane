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
  onDataChange?: (data: {
    name: string;
    spec: SuperplaneEventSourceSpec
  }) => void;
}

export function EventSourceEditModeContent({
  data,
  canvasId,
  organizationId,
  onDataChange
}: EventSourceEditModeContentProps) {
  const [openSections, setOpenSections] = useState<string[]>(['general']);
  const [name, setName] = useState(data.name || '');
  const [hasIntegration, setHasIntegration] = useState(data.integration !== null);
  const [selectedIntegration, setSelectedIntegration] = useState<IntegrationsIntegrationRef | null>(data.integration);
  const [resourceType, setResourceType] = useState(data.resource?.type || '');
  const [resourceName, setResourceName] = useState(data.resource?.name || '');
  const [integrationConfig, setIntegrationConfig] = useState<Record<string, string | boolean>>({});

  // Fetch available integrations
  const { data: canvasIntegrations = [] } = useIntegrations(canvasId, "DOMAIN_TYPE_CANVAS");
  const { data: orgIntegrations = [] } = useIntegrations(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  // Combine canvas and organization integrations
  const availableIntegrations = [...canvasIntegrations, ...orgIntegrations];

  // Notify parent of data changes
  useEffect(() => {
    if (onDataChange) {
      const spec: SuperplaneEventSourceSpec = {};

      if (hasIntegration && selectedIntegration) {
        spec.integration = selectedIntegration;

        if (resourceType && resourceName) {
          spec.resource = {
            type: resourceType,
            name: resourceName
          };
        }
      }

      onDataChange({
        name,
        spec
      });
    }
  }, [name, hasIntegration, selectedIntegration, resourceType, resourceName, onDataChange]);

  const handleAccordionToggle = (sectionId: string) => {
    setOpenSections(prev => {
      return prev.includes(sectionId)
        ? prev.filter(id => id !== sectionId)
        : [...prev, sectionId];
    });
  };

  const handleIntegrationToggle = (enabled: boolean) => {
    setHasIntegration(enabled);
    if (!enabled) {
      setSelectedIntegration(null);
      setResourceType('');
      setResourceName('');
      setIntegrationConfig({});
    }
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

        {/* General Section */}
        <AccordionItem
          id="general"
          title="General Configuration"
          isOpen={openSections.includes('general')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-3">
            <Field>
              <Label>Event Source Name</Label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="github-webhook"
                className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </Field>
          </div>
        </AccordionItem>

        {/* Integration Section */}
        <AccordionItem
          id="integration"
          title={
            <div className="flex items-center justify-between w-full">
              <span>Integration</span>
            </div>
          }
          isOpen={openSections.includes('integration')}
          onToggle={handleAccordionToggle}
        >
          <div className="space-y-3">
            <Field>
              <div className="flex items-center gap-4 mb-3">
                <Label>Use Integration</Label>
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2">
                    <input
                      type="radio"
                      name="hasIntegration"
                      checked={!hasIntegration}
                      onChange={() => handleIntegrationToggle(false)}
                      className="w-4 h-4"
                    />
                    <span className="text-sm">No</span>
                  </label>
                  <label className="flex items-center gap-2">
                    <input
                      type="radio"
                      name="hasIntegration"
                      checked={hasIntegration}
                      onChange={() => handleIntegrationToggle(true)}
                      className="w-4 h-4"
                    />
                    <span className="text-sm">Yes</span>
                  </label>
                </div>
              </div>
            </Field>

            {hasIntegration && (
              <>
                <Field>
                  <Label>Select Integration</Label>
                  <select
                    value={selectedIntegration?.name || ''}
                    onChange={(e) => handleIntegrationChange(e.target.value)}
                    className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Select an integration...</option>
                    {availableIntegrations.map((integration) => (
                      <option key={integration.metadata?.id} value={integration.metadata?.name}>
                        {integration.metadata?.name} ({integration.spec?.type?.replace('TYPE_', '')})
                      </option>
                    ))}
                  </select>
                </Field>

                {availableIntegrations.length === 0 && (
                  <div className="text-sm text-zinc-500 bg-zinc-50 dark:bg-zinc-800 p-3 rounded-md">
                    No integrations available. Create an integration first in the canvas settings.
                  </div>
                )}

                {selectedIntegration && (
                  <div className="border-t border-zinc-200 dark:border-zinc-700 pt-3">
                    <Label className="text-sm font-medium mb-2 block">Resource Configuration</Label>

                    <Field>
                      <Label>Resource Type</Label>
                      <input
                        type="text"
                        value={resourceType}
                        onChange={(e) => setResourceType(e.target.value)}
                        placeholder="project, repository, etc."
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>

                    <Field>
                      <Label>Resource Name</Label>
                      <input
                        type="text"
                        value={resourceName}
                        onChange={(e) => setResourceName(e.target.value)}
                        placeholder="Resource identifier"
                        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </Field>

                    {/* Integration-specific configuration fields */}
                    {renderIntegrationSpecificFields()}
                  </div>
                )}
              </>
            )}

            {!hasIntegration && (
              <div className="text-sm text-zinc-500 bg-zinc-50 dark:bg-zinc-800 p-3 rounded-md">
                This event source will work as a simple webhook endpoint without any external integration.
              </div>
            )}
          </div>
        </AccordionItem>
      </div>
    </div>
  );
}