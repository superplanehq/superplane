import type { MeGitHubOnboardingRepository } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { FEATURE_FACTORY_JIRA_INTAKE } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import type { FirstRunTicketSource } from "./first-run/firstRunTypes";
import {
  canAnalyzeTicketSource,
  DEFAULT_TICKET_SOURCE,
  issuesChoiceForTicketSource,
  ticketSourceFromIssuesChoice,
} from "./first-run/firstRunTicketSource";
import { type IntegrationId, type IssuesChoiceId, type WizardStepId } from "./onboardingFixtures";
import {
  onboardingAgentGate,
  type OnboardingAgentCredentialChoice,
  type OnboardingAgentGate,
} from "./onboardingAgentReadiness";
import { isWizardStepId } from "./onboardingStatus";
import { useGitHubOnboarding } from "./useGitHubOnboarding";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

export type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;
export type FirstRunScreen = "welcome" | "connect" | "choose" | "tickets" | "agent";
type FirstRunBlockingAction =
  | "opening-github"
  | "saving-repository"
  | "saving-ticket-source"
  | "connecting-jira"
  | "finishing-setup";

export { DEFAULT_TICKET_SOURCE };
const SCREEN_FOR_STEP: Record<WizardStepId, FirstRunScreen> = {
  vcs: "connect",
  repo: "choose",
  issues: "tickets",
  agent: "agent",
  name: "agent",
};
const STEP_FOR_SCREEN: Partial<Record<FirstRunScreen, WizardStepId>> = {
  connect: "vcs",
  choose: "repo",
  tickets: "issues",
  agent: "agent",
};

function initialFirstRunScreen(searchParams: URLSearchParams): FirstRunScreen {
  const requestedStep = searchParams.get("step");
  if (isWizardStepId(requestedStep)) return SCREEN_FOR_STEP[requestedStep];
  return "welcome";
}

function screenWithoutAgent(screen: FirstRunScreen, agentGate: OnboardingAgentGate): FirstRunScreen {
  if (screen !== "agent" || agentGate === "show" || agentGate === "first") return screen;
  return "tickets";
}

// The model source comes before the backlog, so the ticket screen waits for it.
function screenWithModelSource(
  screen: FirstRunScreen,
  agentGate: OnboardingAgentGate,
  credentialChoice: OnboardingAgentCredentialChoice | null,
): FirstRunScreen {
  if (screen !== "tickets" || agentGate !== "first" || credentialChoice) return screen;
  return "agent";
}

function useFirstRunBlockingAction() {
  const lock = useRef(false);
  const [action, setAction] = useState<FirstRunBlockingAction | null>(null);
  const begin = useCallback((next: FirstRunBlockingAction): boolean => {
    if (lock.current) return false;
    lock.current = true;
    setAction(next);
    return true;
  }, []);
  const finish = useCallback(() => {
    lock.current = false;
    setAction(null);
  }, []);
  const run = useCallback(
    async (next: FirstRunBlockingAction, operation: () => Promise<void>) => {
      if (!begin(next)) return;
      try {
        await operation();
      } finally {
        finish();
      }
    },
    [begin, finish],
  );
  const runUntilNavigation = useCallback(
    async (next: FirstRunBlockingAction, operation: () => Promise<boolean>) => {
      if (!begin(next)) return;
      try {
        const navigationStarted = await operation();
        if (!navigationStarted) finish();
      } catch (error) {
        finish();
        throw error;
      }
    },
    [begin, finish],
  );
  return { action, busy: action !== null, begin, finish, setAction, run, runUntilNavigation };
}

function useGitHubConnectionState(organizationId: string) {
  const [searchParams] = useSearchParams();
  const onboarding = useGitHubOnboarding(organizationId);
  const repositories = onboarding.data?.repositories ?? [];
  return {
    onboarding,
    identity: onboarding.data?.identity,
    repositories,
    pendingOrganizations: (onboarding.data?.pendingRequests ?? [])
      .map((request) => request.accountLogin?.trim())
      .filter((organization): organization is string => Boolean(organization)),
    synchronizing: Boolean(onboarding.data?.synchronizing),
    appConfigured: Boolean(onboarding.data?.appConfigured),
    initialScreen: initialFirstRunScreen(searchParams),
  };
}

function screenWithoutIncompleteJira(
  screen: FirstRunScreen,
  agentGate: OnboardingAgentGate,
  issuesChoice: IssuesChoiceId | null,
  jiraProjectId: string,
): FirstRunScreen {
  if (screen !== "agent" || agentGate !== "show") return screen;
  if (issuesChoice === "jira" && !jiraProjectId) return "tickets";
  return screen;
}

