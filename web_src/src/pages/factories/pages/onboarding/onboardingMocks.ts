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

/** Mock catalog shown after a provider is connected (Storybook-only). */
export const ONBOARDING_AVAILABLE_REPOS: OnboardingRepo[] = [
  { id: "gh-superplane", name: "superplane", org: "superplanehq", provider: "github" },
  { id: "gh-skills", name: "skills", org: "superplanehq", provider: "github" },
  { id: "gh-docs", name: "docs", org: "superplanehq", provider: "github" },
  { id: "gh-v3", name: "v3", org: "superplanehq", provider: "github" },
  { id: "gh-infra", name: "infra", org: "superplanehq", provider: "github" },
  { id: "gl-backend", name: "backend", org: "acme-ops", provider: "gitlab" },
  { id: "gl-frontend", name: "frontend", org: "acme-ops", provider: "gitlab" },
  { id: "gl-services", name: "services", org: "acme-ops", provider: "gitlab" },
  { id: "gl-pipelines", name: "pipelines", org: "acme-ops", provider: "gitlab" },
];

export function providerLabel(provider: GitProvider) {
  return provider === "github" ? "GitHub" : "GitLab";
}

export type OnboardingStorybookSeed = {
  pending?: PendingOnboarding | null;
  connectedProviders?: GitProvider[];
  enabledReposByWorkspace?: Record<string, OnboardingRepo[]>;
  overviewTipsWorkspaceId?: string | null;
};
