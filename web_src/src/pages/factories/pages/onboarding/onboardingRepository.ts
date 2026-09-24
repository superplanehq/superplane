import type { FactoriesUpdateFactoryOnboardingBody } from "@/api-client";

export function firstRunRepositoryPatch(
  integrationId: string,
  repository: string,
): FactoriesUpdateFactoryOnboardingBody {
  return {
    vcsIntegrationId: integrationId,
    appRepository: repository,
    backlogRepository: repository,
  };
}

export function githubConnectionPatch(
  integrationId: string,
  previousIntegrationId: string | undefined,
): FactoriesUpdateFactoryOnboardingBody {
  if (previousIntegrationId === integrationId) {
    return { vcsIntegrationId: integrationId };
  }

  return {
    vcsIntegrationId: integrationId,
    appRepository: "",
    backlogRepository: "",
    defaultBranch: "",
  };
}
