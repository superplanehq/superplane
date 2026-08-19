import { useCallback, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { OrganizationsBrowserAction } from "@/api-client";
import { integrationKeys } from "@/hooks/useIntegrations";
import type { useUpdateIntegration } from "@/hooks/useIntegrations";
import { rememberIntegrationSetupReturn } from "@/lib/integrationSetupReturn";
import { showErrorToast } from "@/lib/toast";

interface UseBrowserActionSetupParams {
  updateIntegrationMutation: ReturnType<typeof useUpdateIntegration>;
  configuration: Record<string, unknown>;
  organizationId: string;
  integrationId: string | undefined;
  integrationName: string;
  /** Where to send the browser after the external provider redirects back. */
  returnTo?: string;
  onCreated?: (integrationId: string, instanceName: string) => void;
  handleClose: () => void;
}

/** Owns the browser-action step of integration creation: state plus the three ways it can be advanced. */
export function useBrowserActionSetup({
  updateIntegrationMutation,
  configuration,
  organizationId,
  integrationId,
  integrationName,
  returnTo,
  onCreated,
  handleClose,
}: UseBrowserActionSetupParams) {
  const queryClient = useQueryClient();
  const [browserAction, setBrowserAction] = useState<OrganizationsBrowserAction | undefined>(undefined);
  const [browserActionCompleted, setBrowserActionCompleted] = useState(false);

  const resetBrowserAction = useCallback((next?: OrganizationsBrowserAction) => {
    setBrowserAction(next);
    setBrowserActionCompleted(false);
  }, []);

  // Opens the action's URL (or submits its POST form) in the same tab, so the provider
  // redirects back into this same window. A single tab keeps one origin, so the return
  // marker in localStorage survives the round trip and the caller regains control.
  const continueBrowserAction = useCallback(() => {
    if (!browserAction) return;
    // The provider redirects back to the integration page after setup. Record
    // where to continue so callers such as workspace setup regain control. The
    // legacy GitHub connect creates a new integration during the round trip, so
    // the marker is stored per organization, not per integration id.
    rememberIntegrationSetupReturn(organizationId, returnTo);
    const { url, method, formFields } = browserAction;
    if (method?.toUpperCase() === "POST" && formFields) {
      const form = document.createElement("form");
      form.method = "POST";
      form.action = url || "";
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
      return;
    }
    if (url) {
      window.location.assign(url);
    }
  }, [browserAction, organizationId, returnTo]);

  // Used for a browser action with no URL (e.g. "fill in more config and save"): submits the
  // current configuration and advances to whatever browser action comes next, if any.
  const saveBrowserActionConfig = useCallback(async () => {
    try {
      const response = await updateIntegrationMutation.mutateAsync({ configuration });
      const nextBrowserAction = response.data?.integration?.status?.browserAction;
      if (nextBrowserAction) {
        setBrowserAction(nextBrowserAction);
        return;
      }

      handleClose();
      if (integrationId) {
        onCreated?.(integrationId, integrationName);
      }
    } catch {
      showErrorToast("Failed to save integration configuration");
    }
  }, [updateIntegrationMutation, configuration, integrationId, handleClose, onCreated, integrationName]);

  const finishBrowserActionSetup = useCallback(async () => {
    if (!integrationId) {
      handleClose();
      return;
    }
    try {
      await updateIntegrationMutation.mutateAsync({
        configuration: { ...configuration, installed: "true" },
      });
      await queryClient.invalidateQueries({ queryKey: integrationKeys.connected(organizationId) });
      onCreated?.(integrationId, integrationName);
      handleClose();
    } catch {
      showErrorToast("Failed to sync integration");
    }
  }, [
    integrationId,
    updateIntegrationMutation,
    configuration,
    queryClient,
    organizationId,
    onCreated,
    integrationName,
    handleClose,
  ]);

  return {
    browserAction,
    browserActionCompleted,
    setBrowserAction,
    resetBrowserAction,
    continueBrowserAction,
    saveBrowserActionConfig,
    finishBrowserActionSetup,
  };
}
