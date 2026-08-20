import type { FactoriesFactory, FactoriesFactoryLine, FactoryApp, FactoryLineStep } from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryLine,
  useCreateWorkOrder,
  useDeleteFactory,
  useDispatchWorkOrder,
  useFactoryApps,
  useUpdateFactory,
} from "@/hooks/useFactoryData";
import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { githubInstallationUrl } from "@/lib/githubInstallation";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
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

const ONBOARDING_INTEGRATIONS = ["github", "claude"];

function useIntegrationSelections(onboarding: FactoriesFactory["onboarding"]) {
  const [selections, setSelections] = useState<IntegrationSelections>(() => initialOnboardingSelections(onboarding));
  const connected = useMemo(() => {
    const ready = new Set<IntegrationId>();
    if (selections.github?.ready) ready.add("github");
    if (selections.claude?.ready) ready.add("claude");
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
  claude: { id: string };
  loadFactoryApps: () => Promise<FactoryApp[]>;
}): Promise<{ number?: number | string | null }> {
  if (args.workspaceName !== args.factory?.name) {
    await args.updateFactory({ name: args.workspaceName });
  }
  await args.updateOnboarding({
    vcsIntegrationId: args.github.id,
    agentIntegrationId: args.claude.id,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    issuesSource: apiIssuesSource(args.setup.issuesChoice),
    agentHarness: "AGENT_HARNESS_CLAUDE_CODE",
  });
  const { lineId, primaryAppId } = await provisionLine({
    factory: args.factory,
    savedLineId: args.factory?.onboarding?.provisionedLineId,
    savedAppId: args.factory?.onboarding?.provisionedAppId,
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    installFactory: args.installFactory,
    createLine: args.createLine,
    updateOnboarding: args.updateOnboarding,
  });
  await provisionEventApps({
    factoryId: args.factoryId,
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    installFactory: args.installFactory,
    loadExistingApps: args.loadFactoryApps,
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
  loadFactoryApps: () => Promise<FactoryApp[]>;
}) {
  const navigate = useNavigate();
  return async () => {
    const appRepository = args.setup.selectedRepo;
    const backlogRepository = args.setup.issuesRepo ?? appRepository;
    const workspaceName = args.setup.workspaceName.trim();
    const workOrderTitle = args.setup.workOrderTitle.trim();
    const workOrderDescription = args.setup.workOrderDescription.trim();
    const github = args.selections.github;
    const claude = args.selections.claude;
    if (!appRepository || !backlogRepository || !github?.ready || !claude?.ready) {
      showErrorToast("Connect GitHub and Claude, then select both repositories.");
      return;
    }
    if (!workspaceName) {
      showErrorToast("Enter a workspace name.");
      return;
    }
    if (!workOrderTitle || !workOrderDescription) {
      showErrorToast("Enter a work order title and description.");
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
        claude,
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

export function useOnboardingPageModel(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
}) {
  const { canAct } = usePermissions();
  const onboarding = args.factory?.onboarding;
  const integrations = useIntegrationSelections(onboarding);
  const setup = useOnboardingSetupState(args.factory?.name ?? "", {
    connected: integrations.connected,
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
  const factoryApps = useFactoryApps(args.organizationId, args.factoryId);
  const installer = useInstallFactory();
  const githubIntegrationId = integrations.selections.github?.ready ? integrations.selections.github.id : "";
  const githubConnections = useOnboardingGithubConnections({
    integrationData: connect.integrationData,
    openSection,
    selectNewest: searchParams.get("pick") === "newest",
    selections: integrations.selections,
    selectInstance: connect.selectInstance,
  });
  const githubIntegration = useIntegration(args.organizationId, githubIntegrationId);
  const resources = useIntegrationResources(args.organizationId, githubIntegrationId, "repository");
  const repositories = useMemo(
    () =>
      (resources.data ?? [])
        .map((resource) => resource.name ?? resource.id ?? "")
        .filter((repository): repository is string => Boolean(repository)),
    [resources.data],
  );

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
    loadFactoryApps: async () => factoryApps.data ?? (await factoryApps.refetch()).data ?? [],
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
      window.open(githubInstallationUrl(githubIntegration.data), "_blank", "noopener,noreferrer");
    },
    integrationDialogs: connect.dialogs,
    repositories,
    repositoriesLoading: resources.isLoading,
    repositoriesError: resources.error,
    canConfigureWorkspace:
      canAct("factories", "update") &&
      canAct("integrations", "create") &&
      canAct("canvases", "create") &&
      canAct("canvases", "update"),
    saving: saving || installer.isInstalling || createWorkOrder.isPending,
    ...saves,
    finish: finishSetup,
    ...cancel,
  };
}
