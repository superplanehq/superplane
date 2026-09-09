import type { FactoriesFactory, OrganizationsIntegration } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { fetchFactoryApps, useCreateFactoryLine, useUpdateFactory } from "@/hooks/useFactoryData";
import { fetchFactoryIntakes, useCreateFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { fetchFactoryPRFeedbackHandlers, useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";
import { resolveGithubDefaultBranch, useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { useOrganizationWorkspaceUsage } from "@/hooks/useOrganizationWorkspaceUsage";
import { useUpdateOrganization } from "@/hooks/useOrganizationData";
import { getApiErrorMessage } from "@/lib/errors";
import { githubInstallationUrl } from "@/lib/githubInstallation";
import { showErrorToast } from "@/lib/toast";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { useQueryClient } from "@tanstack/react-query";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import { advanceAfterGithubConnect } from "./advanceAfterGithubConnect";
import { AGENT_PROVIDER_IDS, isHostedAgentReady } from "./onboardingAgentReadiness";
import {
  githubIntegrationOwner,
  githubOwnerFromConnections,
  nameOrganizationFromGitHubOwner,
  shouldNameOrganizationFromGitHub,
} from "./initialOnboardingOrganization";
import type { IntegrationId, IssuesChoiceId, WizardStepId } from "./onboardingFixtures";
import type { OnboardingWorkspaceResolution } from "./onboardingWorkspaceResolutionContext";
import { onboardingStepPath } from "./onboardingStepPath";
import type { UpdateOnboarding } from "./onboardingProvision";
import {
  apiIssuesSource,
  initialOnboardingSelections,
  initialWizardStep,
  isWizardStepId,
  localIssuesSource,
} from "./onboardingStatus";
import { saveWithFreeWorkspaceName } from "./uniqueFactoryName";
import { useFactoryOnboarding } from "./useFactoryOnboarding";
import { useFinishOnboarding, type OnboardingDestination } from "./useFinishOnboarding";
import { useFinishSetupAction } from "./useFinishSetupAction";
import { useOnboardingAgentPlan } from "./useOnboardingAgentPlan";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { persistSelectedGithubConnection } from "./onboardingGithubCleanup";
import { useOnboardingGithubConnections } from "./useSelectNewGithubConnection";

const ONBOARDING_INTEGRATIONS = ["github", ...AGENT_PROVIDER_IDS];

// Onboarding never adopts an existing organization GitHub or agent
// connection on its own. GitHub repositories must come from the account the
// user picked. Agent keys stay unselected so a new workspace can use the
// canonical SuperPlane template when hosted credit is available.
const ONBOARDING_MANUAL_SELECTIONS = ["github", ...AGENT_PROVIDER_IDS] as const;

/**
 * Setup only needs the keys that make an agent run. The Anthropic admin key
 * serves usage and cost reports, so users add it later in organization settings.
 */
const ONBOARDING_HIDDEN_CONFIGURATION_FIELDS: Record<string, string[]> = {
  claude: ["adminKey"],
};

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
  takenNames: string[];
  updateFactory: (input: { name: string }) => Promise<unknown>;
  updateOnboarding: UpdateOnboarding;
}) {
  const saveName = () => {
    const workspaceName = args.setup.workspaceName.trim();
    if (!workspaceName) return Promise.resolve(false);
    if (workspaceName === args.factoryName) return Promise.resolve(true);
    return runSave(args.setSaving, () =>
      saveWithFreeWorkspaceName({
        name: workspaceName,
        takenNames: args.takenNames,
        save: (name) => args.updateFactory({ name }),
      }),
    );
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
  // The caller passes the source, because a selection made in the same render
  // is not readable from the setup state yet.
  const saveIssues = (source: IssuesChoiceId) => {
    const backlogRepository = args.setup.issuesRepo ?? args.setup.selectedRepo;
    if (!backlogRepository) return Promise.resolve(false);
    return runSave(args.setSaving, () =>
      args.updateOnboarding({
        backlogRepository,
        issuesSource: apiIssuesSource(source),
      }),
    );
  };
  return { saveName, saveRepository, saveIssues };
}

/** Names held by the other workspaces of the organization. */
function otherWorkspaceNames(factories: FactoriesFactory[], factoryId: string): string[] {
  return factories
    .filter((factory) => factory.id !== factoryId)
    .map((factory) => factory.name ?? "")
    .filter(Boolean);
}

function canConfigureWorkspace(canAct: (resource: string, action: string) => boolean): boolean {
  return (
    canAct("factories", "update") &&
    canAct("integrations", "create") &&
    canAct("canvases", "create") &&
    canAct("canvases", "update")
  );
}

function useOnboardingAgentContext(organizationId: string, connected: Set<IntegrationId>) {
  const spend = useOrganizationWorkspaceUsage(organizationId);
  const remainingCreditCents = parseWorkOrderMetric(spend.data?.remainingCreditCents);
  return useOnboardingAgentPlan(organizationId, connected, remainingCreditCents, {
    provider: spend.data?.defaultHostedProvider,
    model: spend.data?.defaultHostedModel,
  });
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
    // `isFetching` also covers refetches after an edit of the GitHub
    // connection, so the screen shows a placeholder instead of a stale list.
    repositoriesLoading: resources.isFetching,
    repositoriesError: resources.error,
  };
}

/**
 * After an organization rename, the connection list reloads under the new
 * slug before the repository request can start. Keep one loader visible
 * through both requests.
 */
function isRepositoryListLoading(args: {
  savedIntegrationId?: string;
  selectedIntegrationId: string;
  connectionsLoading: boolean;
  repositoriesLoading: boolean;
}): boolean {
  if (!args.savedIntegrationId && !args.selectedIntegrationId) return false;
  return args.connectionsLoading || args.repositoriesLoading;
}

function useOnboardingGithubConnectionSelected(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  onboardingEntryPath?: string | null;
  reresolveWorkspace: OnboardingWorkspaceResolution | null;
  setup: OnboardingSetupApi;
  setOpenSection: (section: WizardStepId) => void;
  updateOnboarding: UpdateOnboarding;
  updateOrganization: ReturnType<typeof useUpdateOrganization>;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  return async (integration: OrganizationsIntegration) => {
    args.setup.selectVcsHost("github");
    args.setOpenSection("repo");

    const integrationId = integration.metadata?.id;
    if (!integrationId) return;

    const previousId = args.factory?.onboarding?.vcsIntegrationId;
    if (
      !(await persistSelectedGithubConnection({
        ...args,
        queryClient,
        integrationId,
        previousId,
      }))
    ) {
      return;
    }

    await advanceAfterGithubConnect({
      onboardingEntryPath: args.onboardingEntryPath,
      organizationId: args.organizationId,
      nextSlug: args.organizationId,
      factoryId: args.factoryId,
      factoryKey: args.factoryKey,
      navigate,
      reresolveWorkspace: args.reresolveWorkspace,
      queryClient,
    });

    const owner = githubIntegrationOwner(integration);
    if (!owner || !shouldNameOrganizationFromGitHub(args.factory)) return;

    try {
      const nextSlug = await nameOrganizationFromGitHubOwner({
        owner,
        currentSlug: args.organizationId,
        update: async (identity) => {
          const response = await args.updateOrganization.mutateAsync(identity);
          return response.data?.organization?.metadata?.slug;
        },
      });
      if (!nextSlug || nextSlug === args.organizationId) return;

      await advanceAfterGithubConnect({
        onboardingEntryPath: args.onboardingEntryPath,
        organizationId: args.organizationId,
        nextSlug,
        factoryId: args.factoryId,
        factoryKey: args.factoryKey,
        navigate,
        reresolveWorkspace: args.reresolveWorkspace,
        queryClient,
      });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Could not name the organization from the GitHub connection"));
    }
  };
}

function useOnboardingGithubConnectionsForPage(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  onboardingEntryPath?: string | null;
  reresolveWorkspace: OnboardingWorkspaceResolution | null;
  searchParams: URLSearchParams;
  setup: OnboardingSetupApi;
  openSection: WizardStepId;
  setOpenSection: (section: WizardStepId) => void;
  updateOnboarding: UpdateOnboarding;
  updateOrganization: ReturnType<typeof useUpdateOrganization>;
  integrationData: Parameters<typeof useOnboardingGithubConnections>[0]["integrationData"];
  selections: IntegrationSelections;
  selectInstance: (integrationName: string, integrationId: string) => void;
}) {
  const selectNewest = args.searchParams.get("pick") === "newest";
  const onConnectionSelected = useOnboardingGithubConnectionSelected(args);

  return useOnboardingGithubConnections({
    integrationData: args.integrationData,
    openSection: args.openSection,
    // Only the `pick=newest` round trip auto-selects. Every other path,
    // including an install request approved outside the round trip, shows
    // the account picker and waits for the user to select the account.
    selectNewest,
    selections: args.selections,
    selectInstance: args.selectInstance,
    onConnectionSelected,
  });
}

