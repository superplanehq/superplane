import type { FactoriesFactory } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useState } from "react";

import {
  describeInstallationName,
  githubIntegrationSelection,
  selectionsWithSavedVcsInstallation,
} from "./githubIntegrationSelection";
import type { VcsHostId } from "./onboardingFixtures";

type FactorySelection = {
  onboarding?: {
    defaultBranch?: string | null;
    vcsIntegrationId?: string | null;
  } | null;
};

type SetSelections = (update: (current: IntegrationSelections) => IntegrationSelections) => void;

export type OnboardingRepositorySelection = {
  organizationId: string;
  setSaving: (saving: boolean) => void;
  selectGitHubRepository: (repositoryId: string) => Promise<FactorySelection>;
  selectBitbucketForgeRepository: (input: { repository: string }) => Promise<FactorySelection>;
  updateOnboarding: (input: {
    vcsIntegrationId: string;
    appRepository: string;
    backlogRepository: string;
  }) => Promise<FactorySelection>;
  bitbucketIntegrationId: string;
  rememberPreferredInstance: (integrationName: string, integrationId: string) => void;
  refetchConnections: () => unknown;
  setSelections: SetSelections;
  selectVcsHost: (host: VcsHostId) => void;
  selectRepo: (repository: string) => void;
};

export async function runSave(setSaving: (saving: boolean) => void, action: () => Promise<unknown>): Promise<boolean> {
  setSaving(true);
  try {
    await action();
    return true;
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "Failed to save workspace setup"));
    return false;
  } finally {
    setSaving(false);
  }
}

function savedBranchResult(saved: boolean, branch: string): boolean | "needs-branch" {
  if (!saved) return false;
  return branch ? true : "needs-branch";
}

function onboardingDefaultBranchName(factory: FactorySelection | undefined) {
  return factory?.onboarding?.defaultBranch?.trim() ?? "";
}

async function rememberBitbucketInstallation(
  organizationId: string,
  integrationId: string,
  setSelections: SetSelections,
) {
  const installationName = await describeInstallationName(organizationId, integrationId);
  if (!installationName.trim() || installationName.trim() === integrationId.trim()) {
    throw new Error("Bitbucket integration name is missing");
  }
  setSelections((current) =>
    selectionsWithSavedVcsInstallation(
      { vcsIntegrationId: integrationId, vcsProvider: "bitbucket" },
      current,
      installationName,
    ),
  );
}

export async function saveCatalogRepository(
  deps: OnboardingRepositorySelection,
  repository: { repositoryId?: string; fullName?: string },
): Promise<boolean | "needs-branch"> {
  if (!repository.repositoryId || !repository.fullName) return false;
  const repositoryId = repository.repositoryId;
  const fullName = repository.fullName;
  let branch = "";
  const saved = await runSave(deps.setSaving, async () => {
    const factory = await deps.selectGitHubRepository(repositoryId);
    branch = onboardingDefaultBranchName(factory);
    const integrationId = factory.onboarding?.vcsIntegrationId;
    if (!integrationId) throw new Error("GitHub repository selection returned no integration");
    const installationName = await describeInstallationName(deps.organizationId, integrationId);
    deps.rememberPreferredInstance("github", integrationId);
    await deps.refetchConnections();
    deps.setSelections((current) => ({
      ...current,
      github: githubIntegrationSelection(integrationId, installationName),
    }));
    deps.selectVcsHost("github");
    deps.selectRepo(fullName);
  });
  return savedBranchResult(saved, branch);
}

export async function saveBitbucketForgeRepository(
  deps: OnboardingRepositorySelection,
  fullName: string,
): Promise<boolean | "needs-branch"> {
  if (!fullName) return false;
  let branch = "";
  const saved = await runSave(deps.setSaving, async () => {
    const factory = await deps.selectBitbucketForgeRepository({ repository: fullName });
    branch = onboardingDefaultBranchName(factory);
    const integrationId = factory.onboarding?.vcsIntegrationId;
    if (!integrationId) throw new Error("Bitbucket repository selection returned no integration");
    deps.rememberPreferredInstance("bitbucket", integrationId);
    await deps.refetchConnections();
    await rememberBitbucketInstallation(deps.organizationId, integrationId, deps.setSelections);
    deps.selectVcsHost("bitbucket");
    deps.selectRepo(fullName);
  });
  return savedBranchResult(saved, branch);
}

export async function saveBitbucketRepository(
  deps: OnboardingRepositorySelection,
  fullName: string,
): Promise<boolean | "needs-branch"> {
  if (!deps.bitbucketIntegrationId || !fullName) return false;
  let branch = "";
  const saved = await runSave(deps.setSaving, async () => {
    const factory = await deps.updateOnboarding({
      vcsIntegrationId: deps.bitbucketIntegrationId,
      appRepository: fullName,
      backlogRepository: fullName,
    });
    branch = onboardingDefaultBranchName(factory);
    deps.selectVcsHost("bitbucket");
    deps.selectRepo(fullName);
  });
  return savedBranchResult(saved, branch);
}

/** The repository step asks for a branch only when none is known yet. */
export function useDefaultBranchGate(
  setSaving: (saving: boolean) => void,
  updateOnboarding: (input: { defaultBranch: string }) => Promise<FactoriesFactory>,
) {
  const [defaultBranchRequired, setDefaultBranchRequired] = useState(false);
  const [chosenDefaultBranch, setChosenDefaultBranch] = useState("");
  const requireDefaultBranch = () => setDefaultBranchRequired(true);
  const saveChosenDefaultBranch = (branch: string) => {
    const value = branch.trim();
    if (!value) return Promise.resolve(false);
    return runSave(setSaving, () => updateOnboarding({ defaultBranch: value }));
  };
  return {
    onMissingDefaultBranch: requireDefaultBranch,
    model: {
      defaultBranchRequired,
      chosenDefaultBranch,
      setChosenDefaultBranch,
      requireDefaultBranch,
      clearDefaultBranchRequirement: () => setDefaultBranchRequired(false),
      saveChosenDefaultBranch,
    },
  };
}
