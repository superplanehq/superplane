import type { FactoriesFactory } from "@/api-client";
import { factoriesListFactories, organizationsDeleteIntegration } from "@/api-client/sdk.gen";
import { integrationKeys } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { QueryClient } from "@tanstack/react-query";

import type { UpdateOnboarding } from "./onboardingProvision";
import { unusedOnboardingVcsIntegrationId } from "./unusedOnboardingIntegration";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export async function saveSelectedGithubConnection(
  args: {
    setup: OnboardingSetupApi;
    updateOnboarding: UpdateOnboarding;
  },
  integrationId: string,
  previousId: string | undefined,
): Promise<boolean> {
  const switchedConnection = Boolean(previousId && previousId !== integrationId);
  if (switchedConnection) {
    args.setup.clearRepository();
  }

  try {
    await args.updateOnboarding({
      vcsIntegrationId: integrationId,
      ...(switchedConnection ? { appRepository: "" } : {}),
    });
    return true;
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Could not save the GitHub connection"));
    return false;
  }
}

export async function deleteUnusedOnboardingIntegration(args: {
  organizationId: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  factoryId: string;
  previousId: string | undefined;
  nextId: string;
  queryClient: QueryClient;
}): Promise<void> {
  const factories = await listOrganizationFactories(args.organizationId);
  const unusedId = unusedOnboardingVcsIntegrationId({
    isInitial: args.factory?.onboarding?.initial === true,
    previousId: args.previousId,
    nextId: args.nextId,
    factories: factories.length > 0 ? factories : args.factories,
    currentFactoryId: args.factoryId,
  });
  if (!unusedId) return;

  try {
    await organizationsDeleteIntegration(
      withOrganizationHeader({
        organizationId: args.organizationId,
        path: { id: args.organizationId, integrationId: unusedId },
      }),
    );
    void args.queryClient.invalidateQueries({ queryKey: integrationKeys.connected(args.organizationId) });
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Could not remove the unused GitHub connection"));
  }
}

async function listOrganizationFactories(organizationId: string): Promise<FactoriesFactory[]> {
  try {
    const response = await factoriesListFactories(withOrganizationHeader({ organizationId }));
    return response.data?.factories ?? [];
  } catch {
    return [];
  }
}
