import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { startDirectLinearConnect } from "@/lib/startDirectLinearConnect";
import { showErrorToast } from "@/lib/toast";
import { useCallback } from "react";

export function useHostedLinearConnect({
  organizationId,
  returnTo,
  connected,
  existingIntegrationNames,
  createIntegration,
}: {
  organizationId: string;
  returnTo?: string;
  connected: OrganizationsIntegration[];
  existingIntegrationNames: Set<string>;
  createIntegration: (payload: {
    integrationName: string;
    name: string;
    configuration?: Record<string, unknown>;
  }) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
}) {
  return useCallback(
    async (forceNew = false): Promise<boolean> => {
      try {
        return await startDirectLinearConnect({
          organizationId,
          returnTo,
          existingNames: existingIntegrationNames,
          connected,
          forceNew,
          create: async (payload) => {
            const response = await createIntegration(payload);
            return response.data;
          },
        });
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to connect Linear"));
        return false;
      }
    },
    [connected, createIntegration, existingIntegrationNames, organizationId, returnTo],
  );
}
