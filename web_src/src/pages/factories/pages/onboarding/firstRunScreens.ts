import type { OnboardingAgentCredentialChoice, OnboardingAgentGate } from "./onboardingAgentReadiness";
import type { IssuesChoiceId, VcsHostId, WizardStepId } from "./onboardingFixtures";

export type FirstRunScreen = "welcome" | "host" | "connect" | "ghe-host" | "choose" | "tickets" | "agent";

export const SCREEN_FOR_STEP: Record<WizardStepId, FirstRunScreen> = {
  vcs: "connect",
  repo: "choose",
  issues: "tickets",
  agent: "agent",
  name: "agent",
};

export const STEP_FOR_SCREEN: Partial<Record<FirstRunScreen, WizardStepId>> = {
  host: "vcs",
  connect: "vcs",
  "ghe-host": "vcs",
  choose: "repo",
  tickets: "issues",
  agent: "agent",
};

export type GitHubScreenState = {
  pending: boolean;
  identity: boolean;
  ready: boolean;
  stayOnConnect: boolean;
  appConfigured: boolean;
  accountConnectionRequired: boolean;
};

export function screenWithGitHubConnection(screen: FirstRunScreen, github: GitHubScreenState): FirstRunScreen {
  if (github.pending) return screen;
  if (!github.accountConnectionRequired) {
    if (!github.appConfigured) return screen === "choose" ? "connect" : screen;
    if (screen === "welcome" || (screen === "connect" && !github.stayOnConnect)) return "choose";
    return screen;
  }
  if (screen === "welcome") return screen;
  if (!github.identity) return "connect";
  if (screen === "connect" && github.ready && !github.stayOnConnect) return "choose";
  if (screen === "choose" && !github.ready) return "connect";
  return screen;
}

export function screenWithVcsConnection(
  screen: FirstRunScreen,
  vcsHost: VcsHostId | null,
  github: GitHubScreenState,
  githubEnterprise: boolean,
): FirstRunScreen {
  if (screen === "host") return screen;
  if (vcsHost === "bitbucket") {
    return screen === "connect" ? "choose" : screen;
  }
  if (githubEnterprise) {
    if (github.ready) {
      if (screen === "welcome" || screen === "connect" || screen === "ghe-host") return "choose";
      return screen;
    }
    if (screen === "choose" || screen === "connect") return "ghe-host";
    return screen;
  }
  return screenWithGitHubConnection(screen, github);
}

export function screenWithoutAgent(screen: FirstRunScreen, agentGate: OnboardingAgentGate): FirstRunScreen {
  if (screen !== "agent" || agentGate === "show" || agentGate === "first") return screen;
  return "tickets";
}

export function screenWithModelSource(
  screen: FirstRunScreen,
  agentGate: OnboardingAgentGate,
  credentialChoice: OnboardingAgentCredentialChoice | null,
): FirstRunScreen {
  if (screen !== "tickets" || agentGate !== "first" || credentialChoice) return screen;
  return "agent";
}

export function screenWithoutIncompleteBacklog(
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
