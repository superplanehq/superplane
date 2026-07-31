import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ExternalLink, Settings } from "lucide-react";
import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/lib/usageLimits";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useUpdateIntegration } from "@/hooks/useIntegrations";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { useBrowserActionSetup } from "./useBrowserActionSetup";
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
  onCreated?: (integrationId: string, instanceName: string) => void;
  /** Called when create returns a capability-based setup step that must continue in the setup wizard. */
  onCapabilitySetupRequired?: (integrationName: string, integrationId: string) => void;
  /** If set, instructions are truncated at this heading (e.g. "## Webhook integration") so only the part before is shown in the create step. */
  instructionsEndBeforeHeading?: string;
  /** If set, only these configuration field names are shown in the initial create step; the rest are shown in the webhook completion step. */
  initialStepFieldNames?: string[];
  /** Optional custom description for the webhook completion step. */
  webhookStepDescription?: ReactNode;
  /** Pre-created integration state for resuming a flow started inline (e.g. browser action after inline creation). */
  initialCreatedIntegrationId?: string;
  initialBrowserAction?: OrganizationsBrowserAction;
  initialWebhookSetup?: { id: string; webhookUrl: string; config: Record<string, unknown> };
  /** Existing configuration to pre-populate when resuming a pending integration flow. */
  initialConfiguration?: Record<string, unknown>;
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
  onCapabilitySetupRequired,
  instructionsEndBeforeHeading,
  initialStepFieldNames,
  webhookStepDescription,
  initialCreatedIntegrationId,
  initialBrowserAction,
  initialWebhookSetup,
  initialConfiguration,
}: IntegrationCreateDialogProps) {
  const [integrationName, setIntegrationName] = useState(defaultName);
  const [configuration, setConfiguration] = useState<Record<string, unknown>>({});
  const [pendingWebhookSetup, setPendingWebhookSetup] = useState<{
    id: string;
    webhookUrl: string;
    config: Record<string, unknown>;
  } | null>(null);
  const [isCreatePending, setIsCreatePending] = useState(false);
  const [createError, setCreateError] = useState<unknown>(null);
  const [createdIntegrationId, setCreatedIntegrationId] = useState<string | undefined>(undefined);
  const prevOpenRef = useRef(false);

  const resolvedIntegrationId = pendingWebhookSetup?.id ?? createdIntegrationId ?? initialCreatedIntegrationId;
  const updateIntegrationMutation = useUpdateIntegration(organizationId, resolvedIntegrationId ?? "");

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

  const handleOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setIntegrationName("");
        setConfiguration({});
        setPendingWebhookSetup(null);
        setCreatedIntegrationId(undefined);
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

  const {
    browserAction: createIntegrationBrowserAction,
    browserActionCompleted,
    setBrowserAction: setCreateIntegrationBrowserAction,
    resetBrowserAction,
    continueBrowserAction: handleBrowserActionContinue,
    saveBrowserActionConfig: handleBrowserActionConfigSave,
    finishBrowserActionSetup: handleFinishBrowserActionSetup,
  } = useBrowserActionSetup({
    updateIntegrationMutation,
    configuration,
    organizationId,
    integrationId: resolvedIntegrationId,
    integrationName,
    onCreated,
    handleClose,
  });

  useEffect(() => {
    const justOpened = open && !prevOpenRef.current;
    prevOpenRef.current = open;
    if (!justOpened) return;

    setIntegrationName(defaultName);
    setConfiguration(initialConfiguration ? { ...initialConfiguration } : {});
    resetBrowserAction(initialBrowserAction ?? undefined);
    setPendingWebhookSetup(initialWebhookSetup ?? null);
    setCreatedIntegrationId(initialCreatedIntegrationId ?? undefined);
  }, [
    open,
    defaultName,
    initialBrowserAction,
    initialWebhookSetup,
    initialCreatedIntegrationId,
    initialConfiguration,
    resetBrowserAction,
  ]);

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
      const webhookUrl = getIntegrationWebhookUrl(integration?.status?.metadata);

      // Capability-based integrations (e.g. GitHub) return a setup wizard step, not a
      // browserAction. Hand off to the setup wizard instead of closing into a dead end.
      if (integration?.status?.setupState?.currentStep && integration.metadata?.id) {
        const setupName = integration.metadata.integrationName || integrationDefinition.name;
        onCapabilitySetupRequired?.(setupName, integration.metadata.id);
        handleClose();
        return;
      }

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
        onCreated?.(integration.metadata.id, nextName);
      }
    } catch (error) {
      setCreateError(error);
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create integration"));
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
    onCapabilitySetupRequired,
    setCreateIntegrationBrowserAction,
  ]);

  const handleCompleteWebhookSetup = useCallback(async () => {
    if (!pendingWebhookSetup) return;

    try {
      await updateIntegrationMutation.mutateAsync({
        configuration: { ...pendingWebhookSetup.config, ...configuration },
      });
      handleClose();
      onCreated?.(pendingWebhookSetup.id, integrationName);
    } catch {
      showErrorToast("Failed to complete setup");
    }
  }, [pendingWebhookSetup, configuration, updateIntegrationMutation, handleClose, onCreated, integrationName]);

  if (!integrationDefinition) return null;

  const displayName =
    getIntegrationTypeDisplayName(undefined, integrationDefinition.name) || integrationDefinition.name;
  const createErrorNotice = createError ? getUsageLimitNotice(createError, organizationId) : null;
  const resolvedHomeHref =
    resolvedIntegrationId && organizationId
      ? `/${organizationId}/settings/integrations/${resolvedIntegrationId}`
      : integrationHomeHref;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        className="sm:max-w-2xl max-h-[80vh] overflow-y-auto border border-edge-default"
        showCloseButton={!isCreatePending && !updateIntegrationMutation.isPending}
      >
        <DialogHeader>
          <div className="flex items-center gap-3">
            <IntegrationIcon
              integrationName={integrationDefinition.name}
              iconSlug={integrationDefinition.icon}
              className="h-6 w-6 text-content-secondary"
            />
            <div className="flex items-center gap-2">
              <DialogTitle>{pendingWebhookSetup ? "Complete webhook setup" : `Configure ${displayName}`}</DialogTitle>
              {resolvedHomeHref && (
                <a
                  href={resolvedHomeHref}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex h-4 w-4 items-center justify-center text-content-secondary hover:text-content-primary transition-colors"
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
              className="mt-2"
            />
          )}
        </DialogHeader>

        <div className="space-y-4 ph-no-capture">
          {pendingWebhookSetup ? (
            <>
              {webhookStepDescription ?? (
                <p className="text-sm text-content-primary">
                  Copy the webhook URL below and complete the required steps in your integration provider. Then enter
                  any required secrets below.
                </p>
              )}
              <div>
                <Label className="text-content-primary mb-2">Webhook URL</Label>
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
                    organizationId={organizationId}
                  />
                ))}
            </>
          ) : (
            <>
              {!browserActionCompleted && (
                <div>
                  <Label className="text-content-primary mb-2">
                    Integration Name
                    <span className="text-content-primary ml-1">*</span>
                  </Label>
                  <Input
                    type="text"
                    value={integrationName}
                    onChange={(e) => setIntegrationName(e.target.value)}
                    placeholder="e.g., my-app-integration"
                  />
                  <p className="text-xs text-content-secondary mt-2">A unique name for this integration</p>
                </div>
              )}
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
                        organizationId={organizationId}
                      />
                    );
                  })}
                </div>
              )}
            </>
          )}
        </div>

        <IntegrationCreateDialogFooter
          pendingWebhookSetup={pendingWebhookSetup}
          createIntegrationBrowserAction={createIntegrationBrowserAction}
          browserActionCompleted={browserActionCompleted}
          mutationPending={updateIntegrationMutation.isPending}
          isCreatePending={isCreatePending}
          integrationName={integrationName}
          onCompleteWebhookSetup={handleCompleteWebhookSetup}
          onBrowserActionContinue={handleBrowserActionContinue}
          onBrowserActionConfigSave={handleBrowserActionConfigSave}
          onFinishBrowserActionSetup={handleFinishBrowserActionSetup}
          onSubmit={handleSubmit}
          onClose={handleClose}
        />

        {createError && createErrorNotice ? <UsageLimitAlert notice={createErrorNotice} className="mt-4" /> : null}
        {createError && !createErrorNotice ? (
          <Alert variant="destructive" className="mt-4">
            <AlertTitle>Unable to create integration</AlertTitle>
            <AlertDescription>Failed to create integration: {getApiErrorMessage(createError)}</AlertDescription>
          </Alert>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

interface IntegrationCreateDialogFooterProps {
  pendingWebhookSetup: { id: string; webhookUrl: string; config: Record<string, unknown> } | null;
  createIntegrationBrowserAction: OrganizationsBrowserAction | undefined;
  browserActionCompleted: boolean;
  mutationPending: boolean;
  isCreatePending: boolean;
  integrationName: string;
  onCompleteWebhookSetup: () => Promise<void>;
  onBrowserActionContinue: () => void;
  onBrowserActionConfigSave: () => Promise<void>;
  onFinishBrowserActionSetup: () => Promise<void>;
  onSubmit: () => Promise<void>;
  onClose: () => void;
}

function IntegrationCreateDialogFooter({
  pendingWebhookSetup,
  createIntegrationBrowserAction,
  browserActionCompleted,
  mutationPending,
  isCreatePending,
  integrationName,
  onCompleteWebhookSetup,
  onBrowserActionContinue,
  onBrowserActionConfigSave,
  onFinishBrowserActionSetup,
  onSubmit,
  onClose,
}: IntegrationCreateDialogFooterProps) {
  if (pendingWebhookSetup) {
    return (
      <DialogFooter className="gap-2 sm:justify-start mt-6">
        <LoadingButton
          color="blue"
          onClick={() => void onCompleteWebhookSetup()}
          loading={mutationPending}
          loadingText="Completing..."
          className="flex items-center gap-2"
        >
          Complete setup
        </LoadingButton>
        <Button variant="outline" onClick={onClose} disabled={mutationPending}>
          Done
        </Button>
      </DialogFooter>
    );
  }

  if (createIntegrationBrowserAction && !browserActionCompleted) {
    return (
      <DialogFooter className="gap-2 sm:justify-start mt-6">
        {createIntegrationBrowserAction.url ? (
          <Button type="button" onClick={onBrowserActionContinue} className="flex items-center gap-2">
            <ExternalLink className="h-4 w-4" />
            Continue setup
          </Button>
        ) : (
          <LoadingButton
            color="blue"
            onClick={() => void onBrowserActionConfigSave()}
            loading={mutationPending}
            loadingText="Saving..."
          >
            Save
          </LoadingButton>
        )}
        <Button variant="outline" onClick={onClose} disabled={mutationPending}>
          Cancel
        </Button>
      </DialogFooter>
    );
  }

  if (createIntegrationBrowserAction) {
    return (
      <DialogFooter className="gap-2 sm:justify-start mt-6">
        <LoadingButton
          color="blue"
          onClick={() => void onFinishBrowserActionSetup()}
          loading={mutationPending}
          loadingText="Saving..."
        >
          Done
        </LoadingButton>
        <Button variant="outline" onClick={onClose} disabled={mutationPending}>
          Cancel
        </Button>
      </DialogFooter>
    );
  }

  return (
    <DialogFooter className="gap-2 sm:justify-start mt-6">
      <LoadingButton
        color="blue"
        onClick={() => void onSubmit()}
        disabled={!integrationName?.trim()}
        loading={isCreatePending}
        loadingText="Connecting..."
        className="flex items-center gap-2"
      >
        Connect
      </LoadingButton>
      <Button variant="outline" onClick={onClose} disabled={isCreatePending}>
        Cancel
      </Button>
    </DialogFooter>
  );
}
