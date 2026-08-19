import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesUpdateFactoryOnboardingBody,
  FactoryLineStep,
} from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactoryLine, useCreateWorkOrder, useDeleteFactory, useUpdateFactory } from "@/hooks/useFactoryData";
import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { githubInstallationUrl } from "@/lib/githubInstallation";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "@/pages/home/factories";
import { useInstallFactory, type InstallFactoryInput } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";

import {
  factoryListPath,
  factoryOverviewPath,
  factorySetupPath,
  workOrderDetailPath,
} from "../../lib/factoryPagePaths";
import { clearLastVisitedFactory } from "../../lib/lastVisitedFactory";
import { markWorkspaceGettingStarted } from "./gettingStartedState";
import type { IntegrationId, IssuesChoiceId, WizardStepId } from "./onboardingFixtures";
import { initialWizardStep } from "./onboardingStatus";
import { useFactoryOnboarding } from "./useFactoryOnboarding";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { workspaceNameFromRepository } from "./workspaceNames";

const DEFAULT_LINE_NAME = "Software delivery";
const ONBOARDING_INTEGRATIONS = ["github", "claude"];

function initialSelections(onboarding: FactoriesFactory["onboarding"]): IntegrationSelections {
  const selections: IntegrationSelections = {};
  if (onboarding?.vcsIntegrationId) {
    selections.github = {
      id: onboarding.vcsIntegrationId,
      name: onboarding.vcsIntegrationId,
      ready: false,
    };
  }
  if (onboarding?.agentIntegrationId) {
    selections.claude = {
      id: onboarding.agentIntegrationId,
      name: onboarding.agentIntegrationId,
      ready: false,
    };
  }
  return selections;
}

function localIssuesSource(source?: string): IssuesChoiceId | null {
  const sources: Record<string, IssuesChoiceId> = {
    ISSUES_SOURCE_VCS: "vcs",
    ISSUES_SOURCE_LINEAR: "linear",
    ISSUES_SOURCE_JIRA: "jira",
    ISSUES_SOURCE_SKIP: "skip",
  };
  return source ? (sources[source] ?? null) : null;
}

function apiIssuesSource(source: IssuesChoiceId | null): FactoriesUpdateFactoryOnboardingBody["issuesSource"] {
  const sources: Partial<Record<IssuesChoiceId, FactoriesUpdateFactoryOnboardingBody["issuesSource"]>> = {
    vcs: "ISSUES_SOURCE_VCS",
    linear: "ISSUES_SOURCE_LINEAR",
    jira: "ISSUES_SOURCE_JIRA",
    skip: "ISSUES_SOURCE_SKIP",
  };
  return source ? sources[source] : "ISSUES_SOURCE_UNSPECIFIED";
}

function useIntegrationSelections(onboarding: FactoriesFactory["onboarding"]) {
  const [selections, setSelections] = useState<IntegrationSelections>(() => initialSelections(onboarding));
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

// The workspace is created with a placeholder name before the repository is
// known, so the app repository provides the name the last step shows.
function useNameFromRepository(setup: OnboardingSetupApi) {
  useEffect(() => {
    if (!setup.selectedRepo) return;
    const suggestion = workspaceNameFromRepository(setup.selectedRepo);
    if (suggestion) setup.suggestWorkspaceName(suggestion);
  }, [setup]);
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

type UpdateOnboarding = (input: FactoriesUpdateFactoryOnboardingBody) => Promise<unknown>;

const PRIMARY_LINE_APP_ENTRYPOINT = ONBOARDING_LINE_APPS[0].entrypointNodeId;

// A finished line has one step per bundled app, each calling the app onRun
// entrypoint. Match on the first entrypoint to recover a line provisioned by an
// earlier, interrupted attempt.
function findProvisionedLine(factory: FactoriesFactory | null): FactoriesFactoryLine | undefined {
  return factory?.lines?.find((line) =>
    line.steps?.some((step) => step.app?.entrypoint === PRIMARY_LINE_APP_ENTRYPOINT),
  );
}

type InstallOnboardingApp = (
  input: InstallFactoryInput,
) => Promise<{ canvasId: string; canvasName: string } | undefined>;

async function installOnboardingApp(args: {
  factoryId: string;
  appFactoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  installFactory: InstallOnboardingApp;
}): Promise<{ canvasId: string; canvasName: string }> {
  const installed = await args.installFactory({
    factoryId: args.appFactoryId,
    workspaceFactoryId: args.factoryId,
    integrations: args.selections,
    installParams: {
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
    },
    startingTaskPrompt: "",
    navigateOnComplete: false,
    startInitialRun: false,
  });
  if (!installed?.canvasId) throw new Error(`Failed to create the ${args.appFactoryId} app`);
  return installed;
}

// Install each bundled app in order and return the line steps that call them.
// installFactory clears its pending-canvas ref after each success, so the
// sequential calls create distinct canvases.
async function provisionLineApps(args: {
  factoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  installFactory: InstallOnboardingApp;
}): Promise<FactoryLineStep[]> {
  const steps: FactoryLineStep[] = [];
  for (const app of ONBOARDING_LINE_APPS) {
    const installed = await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId: app.factoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      installFactory: args.installFactory,
    });
    steps.push({
      type: "runApp",
      app: { app: installed.canvasId, entrypoint: app.entrypointNodeId },
    });
  }
  return steps;
}

