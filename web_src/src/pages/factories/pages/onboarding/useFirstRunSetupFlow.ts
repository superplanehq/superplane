import type { MeVcsProviderRepository } from "@/api-client";
import { useIntakeCatalogAvailability } from "@/hooks/useIntakeCatalogAvailability";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
<<<<<<< HEAD
import { getApiErrorMessage } from "@/lib/errors";
import { isIntakeSelectable } from "@/lib/intakeCatalog";
import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, useState } from "react";
=======
import { FEATURE_FACTORY_JIRA_INTAKE, FEATURE_FACTORY_LINEAR_INTAKE } from "@/lib/experimentalFeatures";
import { useEffect, useRef, useState } from "react";
>>>>>>> origin/main
import { useLocation, useSearchParams } from "react-router";

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
import { onboardingStepPath } from "./onboardingStepPath";
import {
  clearGitHubInstallStarted,
  githubAccessKeys,
  markGitHubInstallStarted,
  type GitHubInstallScope,
} from "./githubInstallReturn";
import { githubConnectReturnPath } from "./onboardingGitHubConnect";
import { useFirstRunBlockingAction, type FirstRunBlocking } from "./useFirstRunBlockingAction";
import {
  githubOnboardingMessage,
  navigateGitHubWindow,
  openGitHubWindow,
  useFirstRunGitHub,
  type GitHubConnectionState,
} from "./useFirstRunGitHub";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

export type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;
export type FirstRunScreen = "welcome" | "connect" | "choose" | "tickets" | "agent";
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

/** Null means no step was requested, so the flow resumes from saved progress. */
function initialFirstRunScreen(searchParams: URLSearchParams): FirstRunScreen | null {
  const requestedStep = searchParams.get("step");
  if (isWizardStepId(requestedStep)) return SCREEN_FOR_STEP[requestedStep];
  return null;
}

// GitHub setup callbacks return without a step. Saved progress or a confirmed
// GitHub connection must not send the user back to the welcome screen.
function resumeFirstRunScreen(args: {
  savedStep: WizardStepId;
  githubPending: boolean;
  githubReady: boolean;
}): FirstRunScreen {
  if (args.savedStep !== "vcs") return SCREEN_FOR_STEP[args.savedStep];
  if (args.githubPending) return "connect";
  return args.githubReady ? "choose" : "welcome";
}

// A GitHub identity from sign-in is not enough. The repository screen opens
// only after the user connects GitHub for this workspace.
function screenWithGitHubConnection(
  screen: FirstRunScreen,
  github: { pending: boolean; identity: boolean; ready: boolean; stayOnConnect: boolean },
): FirstRunScreen {
  if (github.pending || screen === "welcome") return screen;
  if (!github.identity) return "connect";
  if (screen === "connect" && github.ready && !github.stayOnConnect) return "choose";
  if (screen === "choose" && !github.ready) return "connect";
  return screen;
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

function screenWithoutIncompleteBacklog(
  screen: FirstRunScreen,
  agentGate: OnboardingAgentGate,
  issuesChoice: IssuesChoiceId | null,
  jiraProjectId: string,
  linearProjectIds: string[],
): FirstRunScreen {
  if (screen !== "agent" || agentGate !== "show") return screen;
  if (issuesChoice === "jira" && !jiraProjectId) return "tickets";
  if (issuesChoice === "linear" && linearProjectIds.length === 0) return "tickets";
  return screen;
}

function useFirstRunNavigation(
  model: OnboardingPageModel,
  agentGate: OnboardingAgentGate,
  connection: GitHubConnectionState,
  githubReady: boolean,
) {
  const [searchParams] = useSearchParams();
  const [requestedScreen, setRequestedScreen] = useState<FirstRunScreen | null>(() =>
    initialFirstRunScreen(searchParams),
  );
  // Back from repository selection must not forward to it again.
  const [stayOnConnect, setStayOnConnect] = useState(false);
  const openStep = useRef(model.openSection);
  const openedScreen =
    requestedScreen ??
    resumeFirstRunScreen({
      savedStep: model.openSection,
      githubPending: connection.onboarding.isPending,
      githubReady,
    });

  useEffect(() => {
    if (model.openSection === openStep.current) return;
    openStep.current = model.openSection;
    setRequestedScreen(SCREEN_FOR_STEP[model.openSection]);
  }, [model.openSection]);

  const availableScreen = screenWithGitHubConnection(openedScreen, {
    pending: connection.onboarding.isPending,
    identity: Boolean(connection.identity),
    ready: githubReady,
    stayOnConnect,
  });

  const goToScreen = (next: FirstRunScreen) => {
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      openStep.current = step;
      model.setOpenSection(step);
    }
    setStayOnConnect(next === "connect" && availableScreen === "choose");
    setRequestedScreen(next);
  };

  return {
    screen: screenWithoutIncompleteBacklog(
      screenWithModelSource(screenWithoutAgent(availableScreen, agentGate), agentGate, model.agentCredentialChoice),
      agentGate,
      model.setup.issuesChoice,
      model.jiraProjectId,
      model.linearProjectIds,
    ),
    goToScreen,
  };
}

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
}

