import { useEffect, useState } from 'react';
import type { SuperplaneExecutor } from '@/api-client/types.gen';
import { useManifestByType } from '@/hooks/useManifests';
import { DynamicForm } from '@/components/DynamicForm';

interface ExecutorFormSectionProps {
  executor: SuperplaneExecutor;
  dryRun: boolean;
  availableIntegrations: Array<{ metadata?: { id?: string; name?: string }; spec?: { type?: string } }>;
  validationErrors: Record<string, string>;
  fieldErrors: Record<string, string>;
  onExecutorChange: (updates: Partial<SuperplaneExecutor>) => void;
  onFieldErrorChange?: (field: string, error: string) => void;
  organizationId?: string;
  canvasId?: string;
}

export function ExecutorFormSection({
  executor,
  dryRun,
  availableIntegrations,
  validationErrors,
  fieldErrors,
  onExecutorChange,
  organizationId,
  canvasId,
}: ExecutorFormSectionProps) {
  const { manifest, isLoading } = useManifestByType('executor', executor.type);
  const [formValues, setFormValues] = useState<Record<string, any>>({});

  // Initialize form values from executor spec and resource
  useEffect(() => {
    if (manifest) {
      const values: Record<string, any> = { ...(executor.spec as Record<string, any> || {}) };

      // Add resource name to form values if it exists
      if (executor.resource?.name) {
        values.resource = executor.resource.name;
      }

      setFormValues(values);
    }
  }, [manifest, executor.spec, executor.resource]);

  // Update executor when form values change
  const handleFormChange = (newValues: Record<string, any>) => {
    setFormValues(newValues);

    // Extract resource from form values if present
    const { resource, ...specValues } = newValues;

    const updates: Partial<SuperplaneExecutor> = {
      spec: specValues,
    };

    // Handle resource field specially - it goes to executor.resource, not executor.spec
    if (resource !== undefined) {
      // Determine resource type from manifest
      const resourceField = manifest?.fields?.find(f => f.name === 'resource');
      const resourceType = resourceField?.resourceType || 'repository';

      updates.resource = {
        type: resourceType,
        name: resource,
      };
    }

    onExecutorChange(updates);
  };

  if (dryRun) {
    return (
      <div className="text-sm text-zinc-500">
        Dry run mode - no executor configuration required
      </div>
    );
  }

  if (!executor.type) {
    return (
      <div className="text-sm text-zinc-500">
        Select an executor type to configure
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-zinc-500">Loading executor configuration...</div>;
  }

  if (!manifest) {
    return (
      <div className="text-sm text-red-500">
        Unable to load configuration for executor type: {executor.type}
      </div>
    );
  }

  // Check if this executor requires an integration
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
            {validationErrors.executorIntegration && <span className="text-red-500 ml-1">*</span>}
          </label>
          <select
            value={executor.integration?.name || ''}
            onChange={(e) => {
              onExecutorChange({
                integration: {
                  name: e.target.value,
                },
              });
            }}
            className={`w-full px-3 py-2 border rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 ${
              validationErrors.executorIntegration
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
          {validationErrors.executorIntegration && (
            <p className="text-sm text-red-500">{validationErrors.executorIntegration}</p>
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
          integrationName: executor.integration?.name,
          organizationId,
          canvasId,
        }}
      />
    </div>
  );
}
