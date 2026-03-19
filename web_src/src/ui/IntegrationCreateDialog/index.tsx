import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Settings } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { getIntegrationTypeDisplayName } from "@/utils/integrationDisplayName";
import { getApiErrorMessage } from "@/utils/errors";
import { showErrorToast, showSuccessToast } from "@/utils/toast";
import { integrationKeys, useUpdateIntegration } from "@/hooks/useIntegrations";
import { organizationsUpdateIntegration } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { useQueryClient } from "@tanstack/react-query";
import type {
  ConfigurationField,
  IntegrationsIntegrationDefinition,
  OrganizationsBrowserAction,
  OrganizationsCreateIntegrationResponse,
} from "@/api-client";

export type IntegrationCreatePayload = {
  integrationName: string;
  name: string;
  configuration?: Record<string, unknown>;
};

export interface IntegrationCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  integrationDefinition: IntegrationsIntegrationDefinition | null | undefined;
  organizationId: string;
  /** Called to create the integration. Returns the API response (integration id, browser action, webhook url, etc.). */
  onCreateIntegration: (payload: IntegrationCreatePayload) => Promise<OrganizationsCreateIntegrationResponse>;
  /** Optional: called when dialog closes so parent can reset mutation state. */
  onReset?: () => void;
  defaultName?: string;
  integrationHomeHref?: string;
  onCreated?: (integrationId: string) => void;
  /** If set, instructions are truncated at this heading (e.g. "## Webhook integration") so only the part before is shown in the create step. */
  instructionsEndBeforeHeading?: string;
  /** If set, only these configuration field names are shown in the initial create step; the rest are shown in the webhook completion step. */
  initialStepFieldNames?: string[];
  /** Optional custom description for the webhook completion step. */
  webhookStepDescription?: ReactNode;
}