function selectedIssuesChoice(
  model: OnboardingPageModel,
  jiraAvailable: boolean,
  linearAvailable: boolean,
): IssuesChoiceId | null {
  const ticketSource = ticketSourceFromIssuesChoice(model.setup.issuesChoice);
  const issuesChoice = issuesChoiceForTicketSource(ticketSource);
  if (issuesChoice === "jira" && !jiraAvailable) return null;
  if (issuesChoice === "linear" && !linearAvailable) return null;
  if (issuesChoice === "linear" && (model.linearProjectsLoading || model.linearProjectsError)) return null;
  if (
    !issuesChoice ||
    !canAnalyzeTicketSource({
      ticketSource,
      jiraConnected: model.setup.connected.has("jira"),
      jiraProjectId: model.jiraProjectId,
      linearConnected: model.setup.connected.has("linear"),
      linearProjectIds: model.linearProjectIds,
    })
  ) {
    return null;
  }
  return issuesChoice;
}

function useFirstRunCommands(args: {
  model: OnboardingPageModel;
  agentGate: OnboardingAgentGate;
  connection: GitHubConnectionState;
  navigation: ReturnType<typeof useFirstRunNavigation>;
  blocking: FirstRunBlocking;
  jiraAvailable: boolean;
  linearAvailable: boolean;
  installScope: GitHubInstallScope;
}) {
  const { model, agentGate, connection, navigation, blocking, jiraAvailable, linearAvailable, installScope } = args;
  const location = useLocation();
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
      const issuesChoice = selectedIssuesChoice(model, jiraAvailable, linearAvailable);
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
      const returnPath = githubConnectReturnPath(onboardingStepPath(`${location.pathname}${location.search}`, "repo"));
      window.location.assign(linkedAccountConnectHref("github", returnPath));
      return true;
    });
  const selectGitHubIdentity = (userId: string) =>
    blocking.run("switching-github-account", async () => {
      await connection.onboarding.selectIdentity.mutateAsync(userId);
      model.setup.clearRepository();
    });
  const grantGitHubAccess = () =>
    blocking.run("opening-github", async () => {
      markGitHubInstallStarted(installScope, githubAccessKeys(connection.onboarding.data) ?? []);
      const popup = openGitHubWindow();
      try {
        const url = await connection.onboarding.startInstallation.mutateAsync();
        navigateGitHubWindow(popup, url);
      } catch (error) {
        clearGitHubInstallStarted(installScope);
        popup?.close();
        throw error;
      }
    });
  const connectJira = () =>
    blocking.runUntilNavigation("connecting-jira", async () => {
      if (!jiraAvailable) return false;
      model.setup.setIssuesChoice("jira");
      if (!(await model.saveIssues("jira"))) return false;
      await waitForBrowserPaint();
      return model.requestConnect("jira");
    });
  const connectLinear = () =>
    blocking.runUntilNavigation("connecting-linear", async () => {
      if (!linearAvailable) return false;
      model.setup.setIssuesChoice("linear");
      if (!(await model.saveIssues("linear"))) return false;
      await waitForBrowserPaint();
      return model.requestConnect("linear");
    });
  const finishSetup = () =>
    blocking.run("finishing-setup", async () => {
      await model.finish();
    });
  const continueFromAgent = () => {
    if (agentGate === "first") return navigation.goToScreen("tickets");
    return finishSetup();
  };
  const selectTicketSource = (source: FirstRunTicketSource) => {
    if (source === "jira" && !jiraAvailable) return;
    if (source === "linear" && !linearAvailable) return;
    const issuesChoice = issuesChoiceForTicketSource(source);
    if (issuesChoice) model.setup.setIssuesChoice(issuesChoice);
  };
  return {
    connectGitHub,
    connectJira,
    connectLinear,
    continueFromRepository,
    continueFromTickets,
    continueFromAgent,
    grantGitHubAccess,
    selectGitHubIdentity,
    selectTicketSource,
  };
}

