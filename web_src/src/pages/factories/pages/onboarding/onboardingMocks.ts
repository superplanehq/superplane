export type GitProvider = "github" | "gitlab";

export type OnboardingRepo = {
  id: string;
  name: string;
  org: string;
  provider: GitProvider;
};

export type PendingOnboarding = {
  workspaceId: string;
  workspaceName: string;
};

export type OnboardingStorybookSeed = {
  pending?: PendingOnboarding | null;
  enabledReposByWorkspace?: Record<string, OnboardingRepo[]>;
};
