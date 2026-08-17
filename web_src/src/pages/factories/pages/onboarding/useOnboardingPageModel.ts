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
import { useInstallFactory, type InstallFactoryInput } from "@/pages/home/useInstallFactory";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";

import { factoryOverviewPath } from "../../lib/factoryPagePaths";
import { markWorkspaceGettingStarted } from "./gettingStartedState";
import type { SectionId } from "./OnboardingWireframe";
import type { IntegrationId, IssuesChoiceId } from "./onboardingFixtures";
import { useFactoryOnboarding } from "./useFactoryOnboarding";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";

const WORK_ORDER_ENTRYPOINT = "work-order-dispatch";
const DEFAULT_LINE_NAME = "Software delivery";

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

function initialSection(onboarding: FactoriesFactory["onboarding"]): SectionId {
  if (onboarding?.agentHarness || onboarding?.issuesSource) return "agent";
  if (onboarding?.appRepository) return "issues";
  return "name";
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
  const updateSelection = useCallback((name: "github" | "claude", next: IntegrationSelections) => {
    setSelections((current) => {
      const updated = { ...current };
      if (next[name]) updated[name] = next[name];
      else delete updated[name];
      return updated;
    });
  }, []);
  return { selections, connected, updateSelection };
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

async function provisionApp(args: {
  factoryId: string;
  existingAppId?: string;
  selections: IntegrationSelections;
  appRepository: string;
  backlogRepository: string;
  installFactory: (input: InstallFactoryInput) => Promise<{ canvasId: string; canvasName: string } | undefined>;
  updateOnboarding: UpdateOnboarding;
}): Promise<string> {
  let appId = args.existingAppId;
  const installed = await args.installFactory({
    workspaceFactoryId: args.factoryId,
    existingCanvasId: appId,
    integrations: args.selections,
    installParams: {
      appRepository: args.appRepository,
      backlogRepository: args.backlogRepository,
    },
    startingTaskPrompt: "",
    navigateOnComplete: false,
    startInitialRun: false,
    onCanvasReady: async ({ canvasId }) => {
      appId = canvasId;
      await args.updateOnboarding({ provisionedAppId: canvasId });
    },
  });
  appId = installed?.canvasId ?? appId;
  if (!appId) throw new Error("Software Factory app was not created");
  return appId;
}

function existingLineId(factory: FactoriesFactory | null, appId: string): string | undefined {
  return factory?.lines?.find((line) =>
    line.steps?.some((step) => step.app?.app === appId && step.app.entrypoint === WORK_ORDER_ENTRYPOINT),
  )?.id;
}

async function provisionLine(args: {
  factory: FactoriesFactory | null;
  appId: string;
  existingLineId?: string;
  createLine: (input: { name: string; steps: FactoryLineStep[] }) => Promise<FactoriesFactoryLine>;
  updateOnboarding: UpdateOnboarding;
}): Promise<string> {
  const savedLineId = args.existingLineId ?? existingLineId(args.factory, args.appId);
  if (savedLineId) return savedLineId;
  const line = await args.createLine({
    name: DEFAULT_LINE_NAME,
    steps: [
      {
        name: "Build",
        type: "runApp",
        app: { app: args.appId, entrypoint: WORK_ORDER_ENTRYPOINT },
      },
    ],
  });
  if (!line.id) throw new Error("Software delivery line was not created");
  await args.updateOnboarding({ provisionedLineId: line.id });
  return line.id;
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
  installFactory: (input: InstallFactoryInput) => Promise<{ canvasId: string; canvasName: string } | undefined>;
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
      const appId = await provisionApp({
        factoryId: args.factoryId,
        existingAppId: args.factory?.onboarding?.provisionedAppId,
        selections: args.selections,
        appRepository,
        backlogRepository,
        installFactory: args.installFactory,
        updateOnboarding: args.updateOnboarding,
      });
      const lineId = await provisionLine({
        factory: args.factory,
        appId,
        existingLineId: args.factory?.onboarding?.provisionedLineId,
        createLine: args.createLine,
        updateOnboarding: args.updateOnboarding,
      });
      await args.updateOnboarding({
        provisionedAppId: appId,
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
    selections: integrations.selections,
    updateSelection: integrations.updateSelection,
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
