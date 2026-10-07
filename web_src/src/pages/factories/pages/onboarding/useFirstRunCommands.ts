import type { MeVcsProviderRepository } from "@/api-client";
import { linkedAccountConnectHref } from "@/lib/accountSettings";
import { useLocation } from "react-router";

import type { useOnboardingPageModel } from "./useOnboardingPageModel";
import type { FirstRunTicketSource } from "./first-run/firstRunTypes";
import {
  canAnalyzeTicketSource,
  issuesChoiceForTicketSource,
  ticketSourceFromIssuesChoice,
} from "./first-run/firstRunTicketSource";
import type { IssuesChoiceId, VcsHostId } from "./onboardingFixtures";
import { onboardingStepPath } from "./onboardingStepPath";
import { githubConnectReturnPath } from "./onboardingGitHubConnect";
import {
  clearGitHubInstallStarted,
  githubAccessKeys,
  markGitHubInstallStarted,
  type GitHubInstallScope,
} from "./githubInstallReturn";
import { navigateGitHubWindow, openGitHubWindow, type GitHubConnectionState } from "./useFirstRunGitHub";
import type { FirstRunBlocking } from "./useFirstRunBlockingAction";
import type { OnboardingAgentGate } from "./onboardingAgentReadiness";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

function waitForBrowserPaint(): Promise<void> {
  return new Promise((resolve) => window.requestAnimationFrame(() => resolve()));
}

function selectedIssuesChoice(args: {
  model: OnboardingPageModel;
  vcsAvailable: boolean;
  jiraAvailable: boolean;
  linearAvailable: boolean;
}): IssuesChoiceId | null {
  const ticketSource = ticketSourceFromIssuesChoice(args.model.setup.issuesChoice);
  const issuesChoice = issuesChoiceForTicketSource(ticketSource);
  if (issuesChoice === "vcs" && !args.vcsAvailable) return null;
  if (issuesChoice === "jira" && !args.jiraAvailable) return null;
  if (issuesChoice === "linear" && !args.linearAvailable) return null;
  if (issuesChoice === "linear" && (args.model.linearProjectsLoading || args.model.linearProjectsError)) return null;
  if (
    !issuesChoice ||
    !canAnalyzeTicketSource({
      ticketSource,
      jiraConnected: args.model.setup.connected.has("jira"),
      jiraProjectId: args.model.jiraProjectId,
      linearConnected: args.model.setup.connected.has("linear"),
      linearProjectIds: args.model.linearProjectIds,
    })
  ) {
    return null;
  }
  return issuesChoice;
}

async function saveSelectedRepository(args: {
  model: OnboardingPageModel;
  connection: GitHubConnectionState;
}): Promise<boolean> {
  const selectedRepo = args.model.setup.selectedRepo;
  if (!selectedRepo) return false;
  if (args.model.setup.vcsHost === "bitbucket") {
    args.model.setup.commitRepoStep();
    return args.model.selectBitbucketRepository(selectedRepo);
  }
  const repository = args.connection.repositories.find((candidate) => candidate.fullName === selectedRepo);
  if (!repository) return false;
  args.model.setup.commitRepoStep();
  return args.model.selectCatalogRepository(repository);
}

