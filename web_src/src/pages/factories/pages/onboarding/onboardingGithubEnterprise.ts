import type { IntegrationSelections } from "@/pages/home/homeIntegrationStatus";

import { describeInstallationName, selectionsWithSavedVcsInstallation } from "./githubIntegrationSelection";
import type { VcsHostId } from "./onboardingFixtures";
import type { UpdateOnboarding } from "./onboardingProvision";

const FLAG_PREFIX = "superplane:onboarding-github-enterprise";
const HOST_PREFIX = "superplane:onboarding-github-enterprise-host";
const INTEGRATION_PREFIX = "superplane:onboarding-github-enterprise-integration";

function flagKey(factoryId: string): string {
  return `${FLAG_PREFIX}:${factoryId}`;
}

function hostKey(factoryId: string): string {
  return `${HOST_PREFIX}:${factoryId}`;
}

function integrationKey(factoryId: string): string {
  return `${INTEGRATION_PREFIX}:${factoryId}`;
}

export function readOnboardingGithubEnterprise(factoryId: string): {
  enabled: boolean;
  host: string;
  integrationId: string;
} {
  if (!factoryId) {
    return { enabled: false, host: "", integrationId: "" };
  }
  return {
    enabled: sessionStorage.getItem(flagKey(factoryId)) === "1",
    host: sessionStorage.getItem(hostKey(factoryId)) ?? "",
    integrationId: sessionStorage.getItem(integrationKey(factoryId)) ?? "",
  };
}

export function sessionGithubEnterpriseChoice(factoryId: string): "enterprise" | "dotcom" | "" {
  if (!factoryId) {
    return "";
  }
  const value = sessionStorage.getItem(flagKey(factoryId));
  if (value === "1") {
    return "enterprise";
  }
  if (value === "0") {
    return "dotcom";
  }
  return "";
}

export function resolvedOnboardingGithubEnterprise(args: {
  factoryId: string;
  vcsIntegrationId?: string;
  hostFromIntegration?: string;
}): { enabled: boolean; host: string; integrationId: string } {
  const remembered = readOnboardingGithubEnterprise(args.factoryId);
  const choice = sessionGithubEnterpriseChoice(args.factoryId);
  if (choice === "enterprise") {
    return remembered;
  }
  if (choice === "dotcom") {
    return { enabled: false, host: "", integrationId: "" };
  }
  const host = args.hostFromIntegration?.trim() ?? "";
  const integrationId = args.vcsIntegrationId?.trim() ?? "";
  if (!host || !integrationId) {
    return { enabled: false, host: "", integrationId: "" };
  }
  return { enabled: true, host, integrationId };
}

export function writeOnboardingGithubEnterprise(
  factoryId: string,
  enabled: boolean,
  host = "",
  integrationId?: string,
): void {
  if (!factoryId) {
    return;
  }
  if (!enabled) {
    sessionStorage.setItem(flagKey(factoryId), "0");
    sessionStorage.removeItem(hostKey(factoryId));
    sessionStorage.removeItem(integrationKey(factoryId));
    return;
  }
  sessionStorage.setItem(flagKey(factoryId), "1");
  const nextHost = host.trim();
  if (nextHost) {
    sessionStorage.setItem(hostKey(factoryId), nextHost);
  }
  if (integrationId === undefined) {
    return;
  }
  const nextIntegrationId = integrationId.trim();
  if (nextIntegrationId) {
    sessionStorage.setItem(integrationKey(factoryId), nextIntegrationId);
    return;
  }
  sessionStorage.removeItem(integrationKey(factoryId));
}

export function vcsOnboardingIntegrationIds(args: {
  githubSelectionId?: string;
  bitbucketReady?: boolean;
  bitbucketId?: string;
  githubId?: string;
  vcsHost?: string | null;
  vcsProvider?: string;
  vcsIntegrationId?: string;
  rememberedIntegrationId?: string;
}) {
  const bitbucketIntegrationId = args.bitbucketReady ? (args.bitbucketId ?? "") : "";
  const branchIntegrationId = args.vcsHost === "bitbucket" ? (args.bitbucketId ?? "") : (args.githubId ?? "");
  return {
    githubIntegrationId: githubOnboardingIntegrationId({
      selectionId: args.githubSelectionId,
      vcsProvider: args.vcsProvider,
      vcsIntegrationId: args.vcsIntegrationId,
      rememberedIntegrationId: args.rememberedIntegrationId,
    }),
    bitbucketIntegrationId,
    branchIntegrationId,
  };
}

