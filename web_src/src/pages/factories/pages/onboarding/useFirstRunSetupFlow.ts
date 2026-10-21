import type { MeVcsProviderRepository } from "@/api-client";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { startPublicGitHubAppCreate } from "@/lib/githubAppManifest";
import { useEffect, useRef, useState } from "react";
import { useLocation, useSearchParams } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import type { FirstRunTicketSource } from "./first-run/firstRunTypes";
import {
  canAnalyzeTicketSource,
  issuesChoiceForTicketSource,
  resolveTicketSelection,
} from "./first-run/firstRunTicketSource";
import {
  type FirstRunHostChoice,
  type IntegrationId,
  type IssuesChoiceId,
  type WizardStepId,
} from "./onboardingFixtures";
import { readOnboardingGithubEnterprise, writeOnboardingGithubEnterprise } from "./onboardingGithubEnterprise";
import {
  SCREEN_FOR_STEP,
  STEP_FOR_SCREEN,
  screenWithModelSource,
  screenWithoutAgent,
  screenWithoutIncompleteBacklog,
  screenWithVcsConnection,
  type FirstRunScreen,
} from "./firstRunScreens";
import { useFirstRunGitHubEnterprise } from "./useFirstRunGitHubEnterprise";
import { onboardingAgentGate, type OnboardingAgentGate } from "./onboardingAgentReadiness";
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
export type { FirstRunScreen };

/** Null means no step was requested, so the flow resumes from saved progress. */
function initialFirstRunScreen(searchParams: URLSearchParams): FirstRunScreen | null {
  const requestedStep = searchParams.get("step");
  if (isWizardStepId(requestedStep)) return SCREEN_FOR_STEP[requestedStep];
  return null;
}

// A GitHub return can omit the step. Saved progress or that return must not
// open the welcome screen. An installation GitHub App that does not require
// sign-in is that return. A connection saved for the account does not choose
// the opening screen.
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

