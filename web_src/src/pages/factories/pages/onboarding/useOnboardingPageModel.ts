import type { FactoriesFactory, FactoriesFactoryLine, FactoryLineStep } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactoryLine, useCreateWorkOrder, useDeleteFactory, useUpdateFactory } from "@/hooks/useFactoryData";
import { useDispatchWorkOrder } from "@/hooks/useFactoryData";
import { useHostedLLMModels } from "@/hooks/useHostedLLMModels";
import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import { getApiErrorMessage } from "@/lib/errors";
import { githubInstallationUrl } from "@/lib/githubInstallation";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { FactoryAgentRewrite } from "@/pages/home/factories";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import {
  factoryListPath,
  factoryOverviewPath,
  factorySetupPath,
  workOrderDetailPath,
} from "../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../lib/lastVisitedFactory";
import { markWorkspaceGettingStarted } from "./gettingStartedState";
import {
  AGENT_PROVIDER_IDS,
  firstWorkOrderAgentError,
  hostedModelsQueriesLoading,
  resolveOnboardingAgent,
  type OnboardingAgentPlan,
} from "./onboardingAgentReadiness";
import type { IntegrationId, WizardStepId } from "./onboardingFixtures";
import {
  apiIssuesSource,
  initialOnboardingSelections,
  initialWizardStep,
  isWizardStepId,
  localIssuesSource,
} from "./onboardingStatus";
import {
  DEFAULT_LINE_NAME,
  provisionEventApps,
  provisionLine,
  type InstallOnboardingApp,
  type UpdateOnboarding,
} from "./onboardingProvision";
import { createAndDispatchInitialWorkOrder } from "./onboardingWorkOrder";
import { useFactoryOnboarding } from "./useFactoryOnboarding";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { useOnboardingGithubConnections } from "./useSelectNewGithubConnection";
import { useFinishSetupAction } from "./useFinishSetupAction";

const ONBOARDING_INTEGRATIONS = ["github", ...AGENT_PROVIDER_IDS];

function useIntegrationSelections(onboarding: FactoriesFactory["onboarding"]) {
  const [selections, setSelections] = useState<IntegrationSelections>(() => initialOnboardingSelections(onboarding));
  const connected = useMemo(() => {
    const ready = new Set<IntegrationId>();
    if (selections.github?.ready) ready.add("github");
    for (const name of AGENT_PROVIDER_IDS) {
      if (selections[name]?.ready) ready.add(name);
    }
    return ready;
  }, [selections]);
  return { selections, connected, setSelections };
}

function useRestoreSetup(
  setup: OnboardingSetupApi,
  onboarding: FactoriesFactory["onboarding"],
  selections: IntegrationSelections,
) {
  useEffect(() => {
    if (selections.github?.ready && setup.vcsHost !== "github") setup.selectVcsHost("github");
  }, [selections.github?.ready, setup]);
  useEffect(() => {
    if (onboarding?.appRepository && setup.selectedRepo !== onboarding.appRepository) {
      setup.selectRepo(onboarding.appRepository);
    }
  }, [onboarding?.appRepository, setup]);
  useEffect(() => {
    if (onboarding?.backlogRepository && setup.issuesRepo !== onboarding.backlogRepository) {
      setup.selectIssuesRepo(onboarding.backlogRepository);
    }
  }, [onboarding?.backlogRepository, setup]);
  useEffect(() => {
    const source = localIssuesSource(onboarding?.issuesSource);
    if (source && setup.issuesChoice !== source) setup.setIssuesChoice(source);
  }, [onboarding?.issuesSource, setup]);
  useEffect(() => {
    if (!selections.claude?.ready) return;
    setup.setAgent("claude-code");
  }, [onboarding?.agentHarness, selections.claude?.ready, setup]);
}

