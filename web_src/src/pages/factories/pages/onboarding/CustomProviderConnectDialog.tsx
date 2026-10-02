import type { OrganizationsIntegration } from "@/api-client";
import { organizationsDeleteIntegration } from "@/api-client/sdk.gen";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { integrationKeys, useCreateIntegration } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { CUSTOM_LLM_API_TYPES } from "@/pages/factories/pages/settings/organizationLLMModelsCopy";
import { selectionFromInstance, type IntegrationSelections } from "@/pages/home/homeIntegrationStatus";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";

const CUSTOM_LLM_INTEGRATION = "customLlm";

export function CustomProviderConnectDialog({
  open,
  organizationId,
  existingNames,
  selections,
  onSelectionsChange,
  onClose,
}: {
  open: boolean;
  organizationId: string;
  existingNames: Set<string>;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  onClose: () => void;
}) {
  const createIntegration = useCreateIntegration(organizationId, "install_wizard");
  const queryClient = useQueryClient();
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [apiType, setApiType] = useState("");
  const ready = baseUrl.trim() !== "" && apiKey.trim() !== "" && apiType !== "";

  const close = () => {
    setBaseUrl("");
    setApiKey("");
    setApiType("");
    onClose();
  };

  const submit = async () => {
    if (!ready || createIntegration.isPending) return;
    try {
      const response = await createIntegration.mutateAsync({
        integrationName: CUSTOM_LLM_INTEGRATION,
        name: getNextIntegrationName(CUSTOM_LLM_INTEGRATION, existingNames),
        configuration: {
          baseURL: baseUrl.trim().replace(/\/+$/, ""),
          apiKey: apiKey.trim(),
          apiType,
        },
      });
      const integration = response.data?.integration;
      if (integration?.status?.state !== "ready") {
        try {
          await discardUnreadyCustomConnection(queryClient, organizationId, integration);
        } catch {
          // Keep the provider error. A delete failure must not hide it.
        }
        showErrorToast(connectError(integration));
        return;
      }
      const selection = selectionFromInstance(integration);
      if (!selection) {
        showErrorToast(connectError(integration));
        return;
      }
      onSelectionsChange({ ...selections, [CUSTOM_LLM_INTEGRATION]: selection });
      close();
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "SuperPlane could not connect this provider."));
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? undefined : close())}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Custom provider</DialogTitle>
          <DialogDescription>
            Set the provider URL, token, and API type. SuperPlane stores the token for agents in this workspace.
          </DialogDescription>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <CustomProviderFields
            baseUrl={baseUrl}
            apiKey={apiKey}
            apiType={apiType}
            onBaseUrlChange={setBaseUrl}
            onApiKeyChange={setApiKey}
            onApiTypeChange={setApiType}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={close}>
              Cancel
            </Button>
            <Button type="submit" disabled={!ready || createIntegration.isPending}>
              {createIntegration.isPending ? "Connecting…" : "Connect"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function CustomProviderFields({
  baseUrl,
  apiKey,
  apiType,
  onBaseUrlChange,
  onApiKeyChange,
  onApiTypeChange,
}: {
  baseUrl: string;
  apiKey: string;
  apiType: string;
  onBaseUrlChange: (value: string) => void;
  onApiKeyChange: (value: string) => void;
  onApiTypeChange: (value: string) => void;
}) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="onboarding-custom-url">API URL</Label>
        <Input
          id="onboarding-custom-url"
          data-testid="onboarding-custom-url"
          type="url"
          autoComplete="off"
          placeholder="https://example.com/v1"
          value={baseUrl}
          onChange={(event) => onBaseUrlChange(event.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="onboarding-custom-token">API token</Label>
        <Input
          id="onboarding-custom-token"
          data-testid="onboarding-custom-token"
          type="password"
          autoComplete="off"
          value={apiKey}
          onChange={(event) => onApiKeyChange(event.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="onboarding-custom-api-type">API type</Label>
        <Select value={apiType} onValueChange={onApiTypeChange}>
          <SelectTrigger id="onboarding-custom-api-type" data-testid="onboarding-custom-api-type">
            <SelectValue placeholder="Select an API type" />
          </SelectTrigger>
          <SelectContent>
            {CUSTOM_LLM_API_TYPES.map((type) => (
              <SelectItem key={type.id} value={type.id}>
                {type.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </>
  );
}

export function OnboardingConnectDialogs({
  connectDialogs,
  customProviderOpen,
  organizationId,
  existingNames,
  selections,
  onSelectionsChange,
  onCloseCustomProvider,
}: {
  connectDialogs: ReactNode;
  customProviderOpen: boolean;
  organizationId: string;
  existingNames: Set<string>;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  onCloseCustomProvider: () => void;
}) {
  return (
    <>
      {connectDialogs}
      <CustomProviderConnectDialog
        open={customProviderOpen}
        organizationId={organizationId}
        existingNames={existingNames}
        selections={selections}
        onSelectionsChange={onSelectionsChange}
        onClose={onCloseCustomProvider}
      />
    </>
  );
}

async function discardUnreadyCustomConnection(
  queryClient: QueryClient,
  organizationId: string,
  integration: OrganizationsIntegration | undefined,
): Promise<void> {
  const integrationId = integration?.metadata?.id?.trim();
  if (!integrationId) {
    return;
  }

  await organizationsDeleteIntegration(
    withOrganizationHeader({
      organizationId,
      path: { id: organizationId, integrationId },
    }),
  );
  await queryClient.invalidateQueries({ queryKey: integrationKeys.connected(organizationId) });
  queryClient.removeQueries({ queryKey: integrationKeys.integration(organizationId, integrationId) });
}

function connectError(integration: OrganizationsIntegration | undefined): string {
  const description = integration?.status?.stateDescription?.trim();
  if (description) return description;
  return "SuperPlane could not connect this provider. Check the URL, token, and API type.";
}
