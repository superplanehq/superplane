import { useOnboardingModelTransition } from "./useOnboardingModelTransition";
import type { MeVcsProviderRepository } from "@/api-client";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { startPublicGitHubAppCreate } from "@/lib/githubAppManifest";
import { useEffect, useState } from "react";
import { useLocation, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import type { FirstRunTicketSource } from "./first-run/firstRunTypes";
import {
  canAnalyzeTicketSource,
  issuesChoiceForTicketSource,
  resolveTicketSelection,
} from "./first-run/firstRunTicketSource";
import { type IntegrationId, type IssuesChoiceId, type VcsHostId, type WizardStepId } from "./onboardingFixtures";
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
export type FirstRunScreen = "welcome" | "host" | "connect" | "choose" | "tickets" | "card" | "agent" | "complete";
// Screens with no wizard step of their own use their name in the URL.
const URL_ONLY_SCREENS: ReadonlySet<string> = new Set<FirstRunScreen>(["welcome", "host", "card", "complete"]);
// Screens after the backlog step need a repository and a complete intake choice.
const FINAL_SCREENS: ReadonlySet<FirstRunScreen> = new Set<FirstRunScreen>(["card", "agent", "complete"]);
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
  if (requestedStep === "billing") return "complete";
  if (requestedStep && URL_ONLY_SCREENS.has(requestedStep)) return requestedStep as FirstRunScreen;
  if (isWizardStepId(requestedStep)) return SCREEN_FOR_STEP[requestedStep];
  return null;
}

function urlStepFor(screen: FirstRunScreen): string {
  return URL_ONLY_SCREENS.has(screen) ? screen : (STEP_FOR_SCREEN[screen] ?? "vcs");
}

function resumeFirstRunScreen(args: {
  savedStep: WizardStepId;
  githubPending: boolean;
  githubReturned: boolean;
  installationAppReady: boolean;
}): FirstRunScreen {
  if (args.savedStep !== "vcs") return SCREEN_FOR_STEP[args.savedStep];
  if (args.githubPending) return "connect";
  if (args.githubReturned || args.installationAppReady) return "choose";
  return "welcome";
}

// A linked GitHub identity opens repository selection. Sign-in stores that
// identity, so a later workspace does not ask the person to connect again.
function screenWithGitHubConnection(
  screen: FirstRunScreen,
  github: {
    pending: boolean;
    identity: boolean;
    ready: boolean;
    stayOnConnect: boolean;
    appConfigured: boolean;
    accountConnectionRequired: boolean;
  },
): FirstRunScreen {
  if (github.pending) return screen;
  if (!github.accountConnectionRequired) {
    if (!github.appConfigured) return screen === "choose" ? "connect" : screen;
    if (screen === "connect" && !github.stayOnConnect) return "choose";
    return screen;
  }
  if (screen === "welcome") return screen;
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
): FirstRunScreen {
  if (screen !== "connect" && screen !== "choose") return screen;
  if (vcsHost === "bitbucket") {
    return screen === "connect" ? "choose" : screen;
  }
  return screenWithGitHubConnection(screen, github);
}

function useFirstRunNavigation(
  model: OnboardingPageModel,
  connection: GitHubConnectionState,
  github: { ready: boolean; returned: boolean },
) {
  const [searchParams, setSearchParams] = useSearchParams();
  const { factory } = useFactoriesLayout();
  // Back from repository selection must not forward to it again.
  const [stayOnConnect, setStayOnConnect] = useState(false);
  const openedScreen =
    initialFirstRunScreen(searchParams) ??
    resumeFirstRunScreen({
      savedStep: model.openSection,
      githubPending: connection.onboarding.isPending,
      githubReturned: github.returned,
      installationAppReady: connection.appConfigured && !connection.accountConnectionRequired,
    });

  const availableScreen = screenWithVcsConnection(openedScreen, model.setup.vcsHost, {
    pending: connection.onboarding.isPending,
    identity: Boolean(connection.identity),
    ready: github.ready,
    stayOnConnect,
    appConfigured: connection.appConfigured,
    accountConnectionRequired: connection.accountConnectionRequired,
  });

  const goToScreen = (next: FirstRunScreen) => {
    const updated = new URLSearchParams(searchParams);
    updated.set("step", urlStepFor(next));
    setSearchParams(updated);
    const step = STEP_FOR_SCREEN[next];
    if (step) {
      model.setOpenSection(step);
    }
    if (next === "connect") setStayOnConnect(availableScreen === "choose");
  };

  useEffect(() => {
    if (openedScreen !== "connect") setStayOnConnect(false);
  }, [openedScreen]);

  // Final setup steps require a repository and a complete intake choice.
  let screen = availableScreen;
  const savedVcsRestoring =
    Boolean(factory?.onboarding?.vcsIntegrationId) && !model.setup.connected.has(model.setup.vcsHost ?? "github");
  const intakePending = intakeLookupPending(model) || savedVcsRestoring;
  const githubPending = connection.onboarding.isPending;
  if (FINAL_SCREENS.has(screen) && !model.setup.selectedRepo) screen = "host";
  if (FINAL_SCREENS.has(screen) && !intakePrerequisitesSatisfied(model)) screen = "tickets";
  useEffect(() => {
    if (githubPending && !initialFirstRunScreen(searchParams)) return;
    const step = urlStepFor(screen);
    if (searchParams.get("step") === step || (screen === "tickets" && intakePending)) return;
    const updated = new URLSearchParams(searchParams);
    updated.set("step", step);
    if (searchParams.get("step") === "billing") {
      updated.delete("subscribed");
      updated.delete("checkout_id");
      updated.delete("plan");
    }
    setSearchParams(updated, { replace: true });
  }, [screen, intakePending, githubPending, searchParams, setSearchParams]);
  return { screen, goToScreen };
}

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
}

