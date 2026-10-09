import type {
  FactoriesFactory,
  FactoriesFactoryIntakeSettings,
  FactoriesFactoryLine,
  FactoryLineStep,
} from "@/api-client";
import { accountOrganizationsQueryKey } from "@/hooks/useAccountOrganizations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

import { completeInitialOrganizationIdentity } from "./initialOnboardingOrganization";

import { factoryHomePath } from "../../lib/factoryPagePaths";
import { jiraCompletionSettingsToApi } from "../intakeSourceSettingsModel";
import type { JiraCompletionColumnValue } from "../jiraCompletionColumn";
import { describeGitHubInstallationName, selectionsWithGitHubInstallation } from "./githubIntegrationSelection";
import { markWorkspaceGettingStarted } from "./gettingStartedState";
import { firstWorkOrderAgentError, type OnboardingAgentPlan } from "./onboardingAgentReadiness";
import { vcsLabel, type IssuesChoiceId, type VcsHostId } from "./onboardingFixtures";
import {
  provisionEventApps,
  provisionOnboardingIntake,
  provisionLine,
  type CreateFactoryIntake,
  type DeleteFactoryIntake,
  type InstallOnboardingApp,
  type ListFactoryApps,
  type ListFactoryIntakes,
  type UpdateOnboarding,
} from "./onboardingProvision";
import { apiIssuesSource } from "./onboardingStatus";
import { saveWithFreeWorkspaceName } from "./uniqueFactoryName";
import { agentRewriteFromPlan } from "./useOnboardingAgentPlan";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export function finishOnboardingError(args: {
  appRepository: string | null;
  backlogRepository: string | null;
  workspaceName: string;
  vcsReady: boolean;
  vcsHost?: VcsHostId | null;
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
  issuesChoice?: IssuesChoiceId | null;
  jiraReady?: boolean;
  jiraProjectId?: string;
  linearReady?: boolean;
  linearProjectIds?: string[];
}): string | null {
  if (!args.vcsReady) {
    return `Connect ${vcsLabel(args.vcsHost ?? "github")} before you continue.`;
  }
  if (!args.appRepository || !args.backlogRepository) {
    return "Select the code repository and the backlog repository.";
  }
  if (args.issuesChoice === "jira" && (!args.jiraReady || !args.jiraProjectId)) {
    return "Connect Jira, then choose a project.";
  }
  if (args.issuesChoice === "linear" && (!args.linearReady || !args.linearProjectIds?.length)) {
    return "Connect Linear, then choose a project.";
  }
  const agentError = firstWorkOrderAgentError({
    remainingCreditCents: args.remainingCreditCents,
    hostedModelsLoading: args.hostedModelsLoading,
    plan: args.plan,
  });
  if (agentError) return agentError;
  if (!args.workspaceName) {
    return "Enter a workspace name.";
  }
  return null;
}

export type OnboardingDestination = { organizationId: string; factoryKey: string; lineId: string };

/** The line board, where the new GitHub intake sits at the foot of Backlog. */
export function afterOnboardingPath(args: OnboardingDestination) {
  return factoryHomePath(args.organizationId, args.factoryKey, args.lineId);
}

function navigateAfterFinish(
  navigate: ReturnType<typeof useNavigate>,
  organizationId: string,
  factoryKey: string,
  lineId: string,
) {
  navigate(afterOnboardingPath({ organizationId, factoryKey, lineId }), { replace: true });
}

