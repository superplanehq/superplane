import type { VcsHostId, WizardStepId } from "./onboardingFixtures";

export type FirstRunScreen =
  | "welcome"
  | "host"
  | "connect"
  | "ghe-host"
  | "choose"
  | "tickets"
  | "card"
  | "agent"
  | "complete";

export const URL_ONLY_SCREENS: ReadonlySet<string> = new Set<FirstRunScreen>(["welcome", "host", "card", "complete"]);
export const FINAL_SCREENS: ReadonlySet<FirstRunScreen> = new Set<FirstRunScreen>(["card", "agent", "complete"]);

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

export function githubScreenState(args: {
  githubEnterprise: boolean;
  pending: boolean;
  identity: boolean;
  ready: boolean;
  enterpriseReady: boolean;
  stayOnConnect: boolean;
  appConfigured: boolean;
  accountConnectionRequired: boolean;
}): GitHubScreenState {
  if (args.githubEnterprise) {
    return {
      pending: false,
      identity: args.identity,
      ready: args.enterpriseReady,
      stayOnConnect: args.stayOnConnect,
      appConfigured: args.enterpriseReady,
      accountConnectionRequired: false,
    };
  }
  return {
    pending: args.pending,
    identity: args.identity,
    ready: args.ready,
    stayOnConnect: args.stayOnConnect,
    appConfigured: args.appConfigured,
    accountConnectionRequired: args.accountConnectionRequired,
  };
}

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

function screenWithGitHubEnterprise(screen: FirstRunScreen, github: GitHubScreenState): FirstRunScreen {
  if (github.ready) {
    if (screen === "welcome" || screen === "connect") return "choose";
    return screen;
  }
  if (screen === "choose" || screen === "connect") return "ghe-host";
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
  if (githubEnterprise) return screenWithGitHubEnterprise(screen, github);
  if (screen !== "connect" && screen !== "choose") return screen;
  return screenWithGitHubConnection(screen, github);
}
