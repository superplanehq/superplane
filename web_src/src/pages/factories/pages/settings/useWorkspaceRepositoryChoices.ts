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

type NamedResource = {
  name?: string | null;
  id?: string | null;
};

type CatalogRepository = {
  fullName?: string | null;
};

/**
 * The API serializes int64 repository IDs as strings with EmitUnpopulated,
 * so an unset ID arrives as "0", which is truthy in JavaScript. Only a
 * non-zero ID means the workspace uses the version-control catalog.
 */
function hasCatalogRepositoryId(id: string | null | undefined): boolean {
  return Boolean(id && id !== "0");
}

function workspaceRepositoryHost(factory: RepositoryWorkspace | null | undefined): {
  integrationId: string;
  bitbucket: boolean;
  host: VcsHostId;
} {
  const onboarding = factory?.onboarding;
  const bitbucket = onboarding?.vcsProvider === "bitbucket";
  return {
    integrationId: onboarding?.vcsIntegrationId ?? "",
    bitbucket,
    host: bitbucket ? "bitbucket" : "github",
  };
}

function workspaceUsesCatalogRepository(factory: RepositoryWorkspace | null | undefined): boolean {
  const onboarding = factory?.onboarding;
  return hasCatalogRepositoryId(onboarding?.appRepositoryId) || hasCatalogRepositoryId(onboarding?.backlogRepositoryId);
}

function catalogRepositoryNames(repositories: CatalogRepository[] | undefined): string[] {
  return (repositories ?? []).map((repository) => repository.fullName ?? "");
}

function integrationRepositoryNames(resources: NamedResource[] | undefined): string[] {
  return (resources ?? []).map((resource) => resource.name ?? resource.id ?? "");
}

function sortedRepositoryNames(names: string[]): string[] {
  return names
    .filter((repository): repository is string => Boolean(repository))
    .sort((left, right) => left.localeCompare(right));
}

function repositoryListUnavailable(usesCatalog: boolean, catalogFlag: boolean, resourceFlag: boolean, count: number) {
  const pending = usesCatalog ? catalogFlag : resourceFlag;
  return pending && count === 0;
}

/** Repositories the workspace can choose, the same list as workspace repository settings. */
export function useWorkspaceRepositoryChoices(organizationId: string, factory: RepositoryWorkspace | null | undefined) {
  const { integrationId, bitbucket, host } = workspaceRepositoryHost(factory);
  const resources = useIntegrationResources(organizationId, integrationId, "repository");
  const githubOnboarding = useGitHubOnboarding(organizationId, { poll: false });
  const usesCatalogRepository = workspaceUsesCatalogRepository(factory);
  const catalogRepositories = githubOnboarding.data?.repositories;
  const repositories = useMemo(() => {
    const names = usesCatalogRepository
      ? catalogRepositoryNames(catalogRepositories)
      : integrationRepositoryNames(resources.data);
    return sortedRepositoryNames(names);
  }, [catalogRepositories, resources.data, usesCatalogRepository]);
  const isLoading = repositoryListUnavailable(
    usesCatalogRepository,
    githubOnboarding.isPending,
    resources.isLoading,
    repositories.length,
  );
  const isError = repositoryListUnavailable(
    usesCatalogRepository,
    githubOnboarding.isError,
    resources.isError,
    repositories.length,
  );

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
