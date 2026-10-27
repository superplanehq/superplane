import type { IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import {
  describeInstallationName,
  githubIntegrationSelection,
  selectionsWithSavedVcsInstallation,
} from "./githubIntegrationSelection";
import { githubEnterpriseOnboardingActions } from "./onboardingGithubEnterprise";
import type { VcsHostId } from "./onboardingFixtures";
import type { UpdateOnboarding } from "./onboardingProvision";

export async function rememberBitbucketInstallation(
  organizationId: string,
  integrationId: string,
  setSelections: (update: (current: IntegrationSelections) => IntegrationSelections) => void,
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

export function vcsOnboardingRepositorySaves(args: {
  organizationId: string;
  factoryId: string;
  githubIntegrationId: string;
  bitbucketIntegrationId: string;
  setSaving: (saving: boolean) => void;
  updateOnboarding: UpdateOnboarding;
  selectGitHubRepository: (repositoryId: string) => Promise<{ onboarding?: { vcsIntegrationId?: string } }>;
  selectBitbucketForgeRepository: (input: {
    repository: string;
  }) => Promise<{ onboarding?: { vcsIntegrationId?: string } }>;
  rememberPreferredInstance: (integrationName: string, integrationId: string) => void;
  refetchConnections: () => Promise<unknown>;
  setSelections: (update: (current: IntegrationSelections) => IntegrationSelections) => void;
  selectVcsHost: (host: VcsHostId) => void;
  selectRepo: (repository: string) => void;
  runSave: (setSaving: (saving: boolean) => void, action: () => Promise<unknown>) => Promise<boolean>;
}) {
  return {
    selectCatalogRepository: async (repository: { repositoryId?: string; fullName?: string }): Promise<boolean> => {
      if (!repository.repositoryId || !repository.fullName) return false;
      const repositoryId = repository.repositoryId;
      const fullName = repository.fullName;
      return args.runSave(args.setSaving, async () => {
        const factory = await args.selectGitHubRepository(repositoryId);
        const integrationId = factory.onboarding?.vcsIntegrationId;
        if (!integrationId) throw new Error("GitHub repository selection returned no integration");
        const installationName = await describeInstallationName(args.organizationId, integrationId);
        args.rememberPreferredInstance("github", integrationId);
        await args.refetchConnections();
        args.setSelections((current) => ({
          ...current,
          github: githubIntegrationSelection(integrationId, installationName),
        }));
        args.selectVcsHost("github");
        args.selectRepo(fullName);
      });
    },
    selectBitbucketForgeRepository: async (fullName: string): Promise<boolean> => {
      if (!fullName) return false;
      return args.runSave(args.setSaving, async () => {
        const factory = await args.selectBitbucketForgeRepository({ repository: fullName });
        const integrationId = factory.onboarding?.vcsIntegrationId;
        if (!integrationId) throw new Error("Bitbucket repository selection returned no integration");
        args.rememberPreferredInstance("bitbucket", integrationId);
        await args.refetchConnections();
        await rememberBitbucketInstallation(args.organizationId, integrationId, args.setSelections);
        args.selectVcsHost("bitbucket");
        args.selectRepo(fullName);
      });
    },
    selectBitbucketRepository: async (fullName: string): Promise<boolean> => {
      if (!args.bitbucketIntegrationId || !fullName) return false;
      return args.runSave(args.setSaving, async () => {
        await args.updateOnboarding({
          vcsIntegrationId: args.bitbucketIntegrationId,
          appRepository: fullName,
          backlogRepository: fullName,
        });
        args.selectVcsHost("bitbucket");
        args.selectRepo(fullName);
      });
    },
    ...githubEnterpriseOnboardingActions(args),
  };
}