function useSelectOnboardingVcsConnection(args: {
  organizationId: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  factoryId: string;
  setup: OnboardingSetupApi;
  updateOnboarding: UpdateOnboarding;
  currentId: string;
  selectInstance: (integrationName: string, integrationId: string) => void;
}) {
  const queryClient = useQueryClient();
  return async (integrationId: string): Promise<boolean> => {
    if (integrationId !== args.currentId) {
      const saved = await persistSelectedGithubConnection({
        setup: args.setup,
        updateOnboarding: args.updateOnboarding,
        organizationId: args.organizationId,
        factory: args.factory,
        factories: args.factories,
        factoryId: args.factoryId,
        queryClient,
        integrationId,
        previousId: args.currentId || args.factory?.onboarding?.vcsIntegrationId,
      });
      if (!saved) return false;
    }
    args.selectInstance("github", integrationId);
    return true;
  };
}

/** The mutation hooks the page model saves and provisions through. */
function useOnboardingMutations(organizationId: string, factoryId: string) {
  return {
    updateFactory: useUpdateFactory(organizationId, factoryId),
    updateOnboarding: useFactoryOnboarding(organizationId, factoryId),
    updateOrganization: useUpdateOrganization(organizationId),
    createLine: useCreateFactoryLine(organizationId, factoryId),
    createIntake: useCreateFactoryIntake(organizationId, factoryId),
    createPRFeedbackHandler: useCreateFactoryPRFeedbackHandler(organizationId, factoryId),
    installer: useInstallFactory({ organizationId }),
  };
}

