import { resolveRepositoryDefaultBranch, useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useEffect, useMemo, useRef, useState } from "react";

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

const BRANCH_LOOKUP_ERROR = "Could not load the default branch. Try again.";

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
  const catalogRepositories = useRef(args.catalogRepositories);
  catalogRepositories.current = args.catalogRepositories;
  const branchLookup = useRef(0);

  useEffect(() => {
    setRepository(savedRepository);
  }, [savedRepository]);

  useEffect(() => {
    const lookup = ++branchLookup.current;
    const isCurrent = () => branchLookup.current === lookup;
    setIsResolvingDefaultBranch(false);
    if (savedBranch) {
      setBranch(savedBranch);
      setBaselineBranch(savedBranch);
      return () => {
        branchLookup.current += 1;
      };
    }

    void watchBranchLookup(
      isCurrent,
      hostDefaultBranchFor(args.organizationId, args.integrationId, catalogRepositories.current, savedRepository),
      (next) => {
        setBranch(next);
        if (next) setBaselineBranch(next);
      },
    );

    return () => {
      branchLookup.current += 1;
    };
  }, [args.integrationId, args.organizationId, savedBranch, savedRepository]);

  const listedBranchNames = useMemo(() => branchNames(branchResources.data), [branchResources.data]);
  const branchChoices = useMemo(() => repositoryBranchChoices(listedBranchNames, branch), [branch, listedBranchNames]);
  const isDirty =
    repository !== savedRepository || branch !== baselineBranch || (savedBranch === "" && branch.trim() !== "");
  const isSaving = isResolvingDefaultBranch || updateRepository.isPending;
  const selectRepository = (next: string) => {
    setRepository(next);
    if (next === repository) return;
    const lookup = ++branchLookup.current;
    const isCurrent = () => branchLookup.current === lookup;
    setIsResolvingDefaultBranch(true);
    void watchBranchLookup(
      isCurrent,
      hostDefaultBranchFor(args.organizationId, args.integrationId, catalogRepositories.current, next),
      setBranch,
    ).finally(() => {
      if (!isCurrent()) return;
      setIsResolvingDefaultBranch(false);
    });
  };
  const save = async (retargetTasks = true) => {
    if (!repository || !branch || !args.integrationId || isSaving) return;
    try {
      await updateRepository.mutateAsync({ repository, defaultBranch: branch, retargetTasks });
      showSuccessToast("Workspace repository updated.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to update workspace repository"));
    }
  };

  return {
    repository,
    savedRepository,
    branch,
    savedBranch,
    setBranch,
    listedBranchNames,
    branchChoices,
    isDirty,
    isSaving,
    selectRepository,
    save,
  };
}

async function hostDefaultBranchFor(
  organizationId: string,
  integrationId: string,
  catalogRepositories: CatalogRepository[] | undefined,
  repositoryName: string,
) {
  const catalogBranch =
    catalogRepositories?.find((candidate) => candidate.fullName === repositoryName)?.defaultBranch?.trim() ?? "";
  if (!organizationId || !integrationId || !repositoryName) return catalogBranch;
  try {
    const live = (await resolveRepositoryDefaultBranch(organizationId, integrationId, repositoryName))?.trim() ?? "";
    if (live) return live;
  } catch (error) {
    if (!catalogBranch) throw error;
  }
  return catalogBranch;
}

function watchBranchLookup(isCurrent: () => boolean, request: Promise<string>, onBranch: (branch: string) => void) {
  return request
    .then((next) => {
      if (!isCurrent()) return;
      onBranch(next);
    })
    .catch((error: unknown) => {
      if (!isCurrent()) return;
      onBranch("");
      showErrorToast(getApiErrorMessage(error, BRANCH_LOOKUP_ERROR));
    });
}

export function branchNames(resources: { name?: string | null; id?: string | null }[] | undefined) {
  return (resources ?? []).map((resource) => resource.name?.trim() || resource.id?.trim() || "").filter(Boolean);
}

const preferredDefaultBranches = ["main", "master", "develop", "dev", "development", "trunk"];
const preferredDefaultBranchRank = new Map(preferredDefaultBranches.map((name, index) => [name, index]));

export function repositoryBranchChoices(listedBranchNames: string[], branch: string) {
  const names = new Set(listedBranchNames);
  if (branch) names.add(branch);
  return [...names].sort(compareBranchChoices);
}

function compareBranchChoices(left: string, right: string) {
  const leftRank = preferredDefaultBranchRank.get(left.toLowerCase()) ?? Number.POSITIVE_INFINITY;
  const rightRank = preferredDefaultBranchRank.get(right.toLowerCase()) ?? Number.POSITIVE_INFINITY;
  if (leftRank !== rightRank) return leftRank - rightRank;
  return left.localeCompare(right);
}
