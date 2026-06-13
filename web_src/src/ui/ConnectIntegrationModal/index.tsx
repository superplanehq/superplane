import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertTitle, AlertDescription } from "@/ui/alert";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useCreateIntegration } from "@/hooks/useIntegrations";
import { Icon } from "@/ui/icons";
import { Loader2 } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

interface ConnectIntegrationModalProps {
  integrationName: string;
  organizationId: string;
  onClose: () => void;
  onConnected: () => void;
}

export function ConnectIntegrationModal({
  integrationName,
  organizationId,
  onClose,
  onConnected,
}: ConnectIntegrationModalProps) {
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const createMutation = useCreateIntegration(organizationId, "install_wizard");

  const definition = useMemo(
    () => availableIntegrations.find((d) => d.name === integrationName),
    [availableIntegrations, integrationName],
  );

  const [instanceName, setInstanceName] = useState(
    integrationName.charAt(0).toUpperCase() + integrationName.slice(1),
  );
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});

  const handleConnect = useCallback(async () => {
    if (!definition?.name) return;

    try {
      await createMutation.mutateAsync({
        integrationName: definition.name,
        name: instanceName,
        configuration,
      });
      onConnected();
    } catch {
      // Error shown via mutation state
    }
  }, [definition, instanceName, configuration, createMutation, onConnected]);

  if (!definition) return null;

  const label = definition.label || definition.name || integrationName;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <IntegrationIcon
                integrationName={integrationName}
                iconSlug={definition.icon}
                className="w-6 h-6 text-gray-500 dark:text-gray-400"
              />
              <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">Connect {label}</h3>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
              disabled={createMutation.isPending}
            >
              <Icon name="x" size="sm" />
            </button>
          </div>

          {definition.instructions && (
            <div className="mb-4 text-sm text-slate-600 prose prose-sm max-w-none">
              <div dangerouslySetInnerHTML={{ __html: definition.instructions }} />
            </div>
          )}

          <div className="space-y-4">
            <div>
              <Label className="text-gray-800 dark:text-gray-100 mb-2">
                Integration Name
                <span className="text-gray-800 ml-1">*</span>
              </Label>
              <Input
                type="text"
                value={instanceName}
                onChange={(e) => setInstanceName(e.target.value)}
                placeholder="e.g., my-integration"
                required
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">A unique name for this integration</p>
            </div>

            {definition.configuration &&
              definition.configuration.length > 0 &&
              definition.configuration
                .filter((field) => Boolean(field.name))
                .map((field) => (
                  <ConfigurationFieldRenderer
                    key={field.name!}
                    field={field}
                    value={configuration[field.name!]}
                    onChange={(value) => setConfiguration((prev) => ({ ...prev, [field.name!]: value }))}
                    allValues={configuration}
                    domainId={organizationId}
                    domainType="DOMAIN_TYPE_ORGANIZATION"
                    organizationId={organizationId}
                  />
                ))}
          </div>

          <div className="flex justify-start gap-3 mt-6">
            <Button onClick={handleConnect} disabled={createMutation.isPending || !instanceName?.trim()}>
              {createMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin mr-1" />
                  Connecting...
                </>
              ) : (
                "Connect"
              )}
            </Button>
            <Button variant="outline" onClick={onClose} disabled={createMutation.isPending}>
              Cancel
            </Button>
          </div>

          {createMutation.isError && (
            <Alert variant="destructive" className="mt-4">
              <AlertTitle>Unable to connect integration</AlertTitle>
              <AlertDescription>
                {createMutation.error instanceof Error ? createMutation.error.message : "An error occurred"}
              </AlertDescription>
            </Alert>
          )}
        </div>
      </div>
    </div>
  );
}
