import type { FactoriesFactory, FactoriesFactoryLine, FactoryLineStep } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useNavigate } from "react-router";

import { factoryHomePath } from "../../lib/factoryPagePaths";
import { markWorkspaceGettingStarted } from "./gettingStartedState";
import { firstWorkOrderAgentError, type OnboardingAgentPlan } from "./onboardingAgentReadiness";
import {
  provisionEventApps,
  provisionGithubIntake,
  provisionLine,
  provisionPRFeedbackHandler,
  type CreateFactoryIntake,
  type CreateFactoryPRFeedbackHandler,
  type InstallOnboardingApp,
  type ListFactoryIntakes,
  type ListFactoryPRFeedbackHandlers,
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
  githubReady: boolean;
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
}): string | null {
  if (!args.appRepository || !args.backlogRepository || !args.githubReady) {
    return "Connect GitHub, then select both repositories.";
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

/** The line board, where the new GitHub intake sits at the foot of Backlog. */
export function afterOnboardingPath(args: { organizationId: string; factoryKey: string; lineId: string }) {
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

async function provisionWorkspace(args: {
  organizationId: string;
  factoryId: string;
  factory: FactoriesFactory | null;
  setup: OnboardingSetupApi;
  selections: IntegrationSelections;
  updateFactory: (input: { name: string }) => Promise<unknown>;
  updateOnboarding: UpdateOnboarding;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  listIntakes: ListFactoryIntakes;
  createIntake: CreateFactoryIntake;
  listPRFeedbackHandlers: ListFactoryPRFeedbackHandlers;
  createPRFeedbackHandler: CreateFactoryPRFeedbackHandler;
  workspaceName: string;
  takenNames: string[];
  appRepository: string;
  backlogRepository: string;
  github: { id: string };
  agentPlan: OnboardingAgentPlan;
  agentRewrite: FactoryAgentRewrite;
  agentIntegrationId?: string;
}): Promise<{ lineId: string }> {
  if (args.workspaceName !== args.factory?.name) {
    await saveWithFreeWorkspaceName({
      name: args.workspaceName,
      takenNames: args.takenNames,
      save: (name) => args.updateFactory({ name }),
    });
  }
  await args.updateOnboarding({
    vcsIntegrationId: args.github.id,
    ...(args.agentIntegrationId ? { agentIntegrationId: args.agentIntegrationId } : {}),
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    issuesSource: apiIssuesSource(args.setup.issuesChoice),
    agentHarness: args.agentPlan.harness,
  });
  const { lineId, primaryAppId } = await provisionLine({
    factory: args.factory,
    savedLineId: args.factory?.onboarding?.provisionedLineId,
    savedAppId: args.factory?.onboarding?.provisionedAppId,
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    agentRewrite: args.agentRewrite,
    installFactory: args.installFactory,
    createLine: args.createLine,
    updateOnboarding: args.updateOnboarding,
  });
  await provisionEventApps({
    factoryId: args.factoryId,
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    agentRewrite: args.agentRewrite,
    installFactory: args.installFactory,
  });
  // The intake needs the line: it opens tasks that the line runs.
  await provisionGithubIntake({
    listIntakes: args.listIntakes,
    createIntake: args.createIntake,
  });
  await provisionPRFeedbackHandler({
    listHandlers: args.listPRFeedbackHandlers,
    createHandler: args.createPRFeedbackHandler,
    repository: args.appRepository,
  });
  await args.updateOnboarding({
    provisionedAppId: primaryAppId,
    provisionedLineId: lineId,
    complete: true,
  });
  return { lineId };
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
  listPRFeedbackHandlers: ListFactoryPRFeedbackHandlers;
  createPRFeedbackHandler: CreateFactoryPRFeedbackHandler;
  takenNames: string[];
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
}) {
  const navigate = useNavigate();
  return async () => {
    const appRepository = args.setup.selectedRepo;
    const backlogRepository = args.setup.issuesRepo ?? appRepository;
    const workspaceName = args.setup.workspaceName.trim();
    const github = args.selections.github;
    const error = finishOnboardingError({
      appRepository,
      backlogRepository,
      workspaceName,
      githubReady: Boolean(github?.ready),
      remainingCreditCents: args.remainingCreditCents,
      hostedModelsLoading: args.hostedModelsLoading,
      plan: args.plan,
    });
    if (error) {
      showErrorToast(error);
      return;
    }
    if (!appRepository || !backlogRepository || !github?.ready || !args.plan) {
      return;
    }

    args.setSaving(true);
    try {
      const provisioned = await provisionWorkspace({
        ...args,
        workspaceName,
        appRepository,
        backlogRepository,
        github,
        agentPlan: args.plan,
        agentRewrite: agentRewriteFromPlan(args.plan, args.selections),
        agentIntegrationId:
          args.plan.credentialsSource === "integration" ? args.selections[args.plan.integrationName]?.id : undefined,
      });
      markWorkspaceGettingStarted(args.organizationId, args.factoryId);
      navigateAfterFinish(navigate, args.organizationId, args.factoryKey, provisioned.lineId);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to finish workspace setup"));
    } finally {
      args.setSaving(false);
    }
  };
}
