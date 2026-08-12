import { createContext } from "react";

import type { GitProvider, OnboardingRepo, PendingOnboarding } from "./onboardingMocks";

/** Sidebar unlock milestones while Storybook onboarding is pending. */
export type OnboardingNavProgress = {
  repoReady: boolean;
  issuesReady: boolean;
  agentReady: boolean;
  analyzingRepo: boolean;
  analyzingIssues: boolean;
  analyzingAgent: boolean;
};

export type OnboardingNavAnalyzingKey = "repo" | "issues" | "agent";

export const EMPTY_ONBOARDING_NAV_PROGRESS: OnboardingNavProgress = {
  repoReady: false,
  issuesReady: false,
  agentReady: false,
  analyzingRepo: false,
  analyzingIssues: false,
  analyzingAgent: false,
};

export type OnboardingStorybookContextValue = {
  pending: PendingOnboarding | null;
  connectedProviders: GitProvider[];
  setupProgress: OnboardingNavProgress;
  reportSetupProgress: (progress: Pick<OnboardingNavProgress, "repoReady" | "issuesReady" | "agentReady">) => void;
  reportNavAnalyzing: (key: OnboardingNavAnalyzingKey, analyzing: boolean) => void;
  beginOnboarding: (pending: PendingOnboarding) => void;
  connectProvider: (provider: GitProvider) => void;
  completeOnboarding: (workspaceId: string, repos: OnboardingRepo[]) => void;
  shouldShowOverviewTips: (workspaceId: string) => boolean;
  clearOverviewTips: () => void;
  enabledRepos: (workspaceId: string) => OnboardingRepo[];
};

export const OnboardingStorybookContext = createContext<OnboardingStorybookContextValue | null>(null);