export function githubOnboardingIntegrationId(args: {
  selectionId?: string;
  vcsProvider?: string;
  vcsIntegrationId?: string;
  rememberedIntegrationId?: string;
}): string {
  const selected = args.selectionId?.trim() ?? "";
  if (selected) {
    return selected;
  }
  if (args.vcsProvider === "bitbucket") {
    return "";
  }
  const fromOnboarding = args.vcsIntegrationId?.trim() ?? "";
  if (fromOnboarding) {
    return fromOnboarding;
  }
  return args.rememberedIntegrationId?.trim() ?? "";
}

export async function persistGitHubEnterpriseRepository(args: {
  fullName: string;
  githubIntegrationId: string;
  updateOnboarding: UpdateOnboarding;
  selectVcsHost: (host: VcsHostId) => void;
  selectRepo: (repository: string) => void;
}): Promise<void> {
  await args.updateOnboarding({
    vcsIntegrationId: args.githubIntegrationId,
    appRepository: args.fullName,
    backlogRepository: args.fullName,
  });
  args.selectVcsHost("github");
  args.selectRepo(args.fullName);
}

export async function bindGitHubEnterpriseOnboarding(args: {
  organizationId: string;
  factoryId: string;
  integrationId: string;
  host: string;
  rememberPreferredInstance: (integrationName: string, integrationId: string) => void;
  refetchConnections: () => Promise<unknown>;
  setSelections: (update: (current: IntegrationSelections) => IntegrationSelections) => void;
  selectVcsHost: (host: VcsHostId) => void;
}): Promise<void> {
  const id = args.integrationId.trim();
  if (!id) {
    return;
  }
  args.rememberPreferredInstance("github", id);
  await args.refetchConnections();
  const installationName = await describeInstallationName(args.organizationId, id);
  args.setSelections((current) => githubEnterpriseSelections(current, id, installationName));
  writeOnboardingGithubEnterprise(args.factoryId, true, args.host, id);
  args.selectVcsHost("github");
}

export function githubEnterpriseOnboardingActions(args: {
  organizationId: string;
  factoryId: string;
  githubIntegrationId: string;
  setSaving: (saving: boolean) => void;
  updateOnboarding: UpdateOnboarding;
  rememberPreferredInstance: (integrationName: string, integrationId: string) => void;
  refetchConnections: () => Promise<unknown>;
  setSelections: (update: (current: IntegrationSelections) => IntegrationSelections) => void;
  selectVcsHost: (host: VcsHostId) => void;
  selectRepo: (repository: string) => void;
  runSave: (setSaving: (saving: boolean) => void, action: () => Promise<unknown>) => Promise<boolean>;
}) {
  return {
    selectGitHubEnterpriseRepository: (fullName: string) => {
      if (!args.githubIntegrationId || !fullName) return Promise.resolve(false);
      return args.runSave(args.setSaving, () =>
        persistGitHubEnterpriseRepository({
          fullName,
          githubIntegrationId: args.githubIntegrationId,
          updateOnboarding: args.updateOnboarding,
          selectVcsHost: args.selectVcsHost,
          selectRepo: args.selectRepo,
        }),
      );
    },
    bindGitHubEnterpriseIntegration: (integrationId: string) =>
      bindGitHubEnterpriseOnboarding({
        organizationId: args.organizationId,
        factoryId: args.factoryId,
        integrationId,
        host: readOnboardingGithubEnterprise(args.factoryId).host,
        rememberPreferredInstance: args.rememberPreferredInstance,
        refetchConnections: args.refetchConnections,
        setSelections: args.setSelections,
        selectVcsHost: args.selectVcsHost,
      }),
  };
}

function githubEnterpriseSelections(
  current: IntegrationSelections,
  integrationId: string,
  installationName: string,
): IntegrationSelections {
  const named = selectionsWithSavedVcsInstallation(
    { vcsIntegrationId: integrationId, vcsProvider: "github" },
    current,
    installationName,
  );
  if (named.github?.id === integrationId && named.github.ready) {
    return named;
  }
  return {
    ...current,
    github: { id: integrationId, name: integrationId, ready: false },
  };
}
