import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2, Settings } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { useCreateIntegration, useUpdateIntegration } from "@/hooks/useIntegrations";
import type { ConfigurationField, IntegrationsIntegrationDefinition, OrganizationsBrowserAction } from "@/api-client";

export interface IntegrationCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  integrationDefinition: IntegrationsIntegrationDefinition | null | undefined;
  organizationId: string;
  defaultName?: string;
  integrationHomeHref?: string;
  onCreated?: (integrationId: string) => void;
}

export function IntegrationCreateDialog({
  open,
  onOpenChange,
  integrationDefinition,
  organizationId,
  defaultName = "",
  integrationHomeHref,
  onCreated,
}: IntegrationCreateDialogProps) {
  const [integrationName, setIntegrationName] = useState(defaultName);
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [createIntegrationBrowserAction, setCreateIntegrationBrowserAction] = useState<
    OrganizationsBrowserAction | undefined
  >(undefined);
  const [createdIncidentIntegration, setCreatedIncidentIntegration] = useState<{
    id: string;
    webhookUrl: string;
    config: Record<string, unknown>;
  } | null>(null);

  const createIntegrationMutation = useCreateIntegration(organizationId);
  const updateIntegrationMutation = useUpdateIntegration(organizationId, createdIncidentIntegration?.id ?? "");

  const selectedInstructions = integrationDefinition?.instructions?.trim();
  const configurationFields = integrationDefinition?.configuration ?? [];

  useEffect(() => {
    if (open) {
      setIntegrationName(defaultName);
      setConfiguration({});
      setCreateIntegrationBrowserAction(undefined);
      setCreatedIncidentIntegration(null);
    }
  }, [open, defaultName]);

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setIntegrationName("");
        setConfiguration({});
        setCreateIntegrationBrowserAction(undefined);
        setCreatedIncidentIntegration(null);
        createIntegrationMutation.reset();
      }
      onOpenChange(next);
    },
    [onOpenChange, createIntegrationMutation],
  );

  const handleClose = useCallback(() => {
    handleOpenChange(false);
  }, [handleOpenChange]);

  const handleSubmit = useCallback(async () => {
    if (!integrationDefinition?.name || !organizationId) return;
    const nextName = integrationName.trim();
    if (!nextName) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      const result = await createIntegrationMutation.mutateAsync({
        integrationName: integrationDefinition.name,
        name: nextName,
        configuration,
      });

      const integration = result.data?.integration;
      const browserAction = integration?.status?.browserAction;
      const webhookUrl =
        integration?.spec?.integrationName === "incident" &&
        integration?.status?.metadata &&
        typeof integration.status.metadata === "object" &&
        "webhookUrl" in integration.status.metadata
          ? (integration.status.metadata as { webhookUrl?: string }).webhookUrl
          : undefined;

      if (browserAction) {
        setCreateIntegrationBrowserAction(browserAction);
        return;
      }
      if (integration?.metadata?.id && webhookUrl && integrationDefinition.name === "incident") {
        setCreatedIncidentIntegration({
          id: integration.metadata.id,
          webhookUrl,
          config: { ...configuration },
        });
        return;
      }
      handleClose();
      if (integration?.metadata?.id) {
        onCreated?.(integration.metadata.id);
      }
    } catch (error) {
      showErrorToast(`Failed to create integration: ${getApiErrorMessage(error)}`);
    }
  }, [
    integrationDefinition?.name,
    organizationId,
    integrationName,
    configuration,
    createIntegrationMutation,
    handleClose,
    onCreated,
  ]);

  const handleCompleteIncidentSetup = useCallback(async () => {
    if (!createdIncidentIntegration) return;

    try {
      await updateIntegrationMutation.mutateAsync({
        configuration: { ...createdIncidentIntegration.config, ...configuration },
      });
      handleClose();
      onCreated?.(createdIncidentIntegration.id);
    } catch {
      showErrorToast("Failed to complete setup");
    }
  }, [createdIncidentIntegration, configuration, updateIntegrationMutation, handleClose, onCreated]);

  const handleBrowserActionContinue = useCallback(() => {
    if (!createIntegrationBrowserAction) return;
    const { url, method, formFields } = createIntegrationBrowserAction;
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
    } else if (url) {
      window.open(url, "_blank");
    }
  }, [createIntegrationBrowserAction]);

  if (!integrationDefinition) return null;

  const displayName =
    getIntegrationTypeDisplayName(undefined, integrationDefinition.name) || integrationDefinition.name;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="sm:max-w-2xl max-h-[80vh] overflow-y-auto"
        showCloseButton={!createIntegrationMutation.isPending}
      >
        <DialogHeader>
          <div className="flex items-center gap-3">
            <IntegrationIcon
              integrationName={integrationDefinition.name}
              iconSlug={integrationDefinition.icon}
              className="h-6 w-6 text-gray-500 dark:text-gray-400"
            />
            <div className="flex items-center gap-2">
              <DialogTitle>
                {createdIncidentIntegration ? "Complete webhook setup" : `Configure ${displayName}`}
              </DialogTitle>
              {integrationHomeHref && (
                <a
                  href={integrationHomeHref}
                  className="inline-flex h-4 w-4 items-center justify-center text-gray-500 hover:text-gray-800 transition-colors"
                  aria-label="Open integration settings"
                >
                  <Settings className="h-4 w-4" />
                </a>
              )}
            </div>
          </div>
          {!createdIncidentIntegration && (createIntegrationBrowserAction?.description || selectedInstructions) && (
            <IntegrationInstructions
              description={(createIntegrationBrowserAction?.description || selectedInstructions) ?? ""}
              onContinue={createIntegrationBrowserAction?.url ? handleBrowserActionContinue : undefined}
              className="mt-2"
            />
          )}
        </DialogHeader>

        <div className="space-y-4">
          {createdIncidentIntegration ? (
            <>
              <p className="text-sm text-gray-800 dark:text-gray-200">
                Copy the webhook URL below, add it in incident.io Settings → Webhooks, subscribe to Public incident
                created (v2) and Public incident updated (v2), then paste the signing secret.
              </p>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Webhook URL</Label>
                <div className="flex gap-2">
                  <Input
                    type="text"
                    readOnly
                    value={createdIncidentIntegration.webhookUrl}
                    className="font-mono text-sm"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      void navigator.clipboard.writeText(createdIncidentIntegration.webhookUrl);
                      showSuccessToast("Copied to clipboard");
                    }}
                  >
                    Copy
                  </Button>
                </div>
              </div>
              {configurationFields
                ?.filter((f: ConfigurationField) => f.name === "signingSecret")
                .map((field) => (
                  <ConfigurationFieldRenderer
                    key={field.name}
                    field={field}
                    value={configuration[field.name!]}
                    onChange={(value) =>
                      setConfiguration((prev) => ({
                        ...prev,
                        [field.name!]: value,
                      }))
                    }
                    allValues={configuration}
                    domainId={organizationId}
                    domainType="DOMAIN_TYPE_ORGANIZATION"
                    organizationId={organizationId}
                  />
                ))}
            </>
          ) : (
            <>
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">
                  Integration Name
                  <span className="text-gray-800 ml-1">*</span>
                </Label>
                <Input
                  type="text"
                  value={integrationName}
                  onChange={(e) => setIntegrationName(e.target.value)}
                  placeholder="e.g., my-app-integration"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">A unique name for this integration</p>
              </div>
              {configurationFields.length > 0 && (
                <div className="space-y-4">
                  {configurationFields.map((field: ConfigurationField) => {
                    if (!field.name) return null;
                    return (
                      <ConfigurationFieldRenderer
                        key={field.name}
                        field={field}
                        value={configuration[field.name]}
                        onChange={(value) =>
                          setConfiguration((prev) => ({
                            ...prev,
                            [field.name || ""]: value,
                          }))
                        }
                        allValues={configuration}
                        domainId={organizationId}
                        domainType="DOMAIN_TYPE_ORGANIZATION"
                        organizationId={organizationId}
                      />
                    );
                  })}
                </div>
              )}
            </>
          )}
        </div>

        <DialogFooter className="gap-2 sm:justify-start mt-6">
          {createdIncidentIntegration ? (
            <>
              <Button
                color="blue"
                onClick={() => void handleCompleteIncidentSetup()}
                disabled={updateIntegrationMutation.isPending}
                className="flex items-center gap-2"
              >
                {updateIntegrationMutation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Completing...
                  </>
                ) : (
                  "Complete setup"
                )}
              </Button>
              <Button variant="outline" onClick={handleClose} disabled={updateIntegrationMutation.isPending}>
                Done
              </Button>
            </>
          ) : createIntegrationBrowserAction ? (
            <>
              <Button color="blue" onClick={handleClose}>
                Save
              </Button>
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
            </>
          ) : (
            <>
              <Button
                color="blue"
                onClick={() => void handleSubmit()}
                disabled={createIntegrationMutation.isPending || !integrationName?.trim()}
                className="flex items-center gap-2"
              >
                {createIntegrationMutation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  "Connect"
                )}
              </Button>
              <Button variant="outline" onClick={handleClose} disabled={createIntegrationMutation.isPending}>
                Cancel
              </Button>
            </>
          )}
        </DialogFooter>

        {createIntegrationMutation.isError && (
          <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">
              Failed to create integration: {getApiErrorMessage(createIntegrationMutation.error)}
            </p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