function ticketSelection(model: OnboardingPageModel) {
  return resolveTicketSelection({
    issuesChoice: model.setup.issuesChoice,
    vcsHost: model.setup.vcsHost,
    githubConnected: model.setup.connected.has("github"),
  });
}

function selectedIssuesChoice(model: OnboardingPageModel): IssuesChoiceId | null {
  const { ticketSource, issuesChoice } = ticketSelection(model);
  if (issuesChoice === "linear" && (model.linearProjectsLoading || model.linearProjectsError)) return null;
  if (issuesChoice === "jira" && (model.jiraProjectsLoading || model.jiraProjectsError)) return null;
  if (
    !issuesChoice ||
    !canAnalyzeTicketSource({
      ticketSource,
      githubConnected: model.setup.connected.has("github"),
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

function intakeLookupPending(model: OnboardingPageModel) {
  if (model.setup.issuesChoice === "jira") return model.jiraProjectsLoading;
  if (model.setup.issuesChoice === "linear") return model.linearProjectsLoading;
  return false;
}

function intakePrerequisitesSatisfied(model: OnboardingPageModel) {
  if (model.setup.issuesChoice === "skip") return true;
  return selectedIssuesChoice(model) != null;
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

function continueFromTicketSource(args: {
  model: OnboardingPageModel;
  afterTickets: () => Promise<void>;
  blocking: FirstRunBlocking;
}) {
  return args.blocking.run("saving-ticket-source", async () => {
    const issuesChoice = selectedIssuesChoice(args.model);
    if (!issuesChoice) return;
    args.model.setup.setIssuesChoice(issuesChoice);
    args.model.setup.commitIssuesStep();
    if (!(await args.model.saveIssues(issuesChoice))) return;
    await args.afterTickets();
  });
}

function useFirstRunCommands(args: {
  model: OnboardingPageModel;
  afterTickets: () => Promise<void>;
  connection: GitHubConnectionState;
  navigation: ReturnType<typeof useFirstRunNavigation>;
  blocking: FirstRunBlocking;
  forgeConfigured: boolean;
  installScope: GitHubInstallScope;
}) {
  const { model, connection, navigation, blocking, forgeConfigured, installScope } = args;
  const location = useLocation();
  const saveRepository = () => saveSelectedRepository({ model, connection, forgeConfigured });
  const continueFromRepository = () =>
    blocking.run("saving-repository", async () => {
      if (await saveRepository()) {
        navigation.goToScreen("tickets");
      }
    });
  const chooseVcsHost = (host: VcsHostId) => {
    if (host === "gitlab") return;
    model.setup.selectVcsHost(host);
    navigation.goToScreen(host === "bitbucket" ? "choose" : "connect");
  };
  const connectBitbucket = () => {
    void model.requestConnect("bitbucket");
  };
  const continueFromTickets = () => continueFromTicketSource({ model, afterTickets: args.afterTickets, blocking });
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
  const connectGitHubIssues = () =>
    blocking.runUntilNavigation("opening-github", async () => {
      model.setup.setIssuesChoice("vcs");
      if (!(await model.saveIssues("vcs"))) return false;
      await waitForBrowserPaint();
      return model.requestConnect("github");
    });
  const connectIssueTracker = (source: "jira" | "linear") =>
    blocking.runUntilNavigation(source === "jira" ? "connecting-jira" : "connecting-linear", async () => {
      model.setup.setIssuesChoice(source);
      if (!(await model.saveIssues(source))) return false;
      await waitForBrowserPaint();
      return model.requestConnect(source);
    });
  const continueFromAgent = () =>
    blocking.run("finishing-setup", async () => {
      await model.finish();
    });
  const selectTicketSource = (source: FirstRunTicketSource) => {
    const issuesChoice = issuesChoiceForTicketSource(source);
    if (issuesChoice) model.setup.setIssuesChoice(issuesChoice);
  };
  return {
    chooseVcsHost,
    connectBitbucket,
    connectGitHub,
    connectGitHubIssues,
    createGitHubApp,
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
  const { connection, githubReady, githubReturned, installScope, checkingGitHub } = useFirstRunGitHub({
    organizationId,
    factoryId,
    setupFinished,
    connectedBefore: Boolean(model.setup.selectedRepo),
  });
  const bitbucketConnect = useFirstRunBitbucket({ organizationId, setupFinished, blocking });
  const navigation = useFirstRunNavigation(model, connection, {
    ready: githubReady,
    returned: githubReturned,
  });
  const modelTransition = useOnboardingModelTransition(model, navigation, organizationId);
  const commands = useFirstRunCommands({
    model,
    afterTickets: modelTransition.afterTickets,
    connection,
    navigation,
    blocking,
    forgeConfigured: bitbucketConnect.configured,
    installScope,
  });
  const [bitbucketSwitchFailed, setBitbucketSwitchFailed] = useState(false);
  const selectBitbucketIdentity = (providerUserId: string) =>
    blocking.run("switching-bitbucket-account", async () => {
      setBitbucketSwitchFailed(false);
      try {
        await bitbucketConnect.selectIdentity(providerUserId);
        model.setup.clearRepository();
      } catch {
        setBitbucketSwitchFailed(true);
      }
    });
  return {
    ...navigation,
    ...commands,
    ...modelTransition,
    finishSetup: () => blocking.run("finishing-setup", modelTransition.continueAfterModelAccess),
    selectBitbucketIdentity,
    vcsHost: model.setup.vcsHost,
    bitbucketForgeConfigured: bitbucketConnect.configured,
    bitbucketOnboardingPending: bitbucketConnect.pending,
    bitbucketRepositories: bitbucketConnect.repositories,
    bitbucketInstalledWorkspaces: bitbucketConnect.installedWorkspaces,
    bitbucketInstallationAttemptActive: bitbucketConnect.installationAttemptActive,
    bitbucketInstallationAttemptTimedOut: bitbucketConnect.installationAttemptTimedOut,
    bitbucketIdentityLinked: bitbucketConnect.identityLinked,
    bitbucketLogin: bitbucketConnect.login,
    bitbucketProviderUserId: bitbucketConnect.providerUserId,
    bitbucketIdentities: bitbucketConnect.identities,
    bitbucketLoadError: bitbucketConnect.loadError,
    bitbucketSwitchFailed,
    bitbucketLookupFailed: bitbucketConnect.lookupFailed,
    bitbucketLookupRetrying: bitbucketConnect.lookupRetrying,
    retryBitbucketLookup: bitbucketConnect.retryLookup,
    bitbucketConnectHref: bitbucketConnect.connectHref,
    bitbucketInstallUrl: bitbucketConnect.installUrl,
    copyBitbucketInstallLink: bitbucketConnect.copyInstallLink,
    grantBitbucketAccess: bitbucketConnect.grantAccess,
    skipAgentScreen: modelTransition.skipAgentScreen,
    credentialChoice: model.agentCredentialChoice,
    selectCredentialChoice: model.setAgentCredentialChoice,
    ticketSource: ticketSelection(model).ticketSource,
    ...firstRunGitHubProperties(connection, checkingGitHub),
    blockingAction: blocking.action,
    busy: blocking.busy || model.saving || modelTransition.pending,
  };
}

export type FirstRunSetupFlow = ReturnType<typeof useFirstRunSetupFlow>;
export type { IntegrationId };

function firstRunGitHubProperties(connection: GitHubConnectionState, checkingGitHub: boolean) {
  return {
    repositories: connection.repositories.map((repository) => repository.fullName).filter(Boolean) as string[],
    repositoryCatalog: connection.repositories as MeVcsProviderRepository[],
    repositoriesLoading: connection.onboarding.isPending,
    githubLogin: connection.identity?.login ?? "",
    githubUserId: connection.identity?.userId ?? "",
    githubIdentities: connection.identities,
    pendingOrganizations: connection.pendingOrganizations,
    synchronizing: connection.synchronizing || checkingGitHub,
    appConfigured: connection.appConfigured,
    accountConnectionRequired: connection.accountConnectionRequired,
    connectError: connection.onboarding.error
      ? githubOnboardingMessage(connection.onboarding.error, "SuperPlane could not load GitHub access")
      : undefined,
  };
}