function useFirstRunNavigation(
  model: OnboardingPageModel,
  agentGate: OnboardingAgentGate,
  connection: ReturnType<typeof useGitHubConnectionState>,
) {
  const [openedScreen, setOpenedScreen] = useState<FirstRunScreen>(connection.initialScreen);
  const openStep = useRef(model.openSection);

  useEffect(() => {
    if (model.openSection === openStep.current) return;
    openStep.current = model.openSection;
    setOpenedScreen(SCREEN_FOR_STEP[model.openSection]);
  }, [model.openSection]);

  const goToScreen = (next: FirstRunScreen) => {
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      openStep.current = step;
      model.setOpenSection(step);
    }
    setOpenedScreen(next);
  };

  const githubAccessRequired =
    !connection.onboarding.isPending &&
    openedScreen !== "welcome" &&
    (!connection.identity || connection.repositories.length === 0);
  const availableScreen = githubAccessRequired ? "connect" : openedScreen;

  return {
    screen: screenWithoutIncompleteJira(
      screenWithModelSource(screenWithoutAgent(availableScreen, agentGate), agentGate, model.agentCredentialChoice),
      agentGate,
      model.setup.issuesChoice,
      model.jiraProjectId,
    ),
    goToScreen,
  };
}

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
}

function openGitHubWindow(): Window | null {
  const popup = window.open("about:blank", "_blank");
  if (popup) popup.opener = null;
  return popup;
}

function navigateGitHubWindow(popup: Window | null, url: string) {
  if (popup) {
    popup.location.replace(url);
    return;
  }
  window.location.assign(url);
}

function selectedIssuesChoice(model: OnboardingPageModel, jiraAvailable: boolean): IssuesChoiceId | null {
  const ticketSource = ticketSourceFromIssuesChoice(model.setup.issuesChoice);
  const issuesChoice = issuesChoiceForTicketSource(ticketSource);
  if (issuesChoice === "jira" && !jiraAvailable) return null;
  if (
    !issuesChoice ||
    !canAnalyzeTicketSource({
      ticketSource,
      jiraConnected: model.setup.connected.has("jira"),
      jiraProjectId: model.jiraProjectId,
    })
  ) {
    return null;
  }
  return issuesChoice;
}

function useFirstRunCommands(args: {
  model: OnboardingPageModel;
  agentGate: OnboardingAgentGate;
  connection: ReturnType<typeof useGitHubConnectionState>;
  navigation: ReturnType<typeof useFirstRunNavigation>;
  blocking: ReturnType<typeof useFirstRunBlockingAction>;
  jiraAvailable: boolean;
}) {
  const { model, agentGate, connection, navigation, blocking, jiraAvailable } = args;
  const continueFromRepository = () =>
    blocking.run("saving-repository", async () => {
      const repository = connection.repositories.find((candidate) => candidate.fullName === model.setup.selectedRepo);
      if (!repository) return;
      model.setup.commitRepoStep();
      if (await model.selectCatalogRepository(repository)) {
        navigation.goToScreen(agentGate === "first" ? "agent" : "tickets");
      }
    });
  const continueFromTickets = () =>
    blocking.run("saving-ticket-source", async () => {
      const issuesChoice = selectedIssuesChoice(model, jiraAvailable);
      if (!issuesChoice) return;
      model.setup.setIssuesChoice(issuesChoice);
      model.setup.commitIssuesStep();
      if (!(await model.saveIssues(issuesChoice))) return;
      if (agentGate === "pending") return;
      if (agentGate === "show") return navigation.goToScreen("agent");
      blocking.setAction("finishing-setup");
      await model.finish(issuesChoice);
    });
  const connectGitHub = () =>
    blocking.runUntilNavigation("opening-github", async () => {
      if (!connection.identity) {
        const returnPath = `${window.location.pathname}${window.location.search}`;
        window.location.assign(linkedAccountConnectHref("github", returnPath));
        return true;
      }
      const popup = openGitHubWindow();
      try {
        const url = await connection.onboarding.startInstallation.mutateAsync();
        navigateGitHubWindow(popup, url);
      } catch (error) {
        popup?.close();
        throw error;
      }
      return false;
    });
  const connectJira = () =>
    blocking.runUntilNavigation("connecting-jira", async () => {
      if (!jiraAvailable) return false;
      model.setup.setIssuesChoice("jira");
      if (!(await model.saveIssues("jira"))) return false;
      await waitForBrowserPaint();
      return model.requestConnect("jira");
    });
  const finishSetup = () =>
    blocking.run("finishing-setup", async () => {
      await model.finish();
    });
  const continueFromAgent = () => {
    if (agentGate === "first") return navigation.goToScreen("tickets");
    return finishSetup();
  };
  const configureGitHubAccess = () => {
    const selected = connection.repositories.find((repository) => repository.fullName === model.setup.selectedRepo);
    const installationId = selected?.installationId ?? connection.repositories[0]?.installationId;
    if (!installationId) return Promise.resolve();
    return blocking.run("opening-github", async () => {
      const popup = openGitHubWindow();
      try {
        const url = await connection.onboarding.configureInstallation.mutateAsync(installationId);
        navigateGitHubWindow(popup, url);
      } catch (error) {
        popup?.close();
        throw error;
      }
    });
  };
  const selectTicketSource = (source: FirstRunTicketSource) => {
    if (source === "jira" && !jiraAvailable) return;
    const issuesChoice = issuesChoiceForTicketSource(source);
    if (issuesChoice) model.setup.setIssuesChoice(issuesChoice);
  };
  return {
    connectGitHub,
    connectJira,
    continueFromRepository,
    continueFromTickets,
    continueFromAgent,
    configureGitHubAccess,
    selectTicketSource,
  };
}