function useFirstRunNavigation(
  model: OnboardingPageModel,
  agentGate: OnboardingAgentGate,
  connection: GitHubConnectionState,
  github: { ready: boolean; returned: boolean; enterpriseReady: boolean },
  githubEnterprise: boolean,
) {
  const [searchParams] = useSearchParams();
  const [requestedScreen, setRequestedScreen] = useState<FirstRunScreen | null>(() =>
    initialFirstRunScreen(searchParams),
  );
  // Back from repository selection must not forward to it again.
  const [stayOnConnect, setStayOnConnect] = useState(false);
  const openStep = useRef(model.openSection);
  const installationAppReady = connection.appConfigured && !connection.accountConnectionRequired;
  const openedScreen =
    requestedScreen ??
    resumeFirstRunScreen({
      savedStep: model.openSection,
      githubPending: connection.onboarding.isPending,
      githubReturned: github.returned,
      installationAppReady,
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
      pending: githubEnterprise ? false : connection.onboarding.isPending,
      identity: Boolean(connection.identity),
      ready: githubEnterprise ? github.enterpriseReady : github.ready,
      stayOnConnect,
      appConfigured: githubEnterprise ? github.enterpriseReady : connection.appConfigured,
      accountConnectionRequired: githubEnterprise ? false : connection.accountConnectionRequired,
    },
    githubEnterprise,
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

function saveSelectedRepository(args: {
  model: OnboardingPageModel;
  connection: GitHubConnectionState;
  forgeConfigured: boolean;
  githubEnterprise: boolean;
}): Promise<boolean> {
  const selectedRepo = args.model.setup.selectedRepo;
  if (!selectedRepo) return Promise.resolve(false);
  if (args.model.setup.vcsHost === "bitbucket") {
    args.model.setup.commitRepoStep();
    if (args.forgeConfigured) return args.model.selectBitbucketForgeRepository(selectedRepo);
    return args.model.selectBitbucketRepository(selectedRepo);
  }
  if (args.githubEnterprise) {
    args.model.setup.commitRepoStep();
    return args.model.selectGitHubEnterpriseRepository(selectedRepo);
  }
  const repository = args.connection.repositories.find((candidate) => candidate.fullName === selectedRepo);
  if (!repository) return Promise.resolve(false);
  args.model.setup.commitRepoStep();
  return args.model.selectCatalogRepository(repository);
}

function continueFromTicketSource(args: {
  model: OnboardingPageModel;
  agentGate: OnboardingAgentGate;
  blocking: FirstRunBlocking;
  navigation: ReturnType<typeof useFirstRunNavigation>;
}) {
  return args.blocking.run("saving-ticket-source", async () => {
    const issuesChoice = selectedIssuesChoice(args.model);
    if (!issuesChoice) return;
    args.model.setup.setIssuesChoice(issuesChoice);
    args.model.setup.commitIssuesStep();
    if (!(await args.model.saveIssues(issuesChoice))) return;
    if (args.agentGate === "pending") return;
    if (args.agentGate === "show") return args.navigation.goToScreen("agent");
    args.blocking.setAction("finishing-setup");
    await args.model.finish(issuesChoice);
  });
}

function useFirstRunCommands(args: {
  model: OnboardingPageModel;
  agentGate: OnboardingAgentGate;
  connection: GitHubConnectionState;
  navigation: ReturnType<typeof useFirstRunNavigation>;
  blocking: FirstRunBlocking;
  forgeConfigured: boolean;
  installScope: GitHubInstallScope;
  factoryId: string;
  githubEnterprise: boolean;
  setGitHubEnterprise: (enabled: boolean, host?: string) => void;
  connectGitHubEnterpriseApp: (host: string, returnTo: string) => Promise<boolean>;
}) {
  const { model, agentGate, connection, navigation, blocking, forgeConfigured, installScope } = args;
  const location = useLocation();
  const saveRepository = () =>
    saveSelectedRepository({ model, connection, forgeConfigured, githubEnterprise: args.githubEnterprise });
  const continueFromRepository = () =>
    blocking.run("saving-repository", async () => {
      if (await saveRepository()) {
        navigation.goToScreen(agentGate === "first" ? "agent" : "tickets");
      }
    });
  const chooseVcsHost = (host: FirstRunHostChoice) => {
    if (host === "gitlab") return;
    if (host === "github-enterprise") {
      args.setGitHubEnterprise(true);
      model.setup.selectVcsHost("github");
      navigation.goToScreen("ghe-host");
      return;
    }
    args.setGitHubEnterprise(false);
    model.setup.selectVcsHost(host);
    navigation.goToScreen(host === "bitbucket" ? "choose" : "connect");
  };
  const connectGitHubEnterprise = (host: string) =>
    blocking.runUntilNavigation("opening-github", async () => {
      args.setGitHubEnterprise(true, host);
      const returnPath = onboardingStepPath(`${location.pathname}${location.search}`, "repo");
      return args.connectGitHubEnterpriseApp(host, returnPath);
    });
  const connectBitbucket = () => {
    void model.requestConnect("bitbucket");
  };
  const continueFromTickets = () => continueFromTicketSource({ model, agentGate, blocking, navigation });
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
  const continueFromAgent = () => {
    if (agentGate === "first") {
      return blocking.run("saving-agent", async () => {
        if (await model.saveAgent()) navigation.goToScreen("tickets");
      });
    }
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
    connectGitHubEnterprise,
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
  const rememberedEnterprise = readOnboardingGithubEnterprise(factoryId);
  const [githubEnterprise, setGitHubEnterpriseEnabled] = useState(rememberedEnterprise.enabled);
  const [githubEnterpriseHost, setGitHubEnterpriseHost] = useState(rememberedEnterprise.host);
  const setGitHubEnterprise = (enabled: boolean, host = "") => {
    writeOnboardingGithubEnterprise(factoryId, enabled, host);
    setGitHubEnterpriseEnabled(enabled);
    if (!enabled) {
      setGitHubEnterpriseHost("");
      return;
    }
    if (host) setGitHubEnterpriseHost(host);
  };
  const setupFinished = model.provisionedDestination != null;
  const blocking = useFirstRunBlockingAction();
  const githubEnterpriseApp = useFirstRunGitHubEnterprise({
    organizationId,
    factoryId,
    onCreated: model.bindGitHubEnterpriseIntegration,
  });
  const { connection, githubReady, githubReturned, installScope, checkingGitHub } = useFirstRunGitHub({
    organizationId: githubEnterprise ? "" : organizationId,
    factoryId,
    setupFinished,
    connectedBefore: Boolean(model.setup.selectedRepo),
  });
  const bitbucketConnect = useFirstRunBitbucket({ organizationId, setupFinished, blocking });
  const agentGate = onboardingAgentGate({
    hostedModelsAvailable: model.hostedModelsAvailable,
    hostedModelsAvailableLoading: model.hostedModelsAvailableLoading,
    bringYourOwnKey: model.bringYourOwnKey,
    bringYourOwnKeyLoading: model.bringYourOwnKeyLoading,
  });
  const navigation = useFirstRunNavigation(
    model,
    agentGate,
    connection,
    {
      ready: githubReady,
      returned: githubReturned,
      enterpriseReady: Boolean(rememberedEnterprise.integrationId),
    },
    githubEnterprise,
  );
  const commands = useFirstRunCommands({
    model,
    agentGate,
    connection,
    navigation,
    blocking,
    forgeConfigured: bitbucketConnect.configured,
    installScope,
    factoryId,
    githubEnterprise,
    setGitHubEnterprise,
    connectGitHubEnterpriseApp: githubEnterpriseApp.connect,
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
    selectBitbucketIdentity,
    vcsHost: model.setup.vcsHost,
    githubEnterprise,
    githubEnterpriseHost,
    githubEnterpriseIntegrationId: rememberedEnterprise.integrationId,
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
    // The ticket screen is the last screen, so it finishes setup.
    ticketsFinishSetup: agentGate === "skip" || agentGate === "first",
    skipAgentScreen: agentGate === "skip",
    agentBeforeTickets: agentGate === "first",
    credentialChoice: model.agentCredentialChoice,
    selectCredentialChoice: model.setAgentCredentialChoice,
    agentGatePending: agentGate === "pending",
    ticketSource: ticketSelection(model).ticketSource,
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
    blockingAction: blocking.action,
    busy: blocking.busy || model.saving,
  };
}

export type FirstRunSetupFlow = ReturnType<typeof useFirstRunSetupFlow>;
export type { IntegrationId };
