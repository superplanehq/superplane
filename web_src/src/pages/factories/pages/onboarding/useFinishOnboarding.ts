import { onboardingBillingError } from "./onboardingBilling";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { useBYOKAccess } from "@/hooks/useBYOKAccess";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { resolveOnboardingAgent } from "./onboardingAgentReadiness";
import type {
  FactoriesFactory,
  FactoriesFactoryIntakeSettings,
  FactoriesFactoryLine,
  FactoriesFactoryLineStep as FactoryLineStep,
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
import { describeInstallationName, selectionsWithSavedVcsInstallation } from "./githubIntegrationSelection";
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
  type UpdateFactoryIntake,
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
  const intakeError = githubIntakeRepositoryError(args.issuesChoice, args.vcsHost);
  if (intakeError) return intakeError;
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
  skipInitialImport?: boolean;
  bypassAnalysis?: boolean;
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
  if (args.onProvisioned && !args.skipInitialImport && !args.bypassAnalysis) {
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
  updateIntake: UpdateFactoryIntake;
  deleteIntake: DeleteFactoryIntake;
  listApps: ListFactoryApps;
  workspaceName: string;
  takenNames: string[];
  appRepository: string;
  backlogRepository: string;
  issuesChoice: IssuesChoiceId | null;
  skipInitialImport?: boolean;
  /** Start the intake with automatic intake off. */
  paused?: boolean;
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
  const subscription = args.agentPlan.credentialsSource === "subscription";
  await args.updateOnboarding({
    vcsIntegrationId: args.vcs.id,
    agentIntegrationId: subscription ? "" : (args.agentIntegrationId ?? ""),
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    issuesSource: apiIssuesSource(args.issuesChoice),
    agentHarness: subscription ? "AGENT_HARNESS_CODEX" : args.agentPlan.harness,
    codexSubscription: subscription,
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
    updateIntake: args.updateIntake,
    deleteIntake: args.deleteIntake,
    issuesChoice: args.issuesChoice,
    skipInitialImport: args.skipInitialImport,
    paused: args.paused,
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

// Templates copy this name into secret references. The worker loads
// credentials by that name, so GitHub and Bitbucket both need the
// installation name, not the integration id.
async function selectionsWithVcsInstallation(
  organizationId: string,
  selections: IntegrationSelections,
  vcsHost: VcsHostId,
  integrationId: string,
): Promise<IntegrationSelections> {
  const installationName = await describeInstallationName(organizationId, integrationId);
  const name = installationName.trim();
  if (!name || name === integrationId.trim()) {
    throw new Error(`${vcsHost === "bitbucket" ? "Bitbucket" : "GitHub"} integration name is missing`);
  }
  return selectionsWithSavedVcsInstallation(
    { vcsIntegrationId: integrationId, vcsProvider: vcsHost },
    selections,
    name,
  );
}

function agentIntegrationIdForPlan(plan: OnboardingAgentPlan, selections: IntegrationSelections): string | undefined {
  if (plan.credentialsSource !== "integration" || !plan.integrationName) return undefined;
  return selections[plan.integrationName]?.id;
}

async function verifyOnboardingPlan(
  access: ReturnType<typeof useBYOKAccess>,
  usage: ReturnType<typeof useOrganizationWorkspaceUsage>,
  savedPlan: OnboardingAgentPlan | undefined,
) {
  if (savedPlan?.credentialsSource === "subscription") return { plan: savedPlan, bypassAnalysis: false };
  try {
    const byokAllowed = await access.verify();
    if (byokAllowed) return { plan: savedPlan, bypassAnalysis: false };
    const hosted = await usage.refetch();
    if (hosted.error || !hosted.data) throw new Error("SuperPlane could not load hosted models. Try again.");
    const plan = resolveOnboardingAgent({
      connected: new Set(),
      hostedModels: { anthropic: [], openai: [], openrouter: [] },
      defaultHostedProvider: hosted.data.defaultHostedProvider,
      defaultHostedModel: hosted.data.defaultHostedModel,
    });
    if (!plan)
      throw new Error(
        "No hosted default model is available. Ask your administrator to configure a SuperPlane-hosted model.",
      );
    return { plan, bypassAnalysis: true };
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, "SuperPlane could not verify model access. Try again."));
    return undefined;
  }
}

async function verifyOnboardingBilling(billing: ReturnType<typeof useOrganizationBilling>): Promise<void> {
  const currentBilling = await billing.refetch();
  if (currentBilling.error || !currentBilling.data) {
    throw new Error("SuperPlane could not verify billing. Try again.");
  }
  const error = onboardingBillingError(currentBilling.data);
  if (error) throw new Error(error);
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
  updateIntake: UpdateFactoryIntake;
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
  const billing = useOrganizationBilling(args.organizationId);
  const access = useBYOKAccess(args.organizationId);
  const usage = useOrganizationWorkspaceUsage(args.organizationId);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  // The first workspace in an organization always imports: that batch is
  // what the onboarding analysis and the first tasks are built from.
  const skipInitialImport = !args.factory?.onboarding?.initial && args.setup.skipInitialImport;
  // Only an explicit "off" pauses the intake.
  const paused = args.setup.automaticIntake === false;
  // A caller that just changed the issues answer in the same click (the
  // ticket screen's Analyze action) passes it here instead of reading
  // `args.setup.issuesChoice`. That value comes from a render captured before
  // the click, so it would still read the answer the user had before this
  // click, and provisioning would save that stale (often empty) answer over
  // the one `saveIssues` already stored.
  return async (issuesChoiceOverride?: IssuesChoiceId) => {
    const verified = await verifyOnboardingPlan(access, usage, args.plan);
    if (!verified) return;
    const { plan, bypassAnalysis } = verified;
    const appRepository = args.setup.selectedRepo ?? "";
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
      vcsReady: Boolean(vcs?.id),
      vcsHost,
      remainingCreditCents: args.remainingCreditCents,
      hostedModelsLoading: args.hostedModelsLoading,
      plan,
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
    if (!plan || !vcs?.id) {
      return;
    }

    args.setSaving(true);
    try {
      await verifyOnboardingBilling(billing);
      const selections = await selectionsWithVcsInstallation(args.organizationId, args.selections, vcsHost, vcs.id);
      const provisioned = await provisionWorkspace({
        ...args,
        selections,
        workspaceName,
        appRepository,
        backlogRepository,
        issuesChoice,
        skipInitialImport,
        paused,
        vcs: { id: vcs.id, provider: vcsHost },
        agentPlan: plan,
        agentRewrite: agentRewriteFromPlan(plan, selections),
        agentIntegrationId: agentIntegrationIdForPlan(plan, selections),
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
        skipInitialImport,
        bypassAnalysis,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to finish workspace setup"));
    } finally {
      args.setSaving(false);
    }
  };
}

function githubIntakeRepositoryError(choice: IssuesChoiceId | null | undefined, host: VcsHostId | null | undefined) {
  return choice === "vcs" && host === "bitbucket" ? "GitHub Issues requires a GitHub repository." : null;
}
