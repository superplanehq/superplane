import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, Settings, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { getApiErrorMessage } from "@/lib/errors";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { useIntegration, useUpdateIntegration, useAvailableIntegrations } from "@/hooks/useIntegrations";
import { showErrorToast } from "@/lib/toast";
import { useIntegrationConfigureOpen } from "@/lib/analytics";
import type { ConfigurationField } from "@/api-client";

interface ConfigureIntegrationDialogProps {
  integrationId: string | null;
  organizationId: string;
  onClose: () => void;
}

/**
 * Standalone configure integration dialog.
 * Same UI as the one in ComponentSidebar but can be mounted anywhere.
 */
export function ConfigureIntegrationDialog({
  integrationId,
  organizationId,
  onClose,
}: ConfigureIntegrationDialogProps) {
  const { data: integration, isLoading } = useIntegration(organizationId, integrationId ?? "");
  const updateMutation = useUpdateIntegration(organizationId, integrationId ?? "");
  const { data: availableIntegrations } = useAvailableIntegrations();

  const definition = useMemo(
    () =>
      integration?.metadata?.integrationName
        ? availableIntegrations?.find((d) => d.name === integration.metadata?.integrationName)
        : undefined,
    [availableIntegrations, integration?.metadata?.integrationName],
  );

  const [name, setName] = useState("");
  const [config, setConfig] = useState<Record<string, unknown>>({});

  useIntegrationConfigureOpen(integration ?? undefined, integrationId, "node_configuration", organizationId);

  useEffect(() => {
    if (integration?.spec?.configuration) {
      setConfig(integration.spec.configuration as Record<string, unknown>);
    }
  }, [integration?.spec?.configuration]);

  useEffect(() => {
    setName(integration?.metadata?.name || integration?.metadata?.integrationName || "");
  }, [integration?.metadata?.name, integration?.metadata?.integrationName]);

  const handleSubmit = useCallback(async () => {
    if (!integrationId) return;
    const trimmed = name.trim();
    if (!trimmed) {
      showErrorToast("Integration name is required");
      return;
    }
    try {
      await updateMutation.mutateAsync({ name: trimmed, configuration: config });
      onClose();
    } catch {
      showErrorToast("Failed to update integration");
    }
  }, [integrationId, name, config, updateMutation, onClose]);

  const handleBrowserAction = useCallback(() => {
    const browserAction = integration?.status?.browserAction;
    if (!browserAction) return;

    const { url, method, formFields } = browserAction;
    if (method?.toUpperCase() === "POST" && formFields) {
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
      form.target = "_blank";
      form.style.display = "none";
      Object.entries(formFields).forEach(([key, value]) => {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = key;
        input.value = String(value);
        form.appendChild(input);
      });
      document.body.appendChild(form);
      form.submit();
      document.body.removeChild(form);
      return;
    }

    if (url) window.open(url, "_blank");
  }, [integration?.status?.browserAction]);

  return (
    <Dialog open={!!integrationId} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto" showCloseButton={!updateMutation.isPending}>
        {isLoading ? (
          <div className="flex justify-center items-center py-12">
            <Loader2 className="w-8 h-8 animate-spin text-gray-500" />
          </div>
        ) : integrationId && integration ? (
          <>
            <DialogHeader>
              <div className="flex items-center gap-3">
                <IntegrationIcon
                  integrationName={integration.metadata?.integrationName}
                  iconSlug={definition?.icon}
                  className="h-6 w-6 text-gray-500"
                />
                <div className="flex items-center gap-2">
                  <DialogTitle>
                    Configure{" "}
                    {getIntegrationTypeDisplayName(undefined, integration.metadata?.integrationName) ||
                      integration.metadata?.integrationName}
                  </DialogTitle>
                  <a
                    href={
                      integration.metadata?.id
                        ? `/${organizationId}/settings/integrations/${integration.metadata.id}`
                        : `/${organizationId}/settings/integrations`
                    }
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex h-4 w-4 items-center justify-center text-gray-500 hover:text-gray-800 transition-colors"
                    aria-label="Open integration settings"
                  >
                    <Settings className="h-4 w-4" />
                  </a>
                </div>
              </div>
            </DialogHeader>

            {integration.status?.state === "error" && integration.status?.stateDescription && (
              <div className="flex items-start gap-2 text-sm text-red-700">
                <TriangleAlert className="h-4 w-4 mt-0.5 flex-shrink-0" />
                <p>{integration.status.stateDescription}</p>
              </div>
            )}

            {integration?.status?.browserAction && (
              <IntegrationInstructions
                description={integration.status.browserAction.description}
                onContinue={integration.status.browserAction.url ? handleBrowserAction : undefined}
                className="mb-6"
              />
            )}

            <form
              onSubmit={(e) => {
                e.preventDefault();
                void handleSubmit();
              }}
              className="space-y-4"
            >
              <div>
                <Label className="text-gray-800 mb-2">
                  Integration Name
                  <span className="text-gray-800 ml-1">*</span>
                </Label>
                <Input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., my-app-integration"
                />
                <p className="text-xs text-gray-500 mt-2">A unique name for this integration</p>
              </div>

              {definition?.configuration && definition.configuration.length > 0 ? (
                definition.configuration.map((field: ConfigurationField) => {
                  if (!field.name) return null;
                  return (
                    <ConfigurationFieldRenderer
                      key={field.name}
                      field={field}
                      value={config[field.name]}
                      onChange={(value) => setConfig((prev) => ({ ...prev, [field.name || ""]: value }))}
                      allValues={config}
                      domainId={organizationId}
                      domainType="DOMAIN_TYPE_ORGANIZATION"
                      organizationId={organizationId}
                      integrationId={integration.metadata?.id}
                    />
                  );
                })
              ) : (
                <p className="text-sm text-gray-500">No configuration fields available.</p>
              )}

              <DialogFooter className="gap-2 sm:justify-start pt-4">
                <LoadingButton
                  type="submit"
                  color="blue"
                  disabled={!name.trim()}
                  loading={updateMutation.isPending}
                  loadingText="Saving..."
                  className="flex items-center gap-2"
                >
                  Save
                </LoadingButton>
                <Button type="button" variant="outline" onClick={onClose} disabled={updateMutation.isPending}>
                  Cancel
                </Button>
              </DialogFooter>

              {updateMutation.isError && (
                <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-md">
                  <p className="text-sm text-red-800">
                    Failed to update integration: {getApiErrorMessage(updateMutation.error)}
                  </p>
                </div>
              )}
            </form>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