// Event apps listen for GitHub events and are not factory line steps. Install
// them even when the line already exists, so a retry after a failed finish
// still creates PR Closure.
async function provisionEventApps(args: {
  factoryId: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  installFactory: InstallOnboardingApp;
}): Promise<void> {
  for (const appFactoryId of ONBOARDING_EVENT_APPS) {
    await installOnboardingApp({
      factoryId: args.factoryId,
      appFactoryId,
      selections: args.selections,
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
      installFactory: args.installFactory,
    });
  }
}

interface ProvisionedLine {
  lineId: string;
  primaryAppId: string;
}

async function provisionLine(args: {
  factory: FactoriesFactory | null;
  savedLineId?: string;
  savedAppId?: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  updateOnboarding: UpdateOnboarding;
}): Promise<ProvisionedLine> {
  const existing = findProvisionedLine(args.factory);
  const lineId = args.savedLineId ?? existing?.id;
  if (lineId) {
    const primaryAppId = args.savedAppId ?? existing?.steps?.[0]?.app?.app;
    if (primaryAppId) return { lineId, primaryAppId };
  }

  const steps = await provisionLineApps({
    factoryId: args.factory?.id ?? "",
    selections: args.selections,
    appRepository: args.appRepository,
    backlogRepository: args.backlogRepository,
    installFactory: args.installFactory,
  });
  const primaryAppId = steps[0]?.app?.app;
  if (!primaryAppId) throw new Error("Software delivery apps were not created");

  const line = await args.createLine({ name: DEFAULT_LINE_NAME, steps });
  if (!line.id) throw new Error("Software delivery line was not created");
  await args.updateOnboarding({ provisionedAppId: primaryAppId, provisionedLineId: line.id });
  return { lineId: line.id, primaryAppId };
}

function useSectionSaves(args: {
  setup: OnboardingSetupApi;
  selections: IntegrationSelections;
  setSaving: (saving: boolean) => void;
  updateOnboarding: UpdateOnboarding;
}) {
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
  return { saveRepository, saveIssues };
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
  createWorkOrder: (input: { title: string; description: string }) => Promise<{ number?: number | string | null }>;
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
      if (workspaceName !== args.factory?.name) {
        await args.updateFactory({ name: workspaceName });
      }
      await args.updateOnboarding({
        vcsIntegrationId: github.id,
        agentIntegrationId: claude.id,
        appRepository,
        backlogRepository,
        issuesSource: apiIssuesSource(args.setup.issuesChoice),
        agentHarness: "AGENT_HARNESS_CLAUDE_CODE",
      });
      const { lineId, primaryAppId } = await provisionLine({
        factory: args.factory,
        savedLineId: args.factory?.onboarding?.provisionedLineId,
        savedAppId: args.factory?.onboarding?.provisionedAppId,
        selections: args.selections,
        appRepository,
        backlogRepository,
        installFactory: args.installFactory,
        createLine: args.createLine,
        updateOnboarding: args.updateOnboarding,
      });
      await provisionEventApps({
        factoryId: args.factoryId,
        selections: args.selections,
        appRepository,
        backlogRepository,
        installFactory: args.installFactory,
      });
      await args.updateOnboarding({
        provisionedAppId: primaryAppId,
        provisionedLineId: lineId,
        complete: true,
      });
      const order = await args.createWorkOrder({
        title: workOrderTitle,
        description: workOrderDescription,
      });
      markWorkspaceGettingStarted(args.organizationId, args.factoryId);
      if (order.number != null && order.number !== "") {
        navigate(workOrderDetailPath(args.organizationId, args.factoryKey, order.number), { replace: true });
        return;
      }
      navigate(factoryOverviewPath(args.organizationId, args.factoryKey), { replace: true });
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
  const connect = useIntegrationConnectDialog({
    organizationId: args.organizationId,
    returnTo: factorySetupPath(args.organizationId, args.factoryKey),
    integrationNames: ONBOARDING_INTEGRATIONS,
    selections: integrations.selections,
    onSelectionsChange: integrations.setSelections,
  });
  const setup = useOnboardingSetupState(args.factory?.name ?? "", {
    connected: integrations.connected,
    simulateDiscovery: false,
  });
  useRestoreSetup(setup, onboarding, integrations.selections);
  useNameFromRepository(setup);

  const [openSection, setOpenSection] = useState<WizardStepId>(() => initialWizardStep(onboarding));
  const [saving, setSaving] = useState(false);
  const updateFactory = useUpdateFactory(args.organizationId, args.factoryId);
  const updateOnboarding = useFactoryOnboarding(args.organizationId, args.factoryId);
  const createLine = useCreateFactoryLine(args.organizationId, args.factoryId);
  const createWorkOrder = useCreateWorkOrder(args.organizationId, args.factoryId);
  const installer = useInstallFactory();
  const githubIntegrationId = integrations.selections.github?.ready ? integrations.selections.github.id : "";
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
  });
  const cancel = useCancelOnboarding({
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    canDelete: canAct("factories", "delete"),
  });

  return {
    setup,
    openSection,
    setOpenSection,
    requestConnect: connect.requestConnect,
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
    finish,
    ...cancel,
  };
}
