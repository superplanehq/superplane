import type { FactoriesFactory } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import {
  fetchFactoryAutomations,
  useCreateFactoryLine,
  useSelectFactoryVcsProviderRepository,
  useUpdateFactory,
} from "@/hooks/useFactoryData";
import { fetchFactoryIntakes, useCreateFactoryIntake, useDeleteFactoryIntake } from "@/hooks/useFactoryIntakeData";
import { fetchFactoryPRFeedbackHandlers, useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";
import { resolveGithubDefaultBranch } from "@/hooks/useIntegrations";
import { useUpdateOrganization } from "@/hooks/useOrganizationData";
import { getApiErrorMessage } from "@/lib/errors";
import { FEATURE_ORGANIZATION_BYOK } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import { AGENT_PROVIDER_IDS, isHostedAgentReady } from "./onboardingAgentReadiness";
import type { IntegrationId, IssuesChoiceId, WizardStepId } from "./onboardingFixtures";
import { useOnboardingModelSource } from "./onboardingModelSource";
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
import { useOnboardingAgentContext } from "./useOnboardingAgentPlan";
import { useOnboardingJiraBinding } from "./useOnboardingJiraBinding";
import {
  useOnboardingSetupState,
  type InitialOnboardingSetupState,
  type OnboardingSetupApi,
} from "./useOnboardingSetupState";

const ONBOARDING_INTEGRATIONS = ["jira", ...AGENT_PROVIDER_IDS];

// Agent keys stay unselected so a new workspace can use the canonical
// SuperPlane template when hosted credit is available.
const ONBOARDING_MANUAL_SELECTIONS = [...AGENT_PROVIDER_IDS] as const;

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
    if (selections.jira?.ready) ready.add("jira");
    for (const name of AGENT_PROVIDER_IDS) {
      if (selections[name]?.ready) ready.add(name);
    }
    return ready;
  }, [selections]);
  return { selections, connected, setSelections };
}

function initialSetupState(onboarding: FactoriesFactory["onboarding"]): InitialOnboardingSetupState {
  const appRepository = onboarding?.appRepository || null;
  return {
    vcsHost: onboarding?.vcsIntegrationId ? "github" : null,
    selectedRepo: appRepository,
    issuesRepo: onboarding?.backlogRepository || appRepository,
    issuesChoice: localIssuesSource(onboarding?.issuesSource),
  };
}

function useRestoreIntegrationReadiness(setup: OnboardingSetupApi, selections: IntegrationSelections) {
  const { vcsHost, selectVcsHost, setAgent } = setup;
  useEffect(() => {
    if (selections.github?.ready && vcsHost !== "github") selectVcsHost("github");
  }, [selections.github?.ready, selectVcsHost, vcsHost]);
  useEffect(() => {
    if (!selections.claude?.ready) return;
    setAgent("claude-code");
  }, [selections.claude?.ready, setAgent]);
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
  return { saveName, saveIssues };
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

/** The mutation hooks the page model saves and provisions through. */
function useOnboardingMutations(organizationId: string, factoryId: string) {
  return {
    updateFactory: useUpdateFactory(organizationId, factoryId),
    selectGitHubRepository: useSelectFactoryVcsProviderRepository(organizationId, factoryId, "github"),
    updateOnboarding: useFactoryOnboarding(organizationId, factoryId),
    updateOrganization: useUpdateOrganization(organizationId),
    createLine: useCreateFactoryLine(organizationId, factoryId),
    createIntake: useCreateFactoryIntake(organizationId, factoryId),
    deleteIntake: useDeleteFactoryIntake(organizationId, factoryId),
    createPRFeedbackHandler: useCreateFactoryPRFeedbackHandler(organizationId, factoryId),
    installer: useInstallFactory({ organizationId }),
  };
}

type OnboardingGithubSavesAndFinishArgs = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
  onboardingEntryPath?: string | null;
  reresolveWorkspace: OnboardingWorkspaceResolution | null;
  setup: OnboardingSetupApi;
  integrations: ReturnType<typeof useIntegrationSelections>;
  connect: ReturnType<typeof useIntegrationConnectDialog>;
  searchParams: URLSearchParams;
  openSection: WizardStepId;
  setOpenSection: (section: WizardStepId) => void;
  mutations: ReturnType<typeof useOnboardingMutations>;
  setSaving: (saving: boolean) => void;
  setProvisionedDestination: (destination: OnboardingDestination | null) => void;
  agent: ReturnType<typeof useOnboardingAgentContext>;
};