export function useOnboardingPageModel(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  onboardingEntryPath?: string | null;
  reresolveWorkspace?: OnboardingWorkspaceResolution | null;
}) {
  const { canAct } = usePermissions();
  const onboarding = args.factory?.onboarding;
  const integrations = useIntegrationSelections(onboarding);
  const agent = useOnboardingAgentContext(args.organizationId, integrations.connected);
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
    // Return to this step after the provider round trip.
    returnTo: onboardingStepPath(
      args.onboardingEntryPath ?? factorySetupPath(args.organizationId, args.factoryKey),
      openSection,
    ),
    integrationNames: ONBOARDING_INTEGRATIONS,
    selections: integrations.selections,
    onSelectionsChange: integrations.setSelections,
    hiddenConfigurationFields: ONBOARDING_HIDDEN_CONFIGURATION_FIELDS,
    manualSelectionNames: ONBOARDING_MANUAL_SELECTIONS,
  });

  const [saving, setSaving] = useState(false);
  const [provisionedDestination, setProvisionedDestination] = useState<OnboardingDestination | null>(null);
  const {
    updateFactory,
    updateOnboarding,
    updateOrganization,
    createLine,
    createIntake,
    createPRFeedbackHandler,
    installer,
  } = useOnboardingMutations(args.organizationId, args.factoryId);
  const githubIntegrationId = integrations.selections.github?.ready ? integrations.selections.github.id : "";
  const githubConnections = useOnboardingGithubConnectionsForPage({
    ...args,
    reresolveWorkspace: args.reresolveWorkspace ?? null,
    searchParams,
    setup,
    openSection,
    setOpenSection,
    updateOnboarding: updateOnboarding.mutateAsync,
    updateOrganization,
    integrationData: connect.integrationData,
    selections: integrations.selections,
    selectInstance: connect.selectInstance,
  });
  const github = useOnboardingGithubRepos(args.organizationId, githubIntegrationId);

  const takenNames = useMemo(
    () => otherWorkspaceNames(args.factories, args.factoryId),
    [args.factories, args.factoryId],
  );

  const saves = useSectionSaves({
    setup,
    selections: integrations.selections,
    setSaving,
    factoryName: args.factory?.name ?? "",
    takenNames,
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
  });
  const githubOwner = githubOwnerFromConnections(
    [...githubConnections.readyInstances, ...githubConnections.allInstances],
    githubIntegrationId,
  );
  const finish = useFinishOnboarding({
    ...args,
    setup,
    selections: integrations.selections,
    setSaving,
    takenNames,
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
    installFactory: installer.installFactory,
    createLine: createLine.mutateAsync,
    listIntakes: () => fetchFactoryIntakes(args.organizationId, args.factoryId),
    createIntake: createIntake.mutateAsync,
    listPRFeedbackHandlers: () => fetchFactoryPRFeedbackHandlers(args.organizationId, args.factoryId),
    createPRFeedbackHandler: createPRFeedbackHandler.mutateAsync,
    listApps: () => fetchFactoryApps(args.organizationId, args.factoryId),
    resolveDefaultBranch: (repository: string) =>
      resolveGithubDefaultBranch(args.organizationId, githubIntegrationId, repository),
    remainingCreditCents: agent.remainingCreditCents,
    hostedModelsLoading: agent.hostedModelsLoading,
    plan: agent.plan,
    githubOwner,
    updateOrganization: async (identity) => {
      const response = await updateOrganization.mutateAsync(identity);
      return response.data?.organization?.metadata?.slug;
    },
    onProvisioned: setProvisionedDestination,
  });
  const finishSetup = useFinishSetupAction({
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    factoryKey: args.factoryKey,
    factory: args.factory,
    setup,
    finish,
  });

  return {
    setup,
    // True when hosted credentials cover the agent, so setup can skip the
    // agent screen and provision from the ticket screen.
    hostedAgentReady: isHostedAgentReady(agent.plan),
    agentLoading: agent.hostedModelsLoading,
    openSection,
    setOpenSection,
    requestConnect: connect.requestConnect,
    // The connect screen refetches on open, so the picker never shows a
    // stale connection list.
    refreshGithubConnections: connect.refetchConnections,
    githubConnectionsLoading: connect.connectionsLoading,
    requestPrivateGitHubConnect: connect.requestPrivateGitHubConnect,
    offersPrivateGitHubAppSetup: connect.offersPrivateGitHubAppSetup,
    createVcsConnection: () => connect.createNew("github"),
    selectVcsConnection: useSelectOnboardingVcsConnection({
      organizationId: args.organizationId,
      factory: args.factory,
      factories: args.factories,
      factoryId: args.factoryId,
      setup,
      updateOnboarding: updateOnboarding.mutateAsync,
      currentId: githubIntegrationId,
      selectInstance: connect.selectInstance,
    }),
    githubConnections,
    selectedVcsConnectionId: githubIntegrationId || undefined,
    requestConfigure: () => {
      // Manage which repositories the GitHub App can access, on GitHub itself.
      window.open(githubInstallationUrl(github.githubIntegration.data), "_blank", "noopener,noreferrer");
    },
    integrationDialogs: connect.dialogs,
    repositories: github.repositories,
    repositoriesLoading: isRepositoryListLoading({
      savedIntegrationId: onboarding?.vcsIntegrationId,
      selectedIntegrationId: githubIntegrationId,
      connectionsLoading: connect.connectionsLoading,
      repositoriesLoading: github.repositoriesLoading,
    }),
    repositoriesError: github.repositoriesError,
    canConfigureWorkspace: canConfigureWorkspace(canAct),
    saving: saving || installer.isInstalling || createIntake.isPending || createPRFeedbackHandler.isPending,
    ...saves,
    finish: finishSetup,
    // Fresh organization + first workspace: the redesigned split-pane look.
    initialOnboarding: args.factory?.onboarding?.initial === true,
    provisionedDestination,
    // Names the finished organization row on the GitHub stepper card.
    githubOwner,
  };
}