async function runSave(setSaving: (saving: boolean) => void, action: () => Promise<unknown>): Promise<boolean> {
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

function useSectionSaves(args: {
  setup: OnboardingSetupApi;
  selections: IntegrationSelections;
  setSaving: (saving: boolean) => void;
  factoryName: string;
  updateFactory: (input: { name: string }) => Promise<unknown>;
  updateOnboarding: UpdateOnboarding;
}) {
  const saveName = () => {
    const workspaceName = args.setup.workspaceName.trim();
    if (!workspaceName) return Promise.resolve(false);
    if (workspaceName === args.factoryName) return Promise.resolve(true);
    return runSave(args.setSaving, () => args.updateFactory({ name: workspaceName }));
  };
  const saveRepository = (repository: string) => {
    const integrationId = args.selections.github?.id;
    if (!repository || !integrationId) return Promise.resolve(false);
    return runSave(args.setSaving, () =>
      args.updateOnboarding({
        vcsIntegrationId: integrationId,
        appRepository: repository,
      }),
    );
  };
  const saveIssues = () => {
    const backlogRepository = args.setup.issuesRepo ?? args.setup.selectedRepo;
    if (!backlogRepository || !args.setup.issuesChoice) return Promise.resolve(false);
    return runSave(args.setSaving, () =>
      args.updateOnboarding({
        backlogRepository,
        issuesSource: apiIssuesSource(args.setup.issuesChoice),
      }),
    );
  };
  return { saveName, saveRepository, saveIssues };
}

// Persists onboarding answers, provisions the line and event apps, then creates
// and dispatches the first work order. Kept out of the click handler so the
// handler stays focused on validation, saving state, and navigation.
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
  createWorkOrder: (input: {
    title: string;
    description: string;
  }) => Promise<{ id?: string | null; number?: number | string | null }>;
  dispatchWorkOrder: (input: { orderId: string; lineName: string }) => Promise<unknown>;
  workspaceName: string;
  appRepository: string;
  backlogRepository: string;
  workOrderTitle: string;
  workOrderDescription: string;
  github: { id: string };
  agentPlan: OnboardingAgentPlan;
  agentRewrite: FactoryAgentRewrite;
  agentIntegrationId?: string;
}): Promise<{ number?: number | string | null }> {
  if (args.workspaceName !== args.factory?.name) {
    await args.updateFactory({ name: args.workspaceName });
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
  await args.updateOnboarding({
    provisionedAppId: primaryAppId,
    provisionedLineId: lineId,
    complete: true,
  });
  return createAndDispatchInitialWorkOrder({
    title: args.workOrderTitle,
    description: args.workOrderDescription,
    lineName: DEFAULT_LINE_NAME,
    createWorkOrder: args.createWorkOrder,
    dispatchWorkOrder: args.dispatchWorkOrder,
  });
}

function navigateAfterFinish(
  navigate: ReturnType<typeof useNavigate>,
  organizationId: string,
  factoryKey: string,
  orderNumber: number | string | null | undefined,
) {
  if (orderNumber != null && orderNumber !== "") {
    navigate(workOrderDetailPath(organizationId, factoryKey, orderNumber), { replace: true });
    return;
  }
  navigate(factoryOverviewPath(organizationId, factoryKey), { replace: true });
}

function finishOnboardingError(args: {
  appRepository: string | null;
  backlogRepository: string | null;
  workspaceName: string;
  workOrderTitle: string;
  workOrderDescription: string;
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
  if (!args.workOrderTitle || !args.workOrderDescription) {
    return "Enter a work order title and description.";
  }
  return null;
}

function hostedModelIds(models: { id?: string | null }[] | undefined): string[] {
  return (models ?? []).map((model) => model.id ?? "").filter((id) => id !== "");
}

function useOnboardingAgentPlan(organizationId: string, connected: Set<IntegrationId>, remainingCreditCents: number) {
  const needHosted = remainingCreditCents > 0;
  const anthropic = useHostedLLMModels(organizationId, "anthropic", needHosted);
  const openai = useHostedLLMModels(organizationId, "openai", needHosted);
  const openrouter = useHostedLLMModels(organizationId, "openrouter", needHosted);
  return {
    remainingCreditCents,
    hostedModelsLoading: hostedModelsQueriesLoading(needHosted, [anthropic, openai, openrouter]),
    plan: resolveOnboardingAgent({
      connected,
      remainingCreditCents,
      hostedModels: {
        anthropic: hostedModelIds(anthropic.data?.models),
        openai: hostedModelIds(openai.data?.models),
        openrouter: hostedModelIds(openrouter.data?.models),
      },
    }),
  };
}

function agentRewriteFromPlan(plan: OnboardingAgentPlan, selections: IntegrationSelections): FactoryAgentRewrite {
  if (plan.credentialsSource === "hosted") {
    return { component: plan.component, model: plan.model, credentials: { source: "hosted" } };
  }
  return {
    component: plan.component,
    model: plan.model,
    credentials: {
      source: "integration",
      name: selections[plan.integrationName]?.name ?? plan.integrationName,
    },
  };
}

function useFinishOnboarding(args: {
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
  createWorkOrder: (input: {
    title: string;
    description: string;
  }) => Promise<{ id?: string | null; number?: number | string | null }>;
  dispatchWorkOrder: (input: { orderId: string; lineName: string }) => Promise<unknown>;
  remainingCreditCents: number;
  hostedModelsLoading: boolean;
  plan: OnboardingAgentPlan | undefined;
}) {
  const navigate = useNavigate();
  return async () => {
    const appRepository = args.setup.selectedRepo;
    const backlogRepository = args.setup.issuesRepo ?? appRepository;
    const workspaceName = args.setup.workspaceName.trim();
    const workOrderTitle = args.setup.workOrderTitle.trim();
    const workOrderDescription = args.setup.workOrderDescription.trim();
    const github = args.selections.github;
    const error = finishOnboardingError({
      appRepository,
      backlogRepository,
      workspaceName,
      workOrderTitle,
      workOrderDescription,
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
      const order = await provisionWorkspace({
        ...args,
        workspaceName,
        appRepository,
        backlogRepository,
        workOrderTitle,
        workOrderDescription,
        github,
        agentPlan: args.plan,
        agentRewrite: agentRewriteFromPlan(args.plan, args.selections),
        agentIntegrationId:
          args.plan.credentialsSource === "integration" ? args.selections[args.plan.integrationName]?.id : undefined,
      });
      markWorkspaceGettingStarted(args.organizationId, args.factoryId);
      navigateAfterFinish(navigate, args.organizationId, args.factoryKey, order.number);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to finish workspace setup"));
    } finally {
      args.setSaving(false);
    }
  };
}

function useCancelOnboarding(args: { organizationId: string; factoryId: string; canDelete: boolean }) {
  const { account } = useAccount();
  const navigate = useNavigate();
  const deleteFactory = useDeleteFactory(args.organizationId);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const cancelSetup = async () => {
    if (!args.canDelete) {
      showErrorToast("You do not have permission to delete this workspace.");
      throw new Error("Missing permission to delete workspace");
    }
    try {
      await deleteFactory.mutateAsync(args.factoryId);
      clearLastVisitedFactory(account?.id ?? "", args.organizationId, args.factoryId);
      showSuccessToast("Workspace deleted.");
      navigate(factoryListPath(args.organizationId), { replace: true });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to delete workspace"));
      throw error;
    }
  };

  return {
    canDeleteWorkspace: args.canDelete,
    deleteOpen,
    setDeleteOpen,
    deleting: deleteFactory.isPending,
    cancelSetup,
  };
}

function canConfigureWorkspace(canAct: (resource: string, action: string) => boolean): boolean {
  return (
    canAct("factories", "update") &&
    canAct("integrations", "create") &&
    canAct("canvases", "create") &&
    canAct("canvases", "update")
  );
}

function readyGithubIntegrationId(selections: IntegrationSelections): string {
  return selections.github?.ready ? selections.github.id : "";
}

function useOnboardingGithubRepos(organizationId: string, githubIntegrationId: string) {
  const githubIntegration = useIntegration(organizationId, githubIntegrationId);
  const resources = useIntegrationResources(organizationId, githubIntegrationId, "repository");
  const repositories = useMemo(
    () =>
      (resources.data ?? [])
        .map((resource) => resource.name ?? resource.id ?? "")
        .filter((repository): repository is string => Boolean(repository)),
    [resources.data],
  );
  return {
    githubIntegration,
    repositories,
    repositoriesLoading: resources.isLoading,
    repositoriesError: resources.error,
  };
}

export function useOnboardingPageModel(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
}) {
  const { canAct } = usePermissions();
  const onboarding = args.factory?.onboarding;
  const integrations = useIntegrationSelections(onboarding);
  const spend = useOrganizationLLMSpend(args.organizationId);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  const agent = useOnboardingAgentPlan(args.organizationId, integrations.connected, remainingCreditCents);
  const setup = useOnboardingSetupState(args.factory?.name ?? "", {
    connected: integrations.connected,
    remainingCreditCents: agent.remainingCreditCents,
    simulateDiscovery: false,
  });
  useRestoreSetup(setup, onboarding, integrations.selections);
  const [searchParams] = useSearchParams();
  const [openSection, setOpenSection] = useState<WizardStepId>(() => {
    const requestedStep = searchParams.get("step");
    return isWizardStepId(requestedStep) ? requestedStep : initialWizardStep(onboarding);
  });
  const connect = useIntegrationConnectDialog({
    organizationId: args.organizationId,
    // Carry the current step so the round trip to the provider returns here,
    // not at the first step. The provider redirects to the integration page,
    // and IntegrationSetupReturn then sends the browser to this path.
    // pick=newest asks the VCS step to select the connection just created.
    returnTo: `${factorySetupPath(args.organizationId, args.factoryKey)}?step=${openSection}${
      openSection === "vcs" ? "&pick=newest" : ""
    }`,
    integrationNames: ONBOARDING_INTEGRATIONS,
    selections: integrations.selections,
    onSelectionsChange: integrations.setSelections,
  });

  const [saving, setSaving] = useState(false);
  const updateFactory = useUpdateFactory(args.organizationId, args.factoryId);
  const updateOnboarding = useFactoryOnboarding(args.organizationId, args.factoryId);
  const createLine = useCreateFactoryLine(args.organizationId, args.factoryId);
  const createWorkOrder = useCreateWorkOrder(args.organizationId, args.factoryId);
  const dispatchWorkOrder = useDispatchWorkOrder(args.organizationId, args.factoryId);
  const installer = useInstallFactory();
  const githubIntegrationId = readyGithubIntegrationId(integrations.selections);
  const githubConnections = useOnboardingGithubConnections({
    integrationData: connect.integrationData,
    openSection,
    selectNewest: searchParams.get("pick") === "newest",
    selections: integrations.selections,
    selectInstance: connect.selectInstance,
    // The user left the wizard to connect GitHub, so the answer for the VCS
    // step is in. Open the repository step, the same as a connection the user
    // selects by hand. The host must be set here too, or the repository step
    // shows "Connect version control first" until the restore effect runs.
    onConnectionSelected: () => {
      setup.selectVcsHost("github");
      setOpenSection("repo");
    },
  });
  const githubRepos = useOnboardingGithubRepos(args.organizationId, githubIntegrationId);

  const saves = useSectionSaves({
    setup,
    selections: integrations.selections,
    setSaving,
    factoryName: args.factory?.name ?? "",
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
  });
  const finish = useFinishOnboarding({
    ...args,
    setup,
    selections: integrations.selections,
    setSaving,
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
    installFactory: installer.installFactory,
    createLine: createLine.mutateAsync,
    createWorkOrder: createWorkOrder.mutateAsync,
    dispatchWorkOrder: dispatchWorkOrder.mutateAsync,
    remainingCreditCents: agent.remainingCreditCents,
    hostedModelsLoading: agent.hostedModelsLoading,
    plan: agent.plan,
  });
  const cancel = useCancelOnboarding({
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    canDelete: canAct("factories", "delete"),
  });
  const finishSetup = useFinishSetupAction({
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    factoryKey: args.factoryKey,
    setup,
    finish,
  });

  return {
    setup,
    openSection,
    setOpenSection,
    requestConnect: connect.requestConnect,
    createVcsConnection: () => connect.createNew("github"),
    selectVcsConnection: (integrationId: string) => connect.selectInstance("github", integrationId),
    githubConnections,
    selectedVcsConnectionId: githubIntegrationId || undefined,
    requestConfigure: () => {
      // Manage which repositories the GitHub App can access, on GitHub itself.
      window.open(githubInstallationUrl(githubRepos.githubIntegration.data), "_blank", "noopener,noreferrer");
    },
    integrationDialogs: connect.dialogs,
    repositories: githubRepos.repositories,
    repositoriesLoading: githubRepos.repositoriesLoading,
    repositoriesError: githubRepos.repositoriesError,
    canConfigureWorkspace: canConfigureWorkspace(canAct),
    saving: saving || installer.isInstalling || createWorkOrder.isPending,
    ...saves,
    finish: finishSetup,
    ...cancel,
  };
}
