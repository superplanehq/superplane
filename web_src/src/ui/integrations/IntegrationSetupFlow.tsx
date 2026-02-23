import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, Loader2 } from "lucide-react";
import type { ConfigurationField, IntegrationsIntegrationDefinition } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/ui/alert";
import { showErrorToast } from "@/utils/toast";
import { getApiErrorMessage } from "@/utils/errors";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import {
  useAvailableIntegrations,
  useCreateIntegration,
  useIntegration,
  useUpdateIntegration,
} from "@/hooks/useIntegrations";

interface IntegrationSetupFlowProps {
  organizationId: string;
  integrationName: string;
  integrationDefinition?: IntegrationsIntegrationDefinition;
  onCancel?: () => void;
  onCompleted?: (integrationId: string) => void;
  onStateChange?: (state: { name: string; status?: string; isCreated: boolean; isFinalStep: boolean }) => void;
}

export function IntegrationSetupFlow({
  organizationId,
  integrationName,
  integrationDefinition,
  onCancel,
  onCompleted,
  onStateChange,
}: IntegrationSetupFlowProps) {
  const [installationName, setInstallationName] = useState(integrationName);
  const [createdIntegrationId, setCreatedIntegrationId] = useState<string | null>(null);
  const [configValues, setConfigValues] = useState<Record<string, unknown>>({});

  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const createMutation = useCreateIntegration(organizationId);
  const updateMutation = useUpdateIntegration(organizationId, createdIntegrationId || "");
  const { data: integration, isLoading: isLoadingIntegration } = useIntegration(
    organizationId,
    createdIntegrationId || "",
  );

  const activeDefinition = useMemo(() => {
    if (integrationDefinition) {
      return integrationDefinition;
    }

    if (!integrationName) {
      return undefined;
    }

    return availableIntegrations.find((item) => item.name === integrationName);
  }, [availableIntegrations, integrationDefinition, integrationName]);

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

    if (url) {
      window.open(url, "_blank");
    }
  }, [integration?.status?.browserAction]);

  const handleCreate = useCallback(async () => {
    const nextName = installationName.trim();
    if (!nextName) {
      showErrorToast("Integration name is required");
      return;
    }

    try {
      const result = await createMutation.mutateAsync({
        integrationName,
        name: nextName,
        configuration: {},
      });

      const integrationId = result.data?.integration?.metadata?.id;
      if (!integrationId) {
        throw new Error("Failed to create integration");
      }

      setCreatedIntegrationId(integrationId);
      setConfigValues(result.data?.integration?.spec?.configuration || {});
    } catch (error) {
      showErrorToast(`Failed to create integration: ${getApiErrorMessage(error)}`);
    }
  }, [createMutation, installationName, integrationName]);

  const handleSave = useCallback(async () => {
    if (!createdIntegrationId) return;

    try {
      await updateMutation.mutateAsync({
        configuration: configValues,
      });
    } catch (error) {
      showErrorToast(`Failed to update integration: ${getApiErrorMessage(error)}`);
    }
  }, [configValues, createdIntegrationId, updateMutation]);

  useEffect(() => {
    if (!integration?.spec?.configuration) return;
    setConfigValues(integration.spec.configuration);
  }, [integration?.spec?.configuration]);

  useEffect(() => {
    if (!integration?.metadata?.name) return;
    setInstallationName(integration.metadata.name);
  }, [integration?.metadata?.name]);

  useEffect(() => {
    const isReady = integration?.status?.state === "ready";
    const nextIsFinalStep = Boolean(createdIntegrationId && isReady && !integration?.status?.browserAction);
    onStateChange?.({
      name: installationName,
      status: integration?.status?.state,
      isCreated: Boolean(createdIntegrationId),
      isFinalStep: nextIsFinalStep,
    });
  }, [
    onStateChange,
    installationName,
    integration?.status?.state,
    integration?.status?.browserAction,
    createdIntegrationId,
  ]);

  const isCreated = Boolean(createdIntegrationId);
  const isReady = integration?.status?.state === "ready";
  const canFinish = Boolean(createdIntegrationId && isReady && !integration?.status?.browserAction);
  const isFinalStep = canFinish;

  return (
    <div className="space-y-4">
      {!isFinalStep && !isCreated ? (
        <div className="space-y-2">
          <div className="flex items-center gap-4">
            <Label className="w-10 shrink-0 text-gray-800 dark:text-gray-100">Name</Label>
            <Input
              type="text"
              value={installationName}
              onChange={(e) => setInstallationName(e.target.value)}
              placeholder="e.g., my-slack-prod"
            />
          </div>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            You can connect the same provider multiple times, but each integration name
            must be unique in your organization.
          </p>
        </div>
      ) : null}
      {isCreated && isLoadingIntegration ? (
        <div className="flex items-center justify-center py-6">
          <Loader2 className="h-5 w-5 animate-spin text-gray-500" />
        </div>
      ) : null}

      {isCreated && integration?.status?.state === "error" && integration.status?.stateDescription ? (
        <Alert variant="destructive" className="[&>svg+div]:translate-y-0 [&>svg]:top-[14px]">
          <AlertDescription>{integration.status.stateDescription}</AlertDescription>
        </Alert>
      ) : null}

      {isCreated && integration?.status?.browserAction ? (
        <IntegrationInstructions
          description={integration.status.browserAction.description}
          onContinue={integration.status.browserAction.url ? handleBrowserAction : undefined}
        />
      ) : null}

      {!isFinalStep && isCreated ? (
        <div className="space-y-3">
          <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-4">
            {activeDefinition?.configuration && activeDefinition.configuration.length > 0 ? (
              activeDefinition.configuration.map((field: ConfigurationField) => {
                if (!field.name) return null;
                return (
                  <ConfigurationFieldRenderer
                    key={field.name}
                    field={field}
                    value={configValues[field.name]}
                    onChange={(value) => setConfigValues((prev) => ({ ...prev, [field.name || ""]: value }))}
                    allValues={configValues}
                    domainId={organizationId}
                    domainType="DOMAIN_TYPE_ORGANIZATION"
                    organizationId={organizationId}
                    integrationId={createdIntegrationId || undefined}
                  />
                );
              })
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">No configuration fields available.</p>
            )}
            <div className="flex items-center gap-2 pt-2">
              <Button onClick={() => void handleSave()} disabled={updateMutation.isPending}>
                {updateMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : (
                  "Save"
                )}
              </Button>
              {onCancel ? (
                <Button variant="outline" onClick={onCancel} disabled={updateMutation.isPending}>
                  Cancel
                </Button>
              ) : null}
            </div>
            {updateMutation.isError ? (
              <p className="text-sm text-red-600 dark:text-red-400">
                Failed to update integration: {getApiErrorMessage(updateMutation.error)}
              </p>
            ) : null}
          </div>
        </div>
      ) : null}

      {isFinalStep ? (
        <div className="space-y-6">
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-500">
              <Check className="h-9 w-9 text-white" />
            </div>
            <h3 className="mt-4 text-2xl font-semibold text-gray-900 dark:text-gray-100">All set</h3>
            <p className="mt-2 max-w-md text-sm text-gray-600 dark:text-gray-300">
              <span className="font-medium">{installationName || "This integration"}</span> is connected and ready to
              use in workflows.
            </p>
            <p className="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">
              Click <span className="font-medium">Finish</span> to continue. You can reopen this integration anytime
              from settings to update configuration or reconnect if needed.
            </p>
          </div>
          <div className="flex justify-center">
            <Button onClick={() => onCompleted?.(createdIntegrationId!)}>Finish</Button>
          </div>
        </div>
      ) : !isCreated ? (
        <div className={`flex items-center gap-2 pt-2 ${isCreated ? "" : "justify-end"}`}>
          <>
            {onCancel ? (
              <Button
                variant="outline"
                onClick={onCancel}
                disabled={createMutation.isPending || updateMutation.isPending}
              >
                Cancel
              </Button>
            ) : null}
            <Button onClick={() => void handleCreate()} disabled={createMutation.isPending || !installationName.trim()}>
              {createMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Next"
              )}
            </Button>
          </>
        </div>
      ) : null}

      {createMutation.isError ? (
        <p className="text-sm text-red-600 dark:text-red-400">
          Failed to create integration: {getApiErrorMessage(createMutation.error)}
        </p>
      ) : null}
    </div>
  );
}