function useRepositoryErrorToast(error: unknown) {
  const reported = useRef<unknown>(null);
  useEffect(() => {
    if (!error || reported.current === error) return;
    reported.current = error;
    showErrorToast(getApiErrorMessage(error, "Failed to load repositories"));
  }, [error]);
}

export function shouldClearSavedJiraChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  jiraAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  if (args.featureLoading || args.jiraAvailable || args.issuesChoice !== "jira") return false;
  return args.organizationReady;
}

export type SavedJiraChoiceBlock = "loading" | "lookup-failed";

/** A saved Jira choice cannot continue until the feature lookup confirms Jira. */
export function savedJiraChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  jiraAvailable: boolean;
  organizationReady: boolean;
}): SavedJiraChoiceBlock | null {
  if (args.jiraAvailable || args.issuesChoice !== "jira") return null;
  if (args.featureLoading) return "loading";
  if (!args.organizationReady) return "lookup-failed";
  return null;
}

export function useFirstRunSetupFlow(model: OnboardingPageModel) {
  const { organizationId } = useFactoriesLayout();
  const blocking = useFirstRunBlockingAction();
  const connection = useGitHubConnectionState(organizationId);
  const jiraFeature = useExperimentalFeature(organizationId);
  const jiraFeatureLoading = jiraFeature.isLoading;
  const jiraAvailable = !jiraFeatureLoading && jiraFeature.has(FEATURE_FACTORY_JIRA_INTAKE);
  const agentGate = onboardingAgentGate({
    hostedModelsAvailable: model.hostedModelsAvailable,
    hostedModelsAvailableLoading: model.hostedModelsAvailableLoading,
    bringYourOwnKey: model.bringYourOwnKey,
    bringYourOwnKeyLoading: model.bringYourOwnKeyLoading,
  });
  const navigation = useFirstRunNavigation(model, agentGate, connection);
  const commands = useFirstRunCommands({ model, agentGate, connection, navigation, blocking, jiraAvailable });
  useRepositoryErrorToast(connection.onboarding.error);
  useEffect(() => {
    if (navigation.screen !== "connect" || !connection.identity || connection.repositories.length === 0) return;
    navigation.goToScreen("choose");
  }, [connection.identity, connection.repositories.length, navigation]);
  // A saved Jira choice is not valid when the organization does not have the
  // Jira intake feature. Clear it only after the organization lookup confirms
  // the feature is off. A failed lookup has no organization data and must not
  // replace the saved choice with the GitHub Issues default.
  const issuesChoice = model.setup.issuesChoice;
  const setIssuesChoice = model.setup.setIssuesChoice;
  const jiraChoiceArgs = {
    issuesChoice,
    featureLoading: jiraFeatureLoading,
    jiraAvailable,
    organizationReady: jiraFeature.organizationReady,
  };
  const clearSavedJiraChoice = shouldClearSavedJiraChoice(jiraChoiceArgs);
  const jiraChoiceBlock = savedJiraChoiceBlock(jiraChoiceArgs);
  useEffect(() => {
    if (!clearSavedJiraChoice) return;
    setIssuesChoice(null);
  }, [clearSavedJiraChoice, setIssuesChoice]);
  return {
    ...navigation,
    ...commands,
    // The ticket screen is the last screen, so it finishes setup.
    ticketsFinishSetup: agentGate === "skip" || agentGate === "first",
    skipAgentScreen: agentGate === "skip",
    agentBeforeTickets: agentGate === "first",
    credentialChoice: model.agentCredentialChoice,
    selectCredentialChoice: model.setAgentCredentialChoice,
    agentGatePending: agentGate === "pending",
    ticketSource: ticketSourceFromIssuesChoice(model.setup.issuesChoice),
    jiraAvailable,
    jiraFeatureLoading,
    jiraChoiceBlock,
    repositories: connection.repositories.map((repository) => repository.fullName).filter(Boolean) as string[],
    repositoryCatalog: connection.repositories as MeGitHubOnboardingRepository[],
    repositoriesLoading: connection.onboarding.isPending,
    identityConnected: Boolean(connection.identity),
    githubLogin: connection.identity?.login ?? "",
    pendingOrganizations: connection.pendingOrganizations,
    synchronizing: connection.synchronizing,
    appConfigured: connection.appConfigured,
    connectError: connection.onboarding.error
      ? getApiErrorMessage(connection.onboarding.error, "SuperPlane could not load GitHub access")
      : undefined,
    blockingAction: blocking.action,
    busy: blocking.busy || model.saving,
  };
}

export type FirstRunSetupFlow = ReturnType<typeof useFirstRunSetupFlow>;
export type { IntegrationId };
