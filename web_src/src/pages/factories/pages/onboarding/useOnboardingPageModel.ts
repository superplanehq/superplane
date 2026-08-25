import type { FactoriesFactory } from "@/api-client";
import { usePermissions } from "@/contexts/usePermissions";
import {
  useCreateFactoryLine,
  useCreateWorkOrder,
  useDispatchWorkOrder,
  useUpdateFactory,
} from "@/hooks/useFactoryData";
import { useIntegration, useIntegrationResources } from "@/hooks/useIntegrations";
import { useOrganizationLLMSpend } from "@/hooks/useOrganizationLLMSpend";
import { getApiErrorMessage } from "@/lib/errors";
import { githubInstallationUrl } from "@/lib/githubInstallation";
import { showErrorToast } from "@/lib/toast";
import { parseWorkOrderMetric } from "@/pages/factories/lib/workOrderUsage";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useIntegrationConnectDialog } from "@/pages/home/useIntegrationConnectDialog";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";

import { factorySetupPath } from "../../lib/factoryPagePaths";
import { AGENT_PROVIDER_IDS } from "./onboardingAgentReadiness";
import type { IntegrationId, WizardStepId } from "./onboardingFixtures";
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
  const spend = useOrganizationLLMSpend(organizationId);
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

export function useOnboardingPageModel(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory | null;
  factories: FactoriesFactory[];
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
  const githubIntegrationId = integrations.selections.github?.ready ? integrations.selections.github.id : "";
  const githubConnections = useOnboardingGithubConnections({
    integrationData: connect.integrationData,
    openSection,
    selectNewest: searchParams.get("pick") === "newest",
    selections: integrations.selections,
    selectInstance: connect.selectInstance,
    onConnectionSelected: () => {
      setup.selectVcsHost("github");
      setOpenSection("repo");
    },
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
    createWorkOrder: createWorkOrder.mutateAsync,
    dispatchWorkOrder: dispatchWorkOrder.mutateAsync,
    remainingCreditCents: agent.remainingCreditCents,
    hostedModelsLoading: agent.hostedModelsLoading,
    plan: agent.plan,
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
      window.open(githubInstallationUrl(github.githubIntegration.data), "_blank", "noopener,noreferrer");
    },
    integrationDialogs: connect.dialogs,
    repositories: github.repositories,
    repositoriesLoading: github.repositoriesLoading,
    repositoriesError: github.repositoriesError,
    canConfigureWorkspace: canConfigureWorkspace(canAct),
    saving: saving || installer.isInstalling || createWorkOrder.isPending,
    ...saves,
    finish: finishSetup,
  };
}
