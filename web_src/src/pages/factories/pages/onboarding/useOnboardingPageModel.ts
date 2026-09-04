import type { FactoriesFactory, OrganizationsIntegration } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { factoryQueryKeys, fetchFactoryApps, useCreateFactoryLine, useUpdateFactory } from "@/hooks/useFactoryData";
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
import type { NavigateFunction } from "react-router";
import { useNavigate, useSearchParams } from "react-router";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import { AGENT_PROVIDER_IDS, isHostedAgentReady } from "./onboardingAgentReadiness";
import {
  githubIntegrationOwner,
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
import { useFinishOnboarding } from "./useFinishOnboarding";
import { useFinishSetupAction } from "./useFinishSetupAction";
import { useOnboardingAgentPlan } from "./useOnboardingAgentPlan";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { useOnboardingGithubConnections } from "./useSelectNewGithubConnection";

const ONBOARDING_INTEGRATIONS = ["github", ...AGENT_PROVIDER_IDS];

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
  return useOnboardingAgentPlan(organizationId, connected, remainingCreditCents);
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

/**
 * Copies the cached factory list and factory detail from the old organization
 * slug to the new one, so the wizard does not fall back to a full-screen
 * loading state while the workspace re-resolves under the new slug.
 */
function seedFactoryQueriesForNewSlug(
  queryClient: QueryClient,
  oldSlug: string,
  nextSlug: string,
  factoryId: string,
): void {
  const list = queryClient.getQueryData(factoryQueryKeys.list(oldSlug));
  if (list !== undefined) {
    queryClient.setQueryData(factoryQueryKeys.list(nextSlug), list);
  }

  const detail = queryClient.getQueryData(factoryQueryKeys.detail(oldSlug, factoryId));
  if (detail !== undefined) {
    queryClient.setQueryData(factoryQueryKeys.detail(nextSlug, factoryId), detail);
  }
}

/** Marks the seeded queries stale, so they refetch in the background under the new slug. */
function invalidateFactoryQueriesForNewSlug(queryClient: QueryClient, nextSlug: string, factoryId: string): void {
  void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.list(nextSlug) });
  void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.detail(nextSlug, factoryId) });
}

/**
 * Moves the wizard from the vcs step to the repo step after the organization
 * is renamed from the GitHub owner.
 *
 * During initial onboarding the workspace is resolved by
 * `OrganizationOnboardingRedirect` under the organization slug, so renaming
 * the organization invalidates that resolution. Re-resolving the workspace in
 * place (instead of reloading the page) lets the wizard advance with a plain
 * client-side navigation.
 */
export async function advanceAfterGithubConnect(args: {
  onboardingEntryPath?: string | null;
  organizationId: string;
  nextSlug: string;
  factoryId: string;
  factoryKey: string;
  navigate: NavigateFunction;
  reresolveWorkspace: OnboardingWorkspaceResolution | null;
  queryClient: QueryClient;
}): Promise<void> {
  const nextPath = onboardingStepPath(
    args.onboardingEntryPath ?? factorySetupPath(args.nextSlug, args.factoryKey),
    "repo",
  );

  if (!args.onboardingEntryPath) {
    args.navigate(nextPath, { replace: true });
    return;
  }

  if (!args.reresolveWorkspace) {
    // No re-resolution callback is available (for example, in Storybook or a
    // test that renders the wizard outside the onboarding route). Fall back
    // to a full reload so the workspace still resolves under the new slug.
    window.location.replace(nextPath);
    return;
  }

  seedFactoryQueriesForNewSlug(args.queryClient, args.organizationId, args.nextSlug, args.factoryId);
  try {
    await args.reresolveWorkspace();
  } catch {
    // The organization already renamed, so the mounted wizard is bound to the
    // now-invalid old slug. Re-resolving in place failed, which would strand
    // the user on that slug. Fall back to a full reload so the workspace
    // resolves under the new slug and onboarding can continue.
    window.location.replace(nextPath);
    return;
  }
  invalidateFactoryQueriesForNewSlug(args.queryClient, args.nextSlug, args.factoryId);
  args.navigate(nextPath, { replace: true });
}

function useOnboardingGithubConnectionSelected(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  onboardingEntryPath?: string | null;
  reresolveWorkspace: OnboardingWorkspaceResolution | null;
  selectNewest: boolean;
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

    try {
      await args.updateOnboarding({ vcsIntegrationId: integrationId });
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Could not save the GitHub connection"));
      return;
    }

    const owner = githubIntegrationOwner(integration);
    if (!owner || !shouldNameOrganizationFromGitHub(args.factory, args.selectNewest)) return;

    try {
      const nextSlug = await nameOrganizationFromGitHubOwner({
        owner,
        currentSlug: args.organizationId,
        update: async (identity) => {
          const response = await args.updateOrganization.mutateAsync(identity);
          return response.data?.organization?.metadata?.slug;
        },
      });
      if (!nextSlug) return;

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
  const onConnectionSelected = useOnboardingGithubConnectionSelected({ ...args, selectNewest });

  return useOnboardingGithubConnections({
    integrationData: args.integrationData,
    openSection: args.openSection,
    selectNewest,
    selections: args.selections,
    selectInstance: args.selectInstance,
    onConnectionSelected,
  });
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
  });

  const [saving, setSaving] = useState(false);
  const updateFactory = useUpdateFactory(args.organizationId, args.factoryId);
  const updateOnboarding = useFactoryOnboarding(args.organizationId, args.factoryId);
  const updateOrganization = useUpdateOrganization(args.organizationId);
  const createLine = useCreateFactoryLine(args.organizationId, args.factoryId);
  const createIntake = useCreateFactoryIntake(args.organizationId, args.factoryId);
  const createPRFeedbackHandler = useCreateFactoryPRFeedbackHandler(args.organizationId, args.factoryId);
  const installer = useInstallFactory({ organizationId: args.organizationId });
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
    hostedAgentReady: isHostedAgentReady({ hostedModelsLoading: agent.hostedModelsLoading, plan: agent.plan }),
    openSection,
    setOpenSection,
    requestConnect: connect.requestConnect,
    requestPrivateGitHubConnect: connect.requestPrivateGitHubConnect,
    offersPrivateGitHubAppSetup: connect.offersPrivateGitHubAppSetup,
    createVcsConnection: () => connect.createNew("github"),
    selectVcsConnection: (integrationId: string) => connect.selectInstance("github", integrationId),
    githubConnections,
    selectedVcsConnectionId: githubIntegrationId || undefined,
    requestConfigure: () => {
      // Manage which repositories the GitHub App can access, on GitHub itself.
      window.open(githubInstallationUrl(github.githubIntegration.data), "_blank", "noopener,noreferrer");
    },
    integrationDialogs: connect.dialogs,
    repositories: github.repositories,
    repositoriesLoading: github.repositoriesLoading,
    repositoriesError: github.repositoriesError,
    canConfigureWorkspace: canConfigureWorkspace(canAct),
    saving: saving || installer.isInstalling || createIntake.isPending || createPRFeedbackHandler.isPending,
    ...saves,
    finish: finishSetup,
  };
}