type FlaggedIssuesChoice = "jira" | "linear";

function shouldClearSavedFlaggedChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  source: FlaggedIssuesChoice;
  featureLoading: boolean;
  available: boolean;
  organizationReady: boolean;
}): boolean {
  if (args.featureLoading || args.available || args.issuesChoice !== args.source) return false;
  return args.organizationReady;
}

export function shouldClearSavedJiraChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  jiraAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  return shouldClearSavedFlaggedChoice({
    issuesChoice: args.issuesChoice,
    source: "jira",
    featureLoading: args.featureLoading,
    available: args.jiraAvailable,
    organizationReady: args.organizationReady,
  });
}

export function shouldClearSavedLinearChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  linearAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  return shouldClearSavedFlaggedChoice({
    issuesChoice: args.issuesChoice,
    source: "linear",
    featureLoading: args.featureLoading,
    available: args.linearAvailable,
    organizationReady: args.organizationReady,
  });
}

export type SavedFlaggedChoiceBlock = "loading" | "lookup-failed";

function savedFlaggedChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  source: FlaggedIssuesChoice;
  featureLoading: boolean;
  available: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  if (args.available || args.issuesChoice !== args.source) return null;
  if (args.featureLoading) return "loading";
  if (!args.organizationReady) return "lookup-failed";
  return null;
}

/** A saved Jira choice cannot continue until the intake catalog confirms Jira. */
export function savedJiraChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  jiraAvailable: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  return savedFlaggedChoiceBlock({
    issuesChoice: args.issuesChoice,
    source: "jira",
    featureLoading: args.featureLoading,
    available: args.jiraAvailable,
    organizationReady: args.organizationReady,
  });
}

/** A saved Linear choice cannot continue until the feature lookup confirms Linear. */
export function savedLinearChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  linearAvailable: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  return savedFlaggedChoiceBlock({
    issuesChoice: args.issuesChoice,
    source: "linear",
    featureLoading: args.featureLoading,
    available: args.linearAvailable,
    organizationReady: args.organizationReady,
  });
}

