import { useEffect, useState } from 'react';
import type { SuperplaneEventSource } from '@/api-client/types.gen';
import { useManifestByType } from '@/hooks/useManifests';
import { DynamicForm } from '@/components/DynamicForm';

interface EventSourceFormSectionProps {
  eventSource: SuperplaneEventSource;
  availableIntegrations: Array<{ metadata?: { id?: string; name?: string }; spec?: { type?: string } }>;
  validationErrors: Record<string, string>;
  fieldErrors: Record<string, string>;
  onEventSourceChange: (updates: Partial<SuperplaneEventSource>) => void;
  organizationId?: string;
  canvasId?: string;
}

export function EventSourceFormSection({
  eventSource,
  availableIntegrations,
  validationErrors,
  fieldErrors,
  onEventSourceChange,
  organizationId,
  canvasId,
}: EventSourceFormSectionProps) {
  const { manifest, isLoading } = useManifestByType('event_source', eventSource.spec?.type || '');
  const [formValues, setFormValues] = useState<Record<string, any>>({});

  // Initialize form values from event source spec and resource
  useEffect(() => {
    if (manifest && eventSource.spec) {
      const values: Record<string, any> = { ...(eventSource.spec as Record<string, any> || {}) };
      // If there's a resource, add it to form values
      if (eventSource.spec.resource?.name) {
        values.resource = eventSource.spec.resource.name;
      }
      setFormValues(values);
    }
  }, [manifest, eventSource.spec]);

  // Update event source when form values change
  const handleFormChange = (newValues: Record<string, any>) => {
    setFormValues(newValues);

    // Extract resource from form values if present
    const { resource, ...specValues } = newValues;

    const updates: Partial<SuperplaneEventSource> = {
      spec: specValues,
    };

    // Handle resource field specially - it goes to spec.resource
    if (resource !== undefined) {
      // Determine resource type from manifest
      const resourceField = manifest?.fields?.find(f => f.name === 'resource');
      const resourceType = resourceField?.resourceType || 'repository';

      updates.spec = {
        ...specValues,
        resource: {
          type: resourceType,
          name: resource,
        },
      };
    }

    onEventSourceChange(updates);
  };

  if (!eventSource.spec?.type) {
    return (
      <div className="text-sm text-zinc-500">
        Select an event source type to configure
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-zinc-500">Loading event source configuration...</div>;
  }

  if (!manifest) {
    return (
      <div className="text-sm text-red-500">
        Unable to load configuration for event source type: {eventSource.spec?.type}
      </div>
    );
  }

  // Check if this event source requires an integration
  const integrationsForType = availableIntegrations.filter(
    (int) => int.spec?.type === manifest.integrationType
  );
  const requiresIntegration = !!manifest.integrationType;

  return (
    <div className="space-y-4">
      {requiresIntegration && (
        <div className="space-y-1">
          <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300">
            Integration
            {validationErrors.integration && <span className="text-red-500 ml-1">*</span>}
          </label>
          <select
            value={eventSource.spec?.integration?.name || ''}
            onChange={(e) => {
              onEventSourceChange({
                spec: {
                  ...eventSource.spec,
                  integration: {
                    name: e.target.value,
                  },
                },
              });
            }}
            className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${
              validationErrors.integration
                ? 'border-red-300 dark:border-red-600 focus:ring-red-500'
                : 'border-zinc-300 dark:border-zinc-600 focus:ring-blue-500'
            }`}
          >
            <option value="">Select an integration...</option>
            {integrationsForType.map((integration) => (
              <option
                key={integration.metadata?.id}
                value={integration.metadata?.name}
              >
                {integration.metadata?.name}
              </option>
            ))}
          </select>
          {integrationsForType.length === 0 && (
            <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
              No {manifest.displayName} integrations available. Create one in canvas settings.
            </div>
          )}
          {validationErrors.integration && (
            <p className="text-sm text-red-500">{validationErrors.integration}</p>
          )}
        </div>
      )}

      <DynamicForm
        manifest={manifest}
        value={formValues}
        onChange={handleFormChange}
        disabled={false}
        errors={fieldErrors}
        context={{
          integrationName: eventSource.spec?.integration?.name,
          organizationId,
          canvasId,
        }}
      />
    </div>
  );
}