export async function afterWorkspaceProvisioned(args: {
  factory: FactoriesFactory | null;
  owner?: string;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId: string;
  updateOrganization?: (identity: { name: string; slug: string }) => Promise<string | undefined>;
  invalidateAccountOrganizations: () => void;
  navigate: ReturnType<typeof useNavigate>;
  onProvisioned?: (destination: OnboardingDestination) => void;
}): Promise<void> {
  let organizationId = args.organizationId;
  if (args.updateOrganization) {
    try {
      organizationId = await completeInitialOrganizationIdentity({
        factory: args.factory,
        owner: args.owner,
        currentSlug: args.organizationId,
        update: args.updateOrganization,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Could not name the organization from the GitHub connection"));
    }
  }
  args.invalidateAccountOrganizations();
  markWorkspaceGettingStarted(organizationId, args.factoryId);
  const destination = { organizationId, factoryKey: args.factoryKey, lineId: args.lineId };
  if (args.onProvisioned) {
    args.onProvisioned(destination);
    return;
  }
  navigateAfterFinish(args.navigate, organizationId, args.factoryKey, args.lineId);
}

export async function provisionWorkspace(args: {
  organizationId: string;
  factoryId: string;
  factory: FactoriesFactory | null;
  selections: IntegrationSelections;
  updateFactory: (input: { name: string }) => Promise<unknown>;
  updateOnboarding: UpdateOnboarding;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
  deleteIntake: DeleteFactoryIntake;
  listApps: ListFactoryApps;
  workspaceName: string;
  takenNames: string[];
  appRepository: string;
  backlogRepository: string;
  issuesChoice: IssuesChoiceId | null;
  resolveDefaultBranch: (repository: string) => Promise<string>;
  vcs: { id: string; provider: VcsHostId };
  agentPlan: OnboardingAgentPlan;
  agentRewrite: FactoryAgentRewrite;
  agentIntegrationId?: string;
  jira?: { integrationId: string; projectId: string; settings?: FactoriesFactoryIntakeSettings };
  linear?: { integrationId: string; projectIds: string[] };
}): Promise<{ lineId: string }> {
  if (args.workspaceName !== args.factory?.name) {
    await saveWithFreeWorkspaceName({
      name: args.workspaceName,
      takenNames: args.takenNames,
      save: (name) => args.updateFactory({ name }),
    });
  }
  await args.updateOnboarding({
    vcsIntegrationId: args.vcs.id,
    ...(args.agentIntegrationId ? { agentIntegrationId: args.agentIntegrationId } : {}),
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    issuesSource: apiIssuesSource(args.issuesChoice),
    agentHarness: args.agentPlan.harness,
  });
  // Onboarding installs the Implement app with the real default branch (main,
  // master, staging, ...) instead of hardcoding "main", so Create Branch and
  // Create Pull Request target the branch GitHub actually treats as default.
  const defaultBranch = (await args.resolveDefaultBranch(args.appRepository)) || "main";
  await args.updateOnboarding({ defaultBranch });
  const { lineId, primaryAppId } = await provisionLine({
    factory: args.factory,
    savedLineId: args.factory?.onboarding?.provisionedLineId,
    savedAppId: args.factory?.onboarding?.provisionedAppId,
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    defaultBranch,
    agentRewrite: args.agentRewrite,
    vcsProvider: args.vcs.provider,
    installFactory: args.installFactory,
    createLine: args.createLine,
    updateOnboarding: args.updateOnboarding,
  });
  await provisionEventApps({
    factoryId: args.factoryId,
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    defaultBranch,
    agentRewrite: args.agentRewrite,
    vcsProvider: args.vcs.provider,
    installFactory: args.installFactory,
    listApps: args.listApps,
  });
  // The intake needs the line: it opens tasks that the line runs.
  await provisionOnboardingIntake({
    vcsProvider: args.vcs.provider,
    listIntakes: args.listIntakes,
    createIntake: args.createIntake,
    deleteIntake: args.deleteIntake,
    issuesChoice: args.issuesChoice,
    jira: args.jira,
    linear: args.linear,
  });
  await args.updateOnboarding({
    provisionedAppId: primaryAppId,
    provisionedLineId: lineId,
    complete: true,
  });
  return { lineId };
}

function linearIntakeBinding(
  issuesChoice: IssuesChoiceId | null,
  linearId: string | undefined,
  projectIds: string[] | undefined,
): { integrationId: string; projectIds: string[] } | undefined {
  if (issuesChoice !== "linear" || !linearId || !projectIds?.length) return undefined;
  return { integrationId: linearId, projectIds };
}

function jiraIntakeBinding(
  issuesChoice: IssuesChoiceId | null,
  jiraId: string | undefined,
  projectId: string | undefined,
  completion?: JiraCompletionColumnValue,
): { integrationId: string; projectId: string; settings?: FactoriesFactoryIntakeSettings } | undefined {
  if (issuesChoice !== "jira" || !jiraId || !projectId) return undefined;
  return {
    integrationId: jiraId,
    projectId,
    settings: jiraCompletionSettingsToApi(completion ?? { jiraMoveOnComplete: true, jiraCompletionColumn: "" }),
  };
}

function workspaceVcsHost(host: VcsHostId | null): VcsHostId {
  return host === "bitbucket" ? "bitbucket" : "github";
}

// A Bitbucket selection already carries the integration name. A GitHub
// selection names the installation, which templates need.
async function selectionsWithVcsInstallation(
  organizationId: string,
  selections: IntegrationSelections,
  vcsHost: VcsHostId,
  integrationId: string,
): Promise<IntegrationSelections> {
  if (vcsHost !== "github") return selections;
  const installationName = await describeGitHubInstallationName(organizationId, integrationId);
  return selectionsWithGitHubInstallation(selections, installationName);
}

function agentIntegrationIdForPlan(plan: OnboardingAgentPlan, selections: IntegrationSelections): string | undefined {
  if (plan.credentialsSource !== "integration" || !plan.integrationName) return undefined;
  return selections[plan.integrationName]?.id;
}

export function useFinishOnboarding(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  setup: OnboardingSetupApi;
  selections: IntegrationSelections;
  setSaving: (saving: boolean) => void;
  updateFactory: (input: { name: string }) => Promise<unknown>;
  updateOnboarding: UpdateOnboarding;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
  deleteIntake: DeleteFactoryIntake;
  listApps: ListFactoryApps;
  resolveDefaultBranch: (repository: string) => Promise<string>;
  takenNames: string[];
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
  githubOwner?: string;
  jiraProjectId?: string;
  jiraCompletion?: JiraCompletionColumnValue;
  linearProjectIds?: string[];
  updateOrganization?: (identity: { name: string; slug: string }) => Promise<string | undefined>;
  onProvisioned?: (destination: OnboardingDestination) => void;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  // A caller that just changed the issues answer in the same click (the
  // ticket screen's Analyze action) passes it here instead of reading
  // `args.setup.issuesChoice`. That value comes from a render captured before
  // the click, so it would still read the answer the user had before this
  // click, and provisioning would save that stale (often empty) answer over
  // the one `saveIssues` already stored.
  return async (issuesChoiceOverride?: IssuesChoiceId) => {
    const appRepository = args.setup.selectedRepo;
    const backlogRepository = args.setup.issuesRepo ?? appRepository;
    const workspaceName = args.setup.workspaceName.trim();
    const issuesChoice = issuesChoiceOverride ?? args.setup.issuesChoice;
    const vcsHost = workspaceVcsHost(args.setup.vcsHost);
    const vcs = args.selections[vcsHost];
    const jira = args.selections.jira;
    const linear = args.selections.linear;
    const error = finishOnboardingError({
      appRepository,
      backlogRepository,
      workspaceName,
      vcsReady: Boolean(vcs?.ready),
      vcsHost,
      remainingCreditCents: args.remainingCreditCents,
      hostedModelsLoading: args.hostedModelsLoading,
      plan: args.plan,
      issuesChoice,
      jiraReady: Boolean(jira?.ready),
      jiraProjectId: args.jiraProjectId,
      linearReady: Boolean(linear?.ready),
      linearProjectIds: args.linearProjectIds,
    });
    if (error) {
      showErrorToast(error);
      return;
    }
    if (!appRepository || !backlogRepository || !vcs?.ready || !args.plan) {
      return;
    }

    args.setSaving(true);
    try {
      const selections = await selectionsWithVcsInstallation(args.organizationId, args.selections, vcsHost, vcs.id);
      const provisioned = await provisionWorkspace({
        ...args,
        selections,
        workspaceName,
        appRepository,
        backlogRepository,
        issuesChoice,
        vcs: { id: vcs.id, provider: vcsHost },
        agentPlan: args.plan,
        agentRewrite: agentRewriteFromPlan(args.plan, selections),
        agentIntegrationId: agentIntegrationIdForPlan(args.plan, selections),
        jira: jiraIntakeBinding(issuesChoice, jira?.id, args.jiraProjectId, args.jiraCompletion),
        linear: linearIntakeBinding(issuesChoice, linear?.id, args.linearProjectIds),
      });
      await afterWorkspaceProvisioned({
        factory: args.factory,
        owner: args.githubOwner,
        organizationId: args.organizationId,
        factoryId: args.factoryId,
        factoryKey: args.factoryKey,
        lineId: provisioned.lineId,
        updateOrganization: args.updateOrganization,
        invalidateAccountOrganizations: () => {
          void queryClient.invalidateQueries({ queryKey: accountOrganizationsQueryKey });
        },
        navigate,
        onProvisioned: args.onProvisioned,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to finish workspace setup"));
    } finally {
      args.setSaving(false);
    }
  };
}
