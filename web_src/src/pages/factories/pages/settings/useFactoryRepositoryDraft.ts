import { resolveGithubDefaultBranch, useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useMemo, useState } from "react";

import { useFactoryRepository } from "./useFactoryRepository";

type CatalogRepository = {
  fullName?: string | null;
  defaultBranch?: string | null;
};

type RepositoryDraftFactory = {
  onboarding?: {
    appRepository?: string | null;
    defaultBranch?: string | null;
  } | null;
};

export function useFactoryRepositoryDraft(args: {
  organizationId: string;
  factoryId: string;
  factory: RepositoryDraftFactory;
  integrationId: string;
  catalogRepositories?: CatalogRepository[];
}) {
  const savedRepository = args.factory.onboarding?.appRepository ?? "";
  const savedBranch = args.factory.onboarding?.defaultBranch?.trim() ?? "";
  const [repository, setRepository] = useState(savedRepository);
  const [branch, setBranch] = useState(savedBranch);
  const [baselineBranch, setBaselineBranch] = useState(savedBranch);
  const [isResolvingDefaultBranch, setIsResolvingDefaultBranch] = useState(false);
  const updateRepository = useFactoryRepository(args.organizationId, args.factoryId);
  const branchResources = useIntegrationResources(
    args.organizationId,
    args.integrationId,
    "branch",
    repository ? { repository } : undefined,
    { enabled: Boolean(repository) },
  );
  const hostDefaultBranch = useCallback(
    (repositoryName: string) =>
      hostDefaultBranchFor(args.organizationId, args.integrationId, args.catalogRepositories, repositoryName),
    [args.catalogRepositories, args.integrationId, args.organizationId],
  );

  useEffect(() => {
    setRepository(savedRepository);
  }, [savedRepository]);

  useEffect(() => {
    return prefillBranch(savedBranch, savedRepository, hostDefaultBranch, setBranch, setBaselineBranch);
  }, [hostDefaultBranch, savedBranch, savedRepository]);

  const listedBranchNames = useMemo(() => branchNames(branchResources.data), [branchResources.data]);
  const branchChoices = useMemo(() => branchChoicesFor(listedBranchNames, branch), [branch, listedBranchNames]);
  const isDirty = repository !== savedRepository || branch !== baselineBranch;
  const isSaving = isResolvingDefaultBranch || updateRepository.isPending;
  const selectRepository = (next: string) => {
    setRepository(next);
    if (next === repository) return;
    setIsResolvingDefaultBranch(true);
    void hostDefaultBranch(next)
      .then((value) => setBranch(value))
      .finally(() => setIsResolvingDefaultBranch(false));
  };
  const save = async () => {
    if (!repository || !branch || !args.integrationId || isSaving) return;
    try {
      await updateRepository.mutateAsync({ repository, defaultBranch: branch });
      showSuccessToast("Workspace repository updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update workspace repository"));
    }
  };

  return {
    repository,
    branch,
    setBranch,
    listedBranchNames,
    branchChoices,
    isDirty,
    isSaving,
    selectRepository,
    save,
  };
}

function hostDefaultBranchFor(
  organizationId: string,
  integrationId: string,
  catalogRepositories: CatalogRepository[] | undefined,
  repositoryName: string,
) {
  const catalogBranch = catalogRepositories
    ?.find((candidate) => candidate.fullName === repositoryName)
    ?.defaultBranch?.trim();
  if (catalogBranch) return Promise.resolve(catalogBranch);
  if (!organizationId || !integrationId || !repositoryName) return Promise.resolve("");
  return resolveGithubDefaultBranch(organizationId, integrationId, repositoryName);
}

function prefillBranch(
  savedBranch: string,
  savedRepository: string,
  hostDefaultBranch: (repositoryName: string) => Promise<string>,
  setBranch: (branch: string) => void,
  setBaselineBranch: (branch: string) => void,
) {
  if (savedBranch) {
    setBranch(savedBranch);
    setBaselineBranch(savedBranch);
    return;
  }
  let cancelled = false;
  void hostDefaultBranch(savedRepository).then((next) => {
    if (cancelled) return;
    setBranch(next);
    setBaselineBranch(next);
  });
  return () => {
    cancelled = true;
  };
}

function branchNames(resources: { name?: string | null; id?: string | null }[] | undefined) {
  return (resources ?? []).map((resource) => resource.name?.trim() || resource.id?.trim() || "").filter(Boolean);
}

function branchChoicesFor(listedBranchNames: string[], branch: string) {
  const names = new Set(listedBranchNames);
  if (branch) names.add(branch);
  return [...names].sort((left, right) => left.localeCompare(right));
}