function useOnboardingGithubSavesAndFinish(args: OnboardingGithubSavesAndFinishArgs) {
  const {
    updateFactory,
    updateOnboarding,
    updateOrganization,
    createLine,
    createIntake,
    deleteIntake,
    installer,
    createPRFeedbackHandler,
  } = args.mutations;
  const githubIntegrationId = args.integrations.selections.github?.ready ? args.integrations.selections.github.id : "";
  const jira = useOnboardingJiraBinding(args.organizationId, args.factoryId, args.integrations.selections.jira);
  const takenNames = useMemo(
    () => otherWorkspaceNames(args.factories, args.factoryId),
    [args.factories, args.factoryId],
  );
  const saves = useSectionSaves({
    setup: args.setup,
    selections: args.integrations.selections,
    setSaving: args.setSaving,
    factoryName: args.factory?.name ?? "",
    takenNames,
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
  });
  const githubOwner = args.setup.selectedRepo?.split("/")[0];
  const finish = useFinishOnboarding({
    ...args,
    selections: args.integrations.selections,
    setSaving: args.setSaving,
    takenNames,
    updateFactory: updateFactory.mutateAsync,
    updateOnboarding: updateOnboarding.mutateAsync,
    installFactory: installer.installFactory,
    createLine: createLine.mutateAsync,
    listIntakes: () => fetchFactoryIntakes(args.organizationId, args.factoryId),
    createIntake: createIntake.mutateAsync,
    deleteIntake: deleteIntake.mutateAsync,
    listPRFeedbackHandlers: () => fetchFactoryPRFeedbackHandlers(args.organizationId, args.factoryId),
    createPRFeedbackHandler: createPRFeedbackHandler.mutateAsync,
    listApps: () => fetchFactoryAutomations(args.organizationId, args.factoryId),
    resolveDefaultBranch: (repository: string) =>
      resolveGithubDefaultBranch(args.organizationId, githubIntegrationId, repository),
    remainingCreditCents: args.agent.remainingCreditCents,
    hostedModelsLoading: args.agent.hostedModelsLoading,
    plan: args.agent.plan,
    githubOwner,
    jiraProjectId: jira.jiraProjectId,
    jiraCompletion: jira.jiraCompletion,
    updateOrganization: async (identity) => {
      const response = await updateOrganization.mutateAsync(identity);
      return response.data?.organization?.metadata?.slug;
    },
    onProvisioned: args.setProvisionedDestination,
  });
  const finishSetup = useFinishSetupAction({ ...args, finish });
  const selectCatalogRepository = async (repository: {
    repositoryId?: string;
    fullName?: string;
  }): Promise<boolean> => {
    if (!repository.repositoryId || !repository.fullName) return false;
    const repositoryId = repository.repositoryId;
    const fullName = repository.fullName;
    return runSave(args.setSaving, async () => {
      const factory = await args.mutations.selectGitHubRepository.mutateAsync(repositoryId);
      const integrationId = factory.onboarding?.vcsIntegrationId;
      if (!integrationId) throw new Error("GitHub repository selection returned no integration");
      args.integrations.setSelections((current) => ({
        ...current,
        github: { id: integrationId, name: "GitHub", ready: true },
      }));
      args.setup.selectVcsHost("github");
      args.setup.selectRepo(fullName);
    });
  };

  return {
    githubIntegrationId,
    saves,
    githubOwner,
    finishSetup,
    selectCatalogRepository,
    jira,
    installer,
    createIntake,
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
  const bringYourOwnKey = useExperimentalFeature(args.organizationId);
  const [agentCredentialChoice, setAgentCredentialChoice] = useOnboardingModelSource(args.factoryId);
  const onboarding = args.factory?.onboarding;
  const integrations = useIntegrationSelections(onboarding);
  const preferOwnKey = bringYourOwnKey.has(FEATURE_ORGANIZATION_BYOK) && agentCredentialChoice !== "hosted";
  const agent = useOnboardingAgentContext(args.organizationId, integrations.connected, preferOwnKey);
  const setup = useOnboardingSetupState(args.factory?.name ?? "", {
    connected: integrations.connected,
    remainingCreditCents: agent.remainingCreditCents,
    simulateDiscovery: false,
    initial: initialSetupState(onboarding),
  });
  useRestoreIntegrationReadiness(setup, integrations.selections);
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
  const mutations = useOnboardingMutations(args.organizationId, args.factoryId);
  const wired = useOnboardingGithubSavesAndFinish({
    ...args,
    reresolveWorkspace: args.reresolveWorkspace ?? null,
    setup,
    integrations,
    connect,
    searchParams,
    openSection,
    setOpenSection,
    mutations,
    setSaving,
    setProvisionedDestination,
    agent,
  });

  return {
    setup,
    hostedAgentReady: isHostedAgentReady(agent.plan),
    hostedModelsAvailable: agent.hostedModelsAvailable,
    hostedModelsAvailableLoading: agent.hostedModelsAvailableLoading,
    bringYourOwnKey: bringYourOwnKey.has(FEATURE_ORGANIZATION_BYOK),
    bringYourOwnKeyLoading: bringYourOwnKey.isLoading,
    agentCredentialChoice,
    setAgentCredentialChoice,
    agentLoading: agent.hostedModelsLoading,
    openSection,
    setOpenSection,
    requestConnect: connect.requestConnect,
    selectCatalogRepository: wired.selectCatalogRepository,
    integrationDialogs: connect.dialogs,
    canConfigureWorkspace: canConfigureWorkspace(canAct),
    saving: saving || wired.installer.isInstalling || wired.createIntake.isPending,
    ...wired.saves,
    finish: wired.finishSetup,
    provisionedDestination,
    // Names the finished organization row on the GitHub stepper card.
    githubOwner: wired.githubOwner,
    ...wired.jira,
  };
}