export function IntegrationCreateDialog({
  open,
  onOpenChange,
  integrationDefinition,
  organizationId,
  onCreateIntegration,
  onReset,
  defaultName = "",
  integrationHomeHref,
  onCreated,
  instructionsEndBeforeHeading,
  initialStepFieldNames,
  webhookStepDescription,
}: IntegrationCreateDialogProps) {
  const queryClient = useQueryClient();
  const [integrationName, setIntegrationName] = useState(defaultName);
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [createIntegrationBrowserAction, setCreateIntegrationBrowserAction] = useState<
    OrganizationsBrowserAction | undefined
  >(undefined);
  const [pendingWebhookSetup, setPendingWebhookSetup] = useState<{
    id: string;
    webhookUrl: string;
    config: Record<string, unknown>;
  } | null>(null);
  const [isCreatePending, setIsCreatePending] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [createdIntegrationId, setCreatedIntegrationId] = useState<string | undefined>(undefined);
  const [browserActionCompleted, setBrowserActionCompleted] = useState(false);

  const updateIntegrationMutation = useUpdateIntegration(
    organizationId,
    pendingWebhookSetup?.id ?? createdIntegrationId ?? "",
  );

  const selectedInstructions = useMemo(() => {
    const raw = integrationDefinition?.instructions?.trim();
    if (!raw || !instructionsEndBeforeHeading) return raw ?? "";
    const idx = raw.indexOf(instructionsEndBeforeHeading);
    return idx >= 0 ? raw.slice(0, idx).trim() : raw;
  }, [integrationDefinition?.instructions, instructionsEndBeforeHeading]);

  const configurationFields = useMemo(() => {
    const fields = integrationDefinition?.configuration ?? [];
    if (!initialStepFieldNames?.length) return fields;
    return fields.filter((f) => f.name && initialStepFieldNames.includes(f.name));
  }, [integrationDefinition?.configuration, initialStepFieldNames]);

  useEffect(() => {
    if (open) {
      setIntegrationName(defaultName);
      setConfiguration({});
      setCreateIntegrationBrowserAction(undefined);
      setPendingWebhookSetup(null);
      setCreatedIntegrationId(undefined);
      setBrowserActionCompleted(false);
    }
  }, [open, defaultName]);

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setIntegrationName("");
        setConfiguration({});
        setCreateIntegrationBrowserAction(undefined);
        setPendingWebhookSetup(null);
        setCreatedIntegrationId(undefined);
        setBrowserActionCompleted(false);
        setCreateError(null);
        onReset?.();
      }
      onOpenChange(next);
    },
    [onOpenChange, onReset],
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

    setCreateError(null);
    setIsCreatePending(true);
    try {
      const result = await onCreateIntegration({
        integrationName: integrationDefinition.name,
        name: nextName,
        configuration,
      });

      const integration = result.integration;
      const browserAction = integration?.status?.browserAction;
      const webhookUrl =
        integration?.status?.metadata &&
        typeof integration.status.metadata === "object" &&
        "webhookUrl" in integration.status.metadata
          ? (integration.status.metadata as { webhookUrl?: string }).webhookUrl
          : undefined;

      if (browserAction) {
        setCreateIntegrationBrowserAction(browserAction);
        if (integration?.metadata?.id) {
          setCreatedIntegrationId(integration.metadata.id);
        }
        return;
      }
      if (integration?.metadata?.id && webhookUrl) {
        setPendingWebhookSetup({
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
      const message = getApiErrorMessage(error);
      setCreateError(message);
      showErrorToast(`Failed to create integration: ${message}`);
    } finally {
      setIsCreatePending(false);
    }
  }, [
    integrationDefinition?.name,
    organizationId,
    integrationName,
    configuration,
    onCreateIntegration,
    handleClose,
    onCreated,
  ]);

  const handleCompleteWebhookSetup = useCallback(async () => {
    if (!pendingWebhookSetup) return;

    try {
      await updateIntegrationMutation.mutateAsync({
        configuration: { ...pendingWebhookSetup.config, ...configuration },
      });
      handleClose();
      onCreated?.(pendingWebhookSetup.id);
    } catch {
      showErrorToast("Failed to complete setup");
    }
  }, [pendingWebhookSetup, configuration, updateIntegrationMutation, handleClose, onCreated]);

  const handleBrowserActionContinue = useCallback(async () => {
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

    // Trigger a resync so the backend transitions from pending to ready
    if (createdIntegrationId) {
      try {
        await organizationsUpdateIntegration(
          withOrganizationHeader({
            path: { id: organizationId, integrationId: createdIntegrationId },
            body: { configuration: { ...configuration, installed: "true" } },
          }),
        );
        await queryClient.invalidateQueries({
          queryKey: integrationKeys.connected(organizationId),
        });
      } catch {
        // Resync is best-effort; the integration will be resynced eventually
      }
      setBrowserActionCompleted(true);
    }
  }, [createIntegrationBrowserAction, createdIntegrationId, configuration, organizationId, queryClient]);

  if (!integrationDefinition) return null;

  const displayName =
    getIntegrationTypeDisplayName(undefined, integrationDefinition.name) || integrationDefinition.name;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="sm:max-w-2xl max-h-[80vh] overflow-y-auto"
        showCloseButton={!isCreatePending && !updateIntegrationMutation.isPending}
      >
        <DialogHeader>
          <div className="flex items-center gap-3">
            <IntegrationIcon
              integrationName={integrationDefinition.name}
              iconSlug={integrationDefinition.icon}
              className="h-6 w-6 text-gray-500 dark:text-gray-400"
            />
            <div className="flex items-center gap-2">
              <DialogTitle>{pendingWebhookSetup ? "Complete webhook setup" : `Configure ${displayName}`}</DialogTitle>
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
          {!pendingWebhookSetup && (createIntegrationBrowserAction?.description || selectedInstructions) && (
            <IntegrationInstructions
              description={
                browserActionCompleted
                  ? [selectedInstructions, createIntegrationBrowserAction?.description].filter(Boolean).join("\n\n")
                  : ((createIntegrationBrowserAction?.description || selectedInstructions) ?? "")
              }
              onContinue={
                !browserActionCompleted && createIntegrationBrowserAction?.url
                  ? () => void handleBrowserActionContinue()
                  : undefined
              }
              className="mt-2"
            />
          )}
        </DialogHeader>

        <div className="space-y-4">
          {pendingWebhookSetup ? (
            <>
              {webhookStepDescription ?? (
                <p className="text-sm text-gray-800 dark:text-gray-200">
                  Copy the webhook URL below and complete the required steps in your integration provider. Then enter
                  any required secrets below.
                </p>
              )}
              <div>
                <Label className="text-gray-800 dark:text-gray-100 mb-2">Webhook URL</Label>
                <div className="flex gap-2">
                  <Input type="text" readOnly value={pendingWebhookSetup.webhookUrl} className="font-mono text-sm" />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={async () => {
                      try {
                        await navigator.clipboard.writeText(pendingWebhookSetup.webhookUrl);
                        showSuccessToast("Copied to clipboard");
                      } catch {
                        showErrorToast("Failed to copy to clipboard");
                      }
                    }}
                  >
                    Copy
                  </Button>
                </div>
              </div>
              {(integrationDefinition?.configuration ?? [])
                .filter((f: ConfigurationField) => {
                  if (!f.name) return false;
                  if (initialStepFieldNames?.length) return !initialStepFieldNames.includes(f.name);
                  return f.name === "signingSecret" || f.name === "webhookSigningSecret";
                })
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
          ) : browserActionCompleted ? null : (
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
          {pendingWebhookSetup ? (
            <>
              <LoadingButton
                color="blue"
                onClick={() => void handleCompleteWebhookSetup()}
                loading={updateIntegrationMutation.isPending}
                loadingText="Completing..."
                className="flex items-center gap-2"
              >
                Complete setup
              </LoadingButton>
              <Button variant="outline" onClick={handleClose} disabled={updateIntegrationMutation.isPending}>
                Done
              </Button>
            </>
          ) : createIntegrationBrowserAction ? (
            <>
              {browserActionCompleted ? (
                <Button
                  color="blue"
                  onClick={() => {
                    if (createdIntegrationId) {
                      onCreated?.(createdIntegrationId);
                    }
                    handleClose();
                  }}
                >
                  Done
                </Button>
              ) : (
                <>
                  <LoadingButton
                    color="blue"
                    onClick={async () => {
                      try {
                        await updateIntegrationMutation.mutateAsync({ configuration: { ...configuration } });
                        await queryClient.invalidateQueries({ queryKey: integrationKeys.connected(organizationId) });
                        if (createdIntegrationId) onCreated?.(createdIntegrationId);
                        handleClose();
                      } catch {
                        showErrorToast("Failed to sync integration");
                      }
                    }}
                    loading={updateIntegrationMutation.isPending}
                    loadingText="Saving..."
                  >
                    Save
                  </LoadingButton>
                  <Button variant="outline" onClick={handleClose} disabled={updateIntegrationMutation.isPending}>
                    Cancel
                  </Button>
                </>
              )}
            </>
          ) : (
            <>
              <LoadingButton
                color="blue"
                onClick={() => void handleSubmit()}
                disabled={!integrationName?.trim()}
                loading={isCreatePending}
                loadingText="Connecting..."
                className="flex items-center gap-2"
              >
                Connect
              </LoadingButton>
              <Button variant="outline" onClick={handleClose} disabled={isCreatePending}>
                Cancel
              </Button>
            </>
          )}
        </DialogFooter>

        {createError && (
          <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">Failed to create integration: {createError}</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
