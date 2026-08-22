import type { FactoriesFactory } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactoryLine, useCreateWorkOrder, useDeleteFactory, useUpdateFactory } from "@/hooks/useFactoryData";
import { useDispatchWorkOrder } from "@/hooks/useFactoryData";
import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import { getApiErrorMessage } from "@/lib/errors";
import { githubInstallationUrl } from "@/lib/githubInstallation";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { factoryListPath, factorySetupPath } from "../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../lib/lastVisitedFactory";
import { AGENT_PROVIDER_IDS } from "./onboardingAgentReadiness";
import type { IntegrationId, WizardStepId } from "./onboardingFixtures";
import {
  apiIssuesSource,
  initialOnboardingSelections,
  initialWizardStep,
  isWizardStepId,
  localIssuesSource,
} from "./onboardingStatus";
import type { UpdateOnboarding } from "./onboardingProvision";
import { useFactoryOnboarding } from "./useFactoryOnboarding";
import { useFinishOnboarding } from "./useFinishOnboarding";
import { useFinishSetupAction } from "./useFinishSetupAction";
import { useOnboardingAgentPlan } from "./useOnboardingAgentPlan";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { useOnboardingGithubConnections } from "./useSelectNewGithubConnection";

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

function useOnboardingWorkspaceApis(organizationId: string, factoryId: string) {
  const updateFactory = useUpdateFactory(organizationId, factoryId);
  const updateOnboarding = useFactoryOnboarding(organizationId, factoryId);
  const createLine = useCreateFactoryLine(organizationId, factoryId);
  const createWorkOrder = useCreateWorkOrder(organizationId, factoryId);
  const dispatchWorkOrder = useDispatchWorkOrder(organizationId, factoryId);
  const installer = useInstallFactory();
  return { updateFactory, updateOnboarding, createLine, createWorkOrder, dispatchWorkOrder, installer };
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
  const apis = useOnboardingWorkspaceApis(args.organizationId, args.factoryId);
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
    updateFactory: apis.updateFactory.mutateAsync,
    updateOnboarding: apis.updateOnboarding.mutateAsync,
  });
  const finish = useFinishOnboarding({
    ...args,
    setup,
    selections: integrations.selections,
    setSaving,
    updateFactory: apis.updateFactory.mutateAsync,
    updateOnboarding: apis.updateOnboarding.mutateAsync,
    installFactory: apis.installer.installFactory,
    createLine: apis.createLine.mutateAsync,
    createWorkOrder: apis.createWorkOrder.mutateAsync,
    dispatchWorkOrder: apis.dispatchWorkOrder.mutateAsync,
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
    saving: saving || apis.installer.isInstalling || apis.createWorkOrder.isPending,
    ...saves,
    finish: finishSetup,
    ...cancel,
  };
}
