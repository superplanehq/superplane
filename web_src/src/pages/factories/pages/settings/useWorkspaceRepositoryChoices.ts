import { useMemo } from "react";

import { useIntegrationResources } from "@/hooks/useIntegrations";

import { useGitHubOnboarding } from "../onboarding/useGitHubOnboarding";
import type { VcsHostId } from "../onboarding/onboardingFixtures";

type RepositoryWorkspace = {
  onboarding?: {
    vcsIntegrationId?: string;
    vcsProvider?: string;
    appRepositoryId?: string;
    backlogRepositoryId?: string;
  } | null;
};

/**
 * The API serializes int64 repository IDs as strings with EmitUnpopulated,
 * so an unset ID arrives as "0", which is truthy in JavaScript. Only a
 * non-zero ID means the workspace uses the version-control catalog.
 */
function hasCatalogRepositoryId(id: string | null | undefined): boolean {
  return Boolean(id && id !== "0");
}

/** Repositories the workspace can choose, the same list as workspace repository settings. */
export function useWorkspaceRepositoryChoices(organizationId: string, factory: RepositoryWorkspace | null | undefined) {
  const integrationId = factory?.onboarding?.vcsIntegrationId ?? "";
  const bitbucket = factory?.onboarding?.vcsProvider === "bitbucket";
  const host: VcsHostId = bitbucket ? "bitbucket" : "github";
  const resources = useIntegrationResources(organizationId, integrationId, "repository");
  const githubOnboarding = useGitHubOnboarding(organizationId, { poll: false });
  const usesCatalogRepository =
    hasCatalogRepositoryId(factory?.onboarding?.appRepositoryId) ||
    hasCatalogRepositoryId(factory?.onboarding?.backlogRepositoryId);
  const catalogRepositories = githubOnboarding.data?.repositories;
  const repositories = useMemo(() => {
    const names = usesCatalogRepository
      ? (catalogRepositories ?? []).map((repository) => repository.fullName ?? "")
      : (resources.data ?? []).map((resource) => resource.name ?? resource.id ?? "");
    return names
      .filter((repository): repository is string => Boolean(repository))
      .sort((left, right) => left.localeCompare(right));
  }, [catalogRepositories, resources.data, usesCatalogRepository]);
  const isLoading =
    (usesCatalogRepository ? githubOnboarding.isPending : resources.isLoading) && repositories.length === 0;
  const isError = (usesCatalogRepository ? githubOnboarding.isError : resources.isError) && repositories.length === 0;

  return {
    host,
    bitbucket,
    integrationId,
    repositories,
    isLoading,
    isError,
    catalogRepositories,
    usesCatalogRepository,
  };
}
