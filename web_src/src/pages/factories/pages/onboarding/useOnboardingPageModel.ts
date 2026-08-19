import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  FactoriesUpdateFactoryOnboardingBody,
  FactoryLineStep,
} from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateFactoryLine, useUpdateFactory } from "@/hooks/useFactoryData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useOrganizationInviteLink } from "@/hooks/useOrganizationData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { ONBOARDING_EVENT_APPS, ONBOARDING_LINE_APPS } from "@/pages/home/factories";
import { useInstallFactory, type InstallFactoryInput } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";

import { factoryOverviewPath } from "../../lib/factoryPagePaths";
import { markWorkspaceGettingStarted } from "./gettingStartedState";
import type { SectionId } from "./OnboardingWireframe";
import type { IntegrationId, IssuesChoiceId } from "./onboardingFixtures";
import { useFactoryOnboarding } from "./useFactoryOnboarding";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";

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

// The workspace already has a name when this page opens, so setup resumes at
// the first section that still needs an answer.
function initialSection(onboarding: FactoriesFactory["onboarding"]): SectionId {
  if (onboarding?.agentHarness || onboarding?.issuesSource) return "agent";
  if (onboarding?.appRepository) return "issues";
  return "repo";
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
  updateFactory: (input: { name: string }) => Promise<unknown>;
  updateOnboarding: UpdateOnboarding;
}) {
  const saveName = () => runSave(args.setSaving, () => args.updateFactory({ name: args.setup.workspaceName.trim() }));
  const saveRepository = () => {
    const integrationId = args.selections.github?.id;
    if (!args.setup.selectedRepo || !integrationId) return Promise.resolve(false);
    return runSave(args.setSaving, () =>
      args.updateOnboarding({
        vcsIntegrationId: integrationId,
        appRepository: args.setup.selectedRepo!,
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

function useFinishOnboarding(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  setup: OnboardingSetupApi;
  selections: IntegrationSelections;
  setSaving: (saving: boolean) => void;
  updateOnboarding: UpdateOnboarding;
  installFactory: InstallOnboardingApp;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
}) {
  const navigate = useNavigate();
  return async () => {
    const appRepository = args.setup.selectedRepo;
    const backlogRepository = args.setup.issuesRepo ?? appRepository;
    const github = args.selections.github;
    const claude = args.selections.claude;
    if (!appRepository || !backlogRepository || !github?.ready || !claude?.ready) {
      showErrorToast("Connect GitHub and Claude, then select both repositories.");
      return;
    }

    args.setSaving(true);
    try {
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
      markWorkspaceGettingStarted(args.organizationId, args.factoryId);
      navigate(factoryOverviewPath(args.organizationId, args.factoryKey), { replace: true });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to finish workspace setup"));
    } finally {
      args.setSaving(false);
    }
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
    integrationNames: ONBOARDING_INTEGRATIONS,
    selections: integrations.selections,
    onSelectionsChange: integrations.setSelections,
  });
  const setup = useOnboardingSetupState(args.factory?.name ?? "", {
    connected: integrations.connected,
    simulateDiscovery: false,
  });
  useRestoreSetup(setup, onboarding, integrations.selections);

  const [openSection, setOpenSection] = useState<SectionId>(() => initialSection(onboarding));
  const [saving, setSaving] = useState(false);
  const updateFactory = useUpdateFactory(args.organizationId, args.factoryId);
  const updateOnboarding = useFactoryOnboarding(args.organizationId, args.factoryId);
  const createLine = useCreateFactoryLine(args.organizationId, args.factoryId);
  const installer = useInstallFactory();
  const canInvite = canAct("members", "create");
  const invite = useOrganizationInviteLink(args.organizationId, canInvite);
  const githubIntegrationId = integrations.selections.github?.ready ? integrations.selections.github.id : "";
  const resources = useIntegrationResources(args.organizationId, githubIntegrationId, "repository");
  const repositories = useMemo(
    () =>
      (resources.data ?? [])
        .map((resource) => resource.name ?? resource.id ?? "")
        .filter((repository): repository is string => Boolean(repository)),
    [resources.data],
  );
  const inviteUrl =
    invite.data?.enabled && invite.data.token ? `${window.location.origin}/invite/${invite.data.token}` : null;

  const saves = useSectionSaves({
    setup,
    selections: integrations.selections,
    setSaving,
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
  });
  const finish = useFinishOnboarding({
    ...args,
    setup,
    selections: integrations.selections,
    setSaving,
    updateOnboarding: updateOnboarding.mutateAsync,
    installFactory: installer.installFactory,
    createLine: createLine.mutateAsync,
  });

  return {
    setup,
    openSection,
    setOpenSection,
    requestConnect: connect.requestConnect,
    integrationDialogs: connect.dialogs,
    repositories,
    repositoriesLoading: resources.isLoading,
    repositoriesError: resources.error,
    inviteUrl,
    inviteLoading: invite.isLoading,
    canInvite,
    canConfigureWorkspace:
      canAct("factories", "update") &&
      canAct("integrations", "create") &&
      canAct("canvases", "create") &&
      canAct("canvases", "update"),
    saving: saving || installer.isInstalling,
    ...saves,
    finish,
  };
}
