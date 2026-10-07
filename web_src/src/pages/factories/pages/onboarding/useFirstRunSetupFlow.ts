import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { DEFAULT_TICKET_SOURCE, ticketSourceFromIssuesChoice } from "./first-run/firstRunTicketSource";
import { type IntegrationId, type IssuesChoiceId, type VcsHostId, type WizardStepId } from "./onboardingFixtures";
import {
  onboardingAgentGate,
  type OnboardingAgentCredentialChoice,
  type OnboardingAgentGate,
} from "./onboardingAgentReadiness";
import { isWizardStepId } from "./onboardingStatus";
import { useFirstRunBlockingAction } from "./useFirstRunBlockingAction";
import { useFirstRunCommands } from "./useFirstRunCommands";
import { useFirstRunIntakeAvailability } from "./useFirstRunIntakeAvailability";
import { githubOnboardingMessage, useFirstRunGitHub, type GitHubConnectionState } from "./useFirstRunGitHub";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

export type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;
export type FirstRunScreen = "welcome" | "host" | "connect" | "choose" | "tickets" | "agent";
export { DEFAULT_TICKET_SOURCE };
const SCREEN_FOR_STEP: Record<WizardStepId, FirstRunScreen> = {
  vcs: "connect",
  repo: "choose",
  issues: "tickets",
  agent: "agent",
  name: "agent",
};
const STEP_FOR_SCREEN: Partial<Record<FirstRunScreen, WizardStepId>> = {
  host: "vcs",
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

// The host screen comes before any connection. A Bitbucket workspace does not
// need GitHub, and its repository screen connects Bitbucket itself.
function screenWithVcsConnection(
  screen: FirstRunScreen,
  vcsHost: VcsHostId | null,
  github: Parameters<typeof screenWithGitHubConnection>[1],
  bitbucket: { available: boolean; loading: boolean },
): FirstRunScreen {
  if (screen === "host") return screen;
  if (vcsHost === "bitbucket") {
    if (!bitbucket.loading && !bitbucket.available && screen !== "welcome") return "host";
    return screen === "connect" ? "choose" : screen;
  }
  return screenWithGitHubConnection(screen, github);
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
  bitbucket: { available: boolean; loading: boolean },
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

  const availableScreen = screenWithVcsConnection(
    openedScreen,
    model.setup.vcsHost,
    {
      pending: connection.onboarding.isPending,
      identity: Boolean(connection.identity),
      ready: githubReady,
      stayOnConnect,
    },
    bitbucket,
  );

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

type FlaggedIssuesChoice = "vcs" | "jira" | "linear";

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

export function shouldClearSavedVcsChoice(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  vcsAvailable: boolean;
  organizationReady: boolean;
}): boolean {
  return shouldClearSavedFlaggedChoice({
    issuesChoice: args.issuesChoice,
    source: "vcs",
    featureLoading: args.featureLoading,
    available: args.vcsAvailable,
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

/** A saved VCS choice cannot continue until the intake catalog confirms it. */
export function savedVcsChoiceBlock(args: {
  issuesChoice: IssuesChoiceId | null;
  featureLoading: boolean;
  vcsAvailable: boolean;
  organizationReady: boolean;
}): SavedFlaggedChoiceBlock | null {
  return savedFlaggedChoiceBlock({
    issuesChoice: args.issuesChoice,
    source: "vcs",
    featureLoading: args.featureLoading,
    available: args.vcsAvailable,
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
  const { connection, githubReady, installScope, checkingGitHub } = useFirstRunGitHub({
    organizationId,
    factoryId,
    setupFinished,
    connectedBefore: Boolean(model.setup.selectedRepo),
  });
  const intake = useFirstRunIntakeAvailability(organizationId);
  const agentGate = onboardingAgentGate({
    hostedModelsAvailable: model.hostedModelsAvailable,
    hostedModelsAvailableLoading: model.hostedModelsAvailableLoading,
    bringYourOwnKey: model.bringYourOwnKey,
    bringYourOwnKeyLoading: model.bringYourOwnKeyLoading,
  });
  const navigation = useFirstRunNavigation(model, agentGate, connection, githubReady, intake.bitbucket);
  const commands = useFirstRunCommands({
    model,
    agentGate,
    connection,
    navigation,
    blocking,
    vcsAvailable: intake.vcsAvailable,
    jiraAvailable: intake.jiraAvailable,
    linearAvailable: intake.linearAvailable,
    bitbucketAvailable: intake.bitbucketAvailable,
    installScope,
  });
  // A saved Jira or Linear choice is not valid when the intake catalog does
  // not let the organization use that intake. Clear it only after the catalog
  // loads and confirms that. A failed lookup has no catalog data and must not
  // replace the saved choice with the GitHub Issues default.
  const issuesChoice = model.setup.issuesChoice;
  const setIssuesChoice = model.setup.setIssuesChoice;
  const organizationReady = intake.organizationReady;
  const vcsChoiceArgs = {
    issuesChoice,
    featureLoading: intake.intakesLoading,
    vcsAvailable: intake.vcsAvailable,
    organizationReady,
  };
  const jiraChoiceArgs = {
    issuesChoice,
    featureLoading: intake.intakesLoading,
    jiraAvailable: intake.jiraAvailable,
    organizationReady,
  };
  const linearChoiceArgs = {
    issuesChoice,
    featureLoading: intake.intakesLoading,
    linearAvailable: intake.linearAvailable,
    organizationReady,
  };
  const clearSavedVcsChoice = shouldClearSavedVcsChoice(vcsChoiceArgs);
  const clearSavedJiraChoice = shouldClearSavedJiraChoice(jiraChoiceArgs);
  const clearSavedLinearChoice = shouldClearSavedLinearChoice(linearChoiceArgs);
  const vcsChoiceBlock = savedVcsChoiceBlock(vcsChoiceArgs);
  const jiraChoiceBlock = savedJiraChoiceBlock(jiraChoiceArgs);
  const linearChoiceBlock = savedLinearChoiceBlock(linearChoiceArgs);
  useEffect(() => {
    if (!clearSavedVcsChoice && !clearSavedJiraChoice && !clearSavedLinearChoice) return;
    setIssuesChoice(null);
  }, [clearSavedVcsChoice, clearSavedJiraChoice, clearSavedLinearChoice, setIssuesChoice]);
  return {
    ...navigation,
    ...commands,
    vcsHost: model.setup.vcsHost,
    bitbucketAvailable: intake.bitbucketAvailable,
    bitbucketFeatureLoading: intake.bitbucketFeatureLoading,
    // The ticket screen is the last screen, so it finishes setup.
    ticketsFinishSetup: agentGate === "skip" || agentGate === "first",
    skipAgentScreen: agentGate === "skip",
    agentBeforeTickets: agentGate === "first",
    credentialChoice: model.agentCredentialChoice,
    selectCredentialChoice: model.setAgentCredentialChoice,
    agentGatePending: agentGate === "pending",
    ticketSource: ticketSourceFromIssuesChoice(model.setup.issuesChoice),
    jiraAvailable: intake.jiraAvailable,
    intakeState: intake.intakeState,
    intakesLoading: intake.intakesLoading,
    ticketIntakes: intake.ticketIntakes,
    vcsChoiceBlock,
    jiraFeatureLoading: intake.jiraFeatureLoading,
    jiraChoiceBlock,
    linearAvailable: intake.linearAvailable,
    linearFeatureLoading: intake.linearFeatureLoading,
    linearChoiceBlock,
    repositories: commands.repositories,
    repositoryCatalog: commands.repositoryCatalog,
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