export function useFirstRunSetupFlow(model: OnboardingPageModel) {
  const { organizationId, factoryId } = useFactoriesLayout();
  const setupFinished = model.provisionedDestination != null;
  const blocking = useFirstRunBlockingAction();
<<<<<<< HEAD
  const connection = useGitHubConnectionState(organizationId, setupFinished ? { poll: false } : undefined);
  const intakeCatalog = useIntakeCatalogAvailability(organizationId);
  const jiraFeatureLoading = intakeCatalog.loading;
  const jiraAvailable = isIntakeSelectable(intakeCatalog.stateOf("jira-issues"));
=======
  const { connection, githubReady, installScope, checkingGitHub } = useFirstRunGitHub({
    organizationId,
    factoryId,
    setupFinished,
    connectedBefore: Boolean(model.setup.selectedRepo),
  });
  const intakeFeatures = useExperimentalFeature(organizationId);
  const intakeFeatureLoading = intakeFeatures.isLoading;
  const jiraFeatureLoading = intakeFeatureLoading;
  const linearFeatureLoading = intakeFeatureLoading;
  const jiraAvailable = !intakeFeatureLoading && intakeFeatures.has(FEATURE_FACTORY_JIRA_INTAKE);
  const linearAvailable = !intakeFeatureLoading && intakeFeatures.has(FEATURE_FACTORY_LINEAR_INTAKE);
>>>>>>> origin/main
  const agentGate = onboardingAgentGate({
    hostedModelsAvailable: model.hostedModelsAvailable,
    hostedModelsAvailableLoading: model.hostedModelsAvailableLoading,
    bringYourOwnKey: model.bringYourOwnKey,
    bringYourOwnKeyLoading: model.bringYourOwnKeyLoading,
  });
<<<<<<< HEAD
  const navigation = useFirstRunNavigation(model, agentGate, connection);
  const commands = useFirstRunCommands({ model, agentGate, connection, navigation, blocking, jiraAvailable });
  useRepositoryErrorToast(connection.onboarding.error, !setupFinished);
  // A saved Jira choice is not valid when the intake catalog does not let the
  // organization use Jira. Clear it only after the catalog loads and confirms
  // that. A failed lookup has no catalog data and must not replace the saved
  // choice with the GitHub Issues default.
=======
  const navigation = useFirstRunNavigation(model, agentGate, connection, githubReady);
  const commands = useFirstRunCommands({
    model,
    agentGate,
    connection,
    navigation,
    blocking,
    jiraAvailable,
    linearAvailable,
    installScope,
  });
  // A saved Jira choice is not valid when the organization does not have the
  // Jira intake feature. Clear it only after the organization lookup confirms
  // the feature is off. A failed lookup has no organization data and must not
  // replace the saved choice with the GitHub Issues default.
>>>>>>> origin/main
  const issuesChoice = model.setup.issuesChoice;
  const setIssuesChoice = model.setup.setIssuesChoice;
  const organizationReady = intakeFeatures.organizationReady;
  const jiraChoiceArgs = {
    issuesChoice,
    featureLoading: jiraFeatureLoading,
    jiraAvailable,
<<<<<<< HEAD
    organizationReady: intakeCatalog.loaded,
=======
    organizationReady,
  };
  const linearChoiceArgs = {
    issuesChoice,
    featureLoading: linearFeatureLoading,
    linearAvailable,
    organizationReady,
>>>>>>> origin/main
  };
  const clearSavedJiraChoice = shouldClearSavedJiraChoice(jiraChoiceArgs);
  const clearSavedLinearChoice = shouldClearSavedLinearChoice(linearChoiceArgs);
  const jiraChoiceBlock = savedJiraChoiceBlock(jiraChoiceArgs);
  const linearChoiceBlock = savedLinearChoiceBlock(linearChoiceArgs);
  useEffect(() => {
    if (!clearSavedJiraChoice && !clearSavedLinearChoice) return;
    setIssuesChoice(null);
  }, [clearSavedJiraChoice, clearSavedLinearChoice, setIssuesChoice]);
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
    intakeState: intakeCatalog.stateOf,
    intakesLoading: jiraFeatureLoading,
    ticketIntakes: intakeCatalog.loaded ? intakeCatalog.entriesFor("onboardingTickets") : null,
    jiraChoiceBlock,
    linearAvailable,
    linearFeatureLoading,
    linearChoiceBlock,
    repositories: connection.repositories.map((repository) => repository.fullName).filter(Boolean) as string[],
    repositoryCatalog: connection.repositories as MeVcsProviderRepository[],
    repositoriesLoading: connection.onboarding.isPending,
    githubLogin: connection.identity?.login ?? "",
    githubUserId: connection.identity?.userId ?? "",
    githubIdentities: connection.identities,
    pendingOrganizations: connection.pendingOrganizations,
    synchronizing: connection.synchronizing || checkingGitHub,
    appConfigured: connection.appConfigured,
    connectError: connection.onboarding.error
      ? githubOnboardingMessage(connection.onboarding.error, "SuperPlane could not load GitHub access")
      : undefined,
    blockingAction: blocking.action,
    busy: blocking.busy || model.saving,
  };
}

export type FirstRunSetupFlow = ReturnType<typeof useFirstRunSetupFlow>;
export type { IntegrationId };
