import { useCallback, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { OrganizationsBrowserAction } from "@/api-client";
import { integrationKeys } from "@/hooks/useIntegrations";
import type { useUpdateIntegration } from "@/hooks/useIntegrations";
import { showErrorToast } from "@/lib/toast";

interface UseBrowserActionSetupParams {
  updateIntegrationMutation: ReturnType<typeof useUpdateIntegration>;
  configuration: Record<string, unknown>;
  organizationId: string;
  integrationId: string | undefined;
  integrationName: string;
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

  // Opens the action's URL (or submits its POST form), then advances the dialog - even when
  // resuming a pending integration, since there is no follow-up round trip in that case.
  const continueBrowserAction = useCallback(() => {
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
    } else if (url) {
      window.open(url, "_blank");
    }

    setBrowserActionCompleted(true);
  }, [browserAction]);

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
