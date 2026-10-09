import type { MeVcsProviderRepository } from "@/api-client";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { FEATURE_FACTORY_BITBUCKET } from "@/lib/experimentalFeatures";
import {
  fetchGitHubLoginClient,
  saveGitHubLoginClient,
  startPublicGitHubAppCreate,
  type GitHubLoginClient,
} from "@/lib/githubAppManifest";
import { useEffect, useRef, useState } from "react";
import { useLocation, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import type { FirstRunTicketSource } from "./first-run/firstRunTypes";
import {
  canAnalyzeTicketSource,
  DEFAULT_TICKET_SOURCE,
  issuesChoiceForTicketSource,
  ticketSourceFromIssuesChoice,
} from "./first-run/firstRunTicketSource";
import { type IntegrationId, type IssuesChoiceId, type VcsHostId, type WizardStepId } from "./onboardingFixtures";
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
import { useFirstRunBitbucket } from "./useFirstRunBitbucket";
import {
  githubOnboardingMessage,
  navigateGitHubWindow,
  openGitHubWindow,
  useFirstRunGitHub,
  type GitHubConnectionState,
} from "./useFirstRunGitHub";
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

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
}

function selectedIssuesChoice(model: OnboardingPageModel): IssuesChoiceId | null {
  const ticketSource = ticketSourceFromIssuesChoice(model.setup.issuesChoice);
  const issuesChoice = issuesChoiceForTicketSource(ticketSource);
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

function saveSelectedRepository(args: {
  model: OnboardingPageModel;
  connection: GitHubConnectionState;
  forgeConfigured: boolean;
}): Promise<boolean> {
  const selectedRepo = args.model.setup.selectedRepo;
  if (!selectedRepo) return Promise.resolve(false);
  if (args.model.setup.vcsHost === "bitbucket") {
    args.model.setup.commitRepoStep();
    if (args.forgeConfigured) return args.model.selectBitbucketForgeRepository(selectedRepo);
    return args.model.selectBitbucketRepository(selectedRepo);
  }
  const repository = args.connection.repositories.find((candidate) => candidate.fullName === selectedRepo);
  if (!repository) return Promise.resolve(false);
  args.model.setup.commitRepoStep();
  return args.model.selectCatalogRepository(repository);
}

function useFirstRunCommands(args: {
  model: OnboardingPageModel;
  agentGate: OnboardingAgentGate;
  connection: GitHubConnectionState;
  navigation: ReturnType<typeof useFirstRunNavigation>;
  blocking: FirstRunBlocking;
  bitbucketAvailable: boolean;
  forgeConfigured: boolean;
  installScope: GitHubInstallScope;
  setLoginError: (message: string | undefined) => void;
}) {
  const {
    model,
    agentGate,
    connection,
    navigation,
    blocking,
    bitbucketAvailable,
    forgeConfigured,
    installScope,
    setLoginError,
  } = args;
  const location = useLocation();
  const saveRepository = () => saveSelectedRepository({ model, connection, forgeConfigured });
  const continueFromRepository = () =>
    blocking.run("saving-repository", async () => {
      if (await saveRepository()) {
        navigation.goToScreen(agentGate === "first" ? "agent" : "tickets");
      }
    });
  const chooseVcsHost = (host: VcsHostId) => {
    if (host === "gitlab" || (host === "bitbucket" && !bitbucketAvailable)) return;
    model.setup.selectVcsHost(host);
    navigation.goToScreen(host === "bitbucket" ? "choose" : "connect");
  };
  const connectBitbucket = () => {
    void model.requestConnect("bitbucket");
  };
  const continueFromTickets = () =>
    blocking.run("saving-ticket-source", async () => {
      const issuesChoice = selectedIssuesChoice(model);
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
  const createGitHubApp = () =>
    blocking.runUntilNavigation("opening-github", async () => {
      return startPublicGitHubAppCreate(onboardingStepPath(`${location.pathname}${location.search}`, "vcs"));
    });
  const saveGitHubLogin = (clientId: string, clientSecret: string) =>
    blocking.runUntilNavigation("opening-github", async () => {
      setLoginError(undefined);
      try {
        await saveGitHubLoginClient(clientId, clientSecret);
      } catch {
        setLoginError(FIRST_RUN_COPY.connect.loginSaveError);
        return false;
      }
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
  const connectIssueTracker = (source: "jira" | "linear") =>
    blocking.runUntilNavigation(source === "jira" ? "connecting-jira" : "connecting-linear", async () => {
      model.setup.setIssuesChoice(source);
      if (!(await model.saveIssues(source))) return false;
      await waitForBrowserPaint();
      return model.requestConnect(source);
    });
  const continueFromAgent = () => {
    if (agentGate === "first") return navigation.goToScreen("tickets");
    return blocking.run("finishing-setup", async () => {
      await model.finish();
    });
  };
  const selectTicketSource = (source: FirstRunTicketSource) => {
    const issuesChoice = issuesChoiceForTicketSource(source);
    if (issuesChoice) model.setup.setIssuesChoice(issuesChoice);
  };
  return {
    chooseVcsHost,
    connectBitbucket,
    connectGitHub,
    createGitHubApp,
    saveGitHubLogin,
    connectJira: () => connectIssueTracker("jira"),
    connectLinear: () => connectIssueTracker("linear"),
    continueFromRepository,
    continueFromTickets,
    continueFromAgent,
    grantGitHubAccess,
    selectGitHubIdentity,
    selectTicketSource,
  };
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
  const bitbucketConnect = useFirstRunBitbucket({ organizationId, setupFinished, blocking });
  const intakeFeatures = useExperimentalFeature(organizationId);
  const intakeFeatureLoading = intakeFeatures.isLoading;
  const bitbucketAvailable = !intakeFeatureLoading && intakeFeatures.has(FEATURE_FACTORY_BITBUCKET);
  const bitbucket = { available: bitbucketAvailable, loading: intakeFeatureLoading };
  const agentGate = onboardingAgentGate({
    hostedModelsAvailable: model.hostedModelsAvailable,
    hostedModelsAvailableLoading: model.hostedModelsAvailableLoading,
    bringYourOwnKey: model.bringYourOwnKey,
    bringYourOwnKeyLoading: model.bringYourOwnKeyLoading,
  });
  const navigation = useFirstRunNavigation(model, agentGate, connection, githubReady, bitbucket);
  const [loginError, setLoginError] = useState<string>();
  const [editingLogin, setEditingLogin] = useState(false);
  const [loginClient, setLoginClient] = useState<GitHubLoginClient>({ state: "missing", canUpdate: false });
  const commands = useFirstRunCommands({
    model,
    agentGate,
    connection,
    navigation,
    blocking,
    bitbucketAvailable,
    forgeConfigured: bitbucketConnect.configured,
    installScope,
    setLoginError,
  });
  useEffect(() => {
    let cancelled = false;
    void fetchGitHubLoginClient()
      .then((status) => {
        if (!cancelled) setLoginClient(status);
      })
      .catch(() => {
        if (!cancelled) setLoginClient({ state: "missing", canUpdate: false });
      });
    return () => {
      cancelled = true;
    };
  }, [connection.appConfigured]);
  const loginFormOpen = (loginClient.state === "needs_client" && !connection.appConfigured) || editingLogin;
  return {
    ...navigation,
    ...commands,
    vcsHost: model.setup.vcsHost,
    bitbucketAvailable,
    bitbucketFeatureLoading: intakeFeatureLoading,
    bitbucketForgeConfigured: bitbucketConnect.configured,
    bitbucketOnboardingPending: bitbucketConnect.pending,
    bitbucketRepositories: bitbucketConnect.repositories,
    bitbucketInstalledWorkspaces: bitbucketConnect.installedWorkspaces,
    bitbucketInstallationAttemptActive: bitbucketConnect.installationAttemptActive,
    bitbucketInstallationAttemptTimedOut: bitbucketConnect.installationAttemptTimedOut,
    bitbucketIdentityLinked: bitbucketConnect.identityLinked,
    bitbucketLoadError: bitbucketConnect.loadError,
    bitbucketLookupFailed: bitbucketConnect.lookupFailed,
    bitbucketLookupRetrying: bitbucketConnect.lookupRetrying,
    retryBitbucketLookup: bitbucketConnect.retryLookup,
    bitbucketConnectHref: bitbucketConnect.connectHref,
    grantBitbucketAccess: bitbucketConnect.grantAccess,
    // The ticket screen is the last screen, so it finishes setup.
    ticketsFinishSetup: agentGate === "skip" || agentGate === "first",
    skipAgentScreen: agentGate === "skip",
    agentBeforeTickets: agentGate === "first",
    credentialChoice: model.agentCredentialChoice,
    selectCredentialChoice: model.setAgentCredentialChoice,
    agentGatePending: agentGate === "pending",
    ticketSource: ticketSourceFromIssuesChoice(model.setup.issuesChoice),
    repositories: connection.repositories.map((repository) => repository.fullName).filter(Boolean) as string[],
    repositoryCatalog: connection.repositories as MeVcsProviderRepository[],
    repositoriesLoading: connection.onboarding.isPending,
    githubLogin: connection.identity?.login ?? "",
    githubUserId: connection.identity?.userId ?? "",
    githubIdentities: connection.identities,
    pendingOrganizations: connection.pendingOrganizations,
    synchronizing: connection.synchronizing || checkingGitHub,
    appConfigured: connection.appConfigured,
    loginClientNeeded: loginFormOpen,
    canChangeLogin: loginClient.canUpdate && !loginFormOpen,
    startGitHubLoginEdit: () => {
      setLoginError(undefined);
      setEditingLogin(true);
    },
    loginError,
    connectError: connection.onboarding.error
      ? githubOnboardingMessage(connection.onboarding.error, "SuperPlane could not load GitHub access")
      : undefined,
    blockingAction: blocking.action,
    busy: blocking.busy || model.saving,
  };
}

export type FirstRunSetupFlow = ReturnType<typeof useFirstRunSetupFlow>;
export type { IntegrationId };