export function useFirstRunCommands(args: {
  model: OnboardingPageModel;
  agentGate: OnboardingAgentGate;
  connection: GitHubConnectionState;
  navigation: { goToScreen: (screen: "welcome" | "host" | "connect" | "choose" | "tickets" | "agent") => void };
  blocking: FirstRunBlocking;
  vcsAvailable: boolean;
  jiraAvailable: boolean;
  linearAvailable: boolean;
  bitbucketAvailable: boolean;
  installScope: GitHubInstallScope;
}) {
  const location = useLocation();

  const continueFromRepository = () =>
    args.blocking.run("saving-repository", async () => {
      if (!(await saveSelectedRepository({ model: args.model, connection: args.connection }))) return;
      args.navigation.goToScreen(args.agentGate === "first" ? "agent" : "tickets");
    });

  const chooseVcsHost = (host: VcsHostId) => {
    if (host === "gitlab" || (host === "bitbucket" && !args.bitbucketAvailable)) return;
    args.model.setup.selectVcsHost(host);
    args.navigation.goToScreen(host === "bitbucket" ? "choose" : "connect");
  };

  const connectBitbucket = () => {
    void args.model.requestConnect("bitbucket");
  };

  const continueFromTickets = () =>
    args.blocking.run("saving-ticket-source", async () => {
      const issuesChoice = selectedIssuesChoice({
        model: args.model,
        vcsAvailable: args.vcsAvailable,
        jiraAvailable: args.jiraAvailable,
        linearAvailable: args.linearAvailable,
      });
      if (!issuesChoice) return;
      args.model.setup.setIssuesChoice(issuesChoice);
      args.model.setup.commitIssuesStep();
      if (!(await args.model.saveIssues(issuesChoice))) return;
      if (args.agentGate === "pending") return;
      if (args.agentGate === "show") return args.navigation.goToScreen("agent");
      args.blocking.setAction("finishing-setup");
      await args.model.finish(issuesChoice);
    });

  const connectGitHub = () =>
    args.blocking.runUntilNavigation("opening-github", async () => {
      const returnPath = githubConnectReturnPath(onboardingStepPath(`${location.pathname}${location.search}`, "repo"));
      window.location.assign(linkedAccountConnectHref("github", returnPath));
      return true;
    });

  const selectGitHubIdentity = (userId: string) =>
    args.blocking.run("switching-github-account", async () => {
      await args.connection.onboarding.selectIdentity.mutateAsync(userId);
      args.model.setup.clearRepository();
    });

  const grantGitHubAccess = () =>
    args.blocking.run("opening-github", async () => {
      markGitHubInstallStarted(args.installScope, githubAccessKeys(args.connection.onboarding.data) ?? []);
      const popup = openGitHubWindow();
      try {
        const url = await args.connection.onboarding.startInstallation.mutateAsync();
        navigateGitHubWindow(popup, url);
      } catch (error) {
        clearGitHubInstallStarted(args.installScope);
        popup?.close();
        throw error;
      }
    });

  const connectIssueTracker = (source: "jira" | "linear") =>
    args.blocking.runUntilNavigation(source === "jira" ? "connecting-jira" : "connecting-linear", async () => {
      if (source === "jira" ? !args.jiraAvailable : !args.linearAvailable) return false;
      args.model.setup.setIssuesChoice(source);
      if (!(await args.model.saveIssues(source))) return false;
      await waitForBrowserPaint();
      return args.model.requestConnect(source);
    });

  const continueFromAgent = () => {
    if (args.agentGate === "first") return args.navigation.goToScreen("tickets");
    return args.blocking.run("finishing-setup", async () => {
      await args.model.finish();
    });
  };

  const selectTicketSource = (source: FirstRunTicketSource) => {
    if (source === "jira" && !args.jiraAvailable) return;
    if (source === "linear" && !args.linearAvailable) return;
    const issuesChoice = issuesChoiceForTicketSource(source);
    if (issuesChoice) args.model.setup.setIssuesChoice(issuesChoice);
  };

  return {
    chooseVcsHost,
    connectBitbucket,
    connectGitHub,
    connectJira: () => connectIssueTracker("jira"),
    connectLinear: () => connectIssueTracker("linear"),
    continueFromRepository,
    continueFromTickets,
    continueFromAgent,
    grantGitHubAccess,
    selectGitHubIdentity,
    selectTicketSource,
    repositories: args.connection.repositories.map((repository) => repository.fullName).filter(Boolean) as string[],
    repositoryCatalog: args.connection.repositories as MeVcsProviderRepository[],
  };
}
