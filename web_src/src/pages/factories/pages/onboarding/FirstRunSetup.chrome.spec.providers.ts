import { vi } from "bun:test";

vi.mock("./useFirstRunGitHubEnterprise", () => ({
  useFirstRunGitHubEnterprise: () => ({ connect: vi.fn(), resume: vi.fn(), ready: false }),
}));

vi.mock("./useBitbucketOnboarding", () => ({
  useBitbucketOnboarding: () => ({
    data: { providerConfigured: false, identity: undefined, repositories: [], installUrl: "" },
    isPending: false,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
  }),
}));

export const githubState = { pending: false, linked: true };
vi.mock("./useGitHubOnboarding", () => ({
  useGitHubOnboarding: () => ({
    data: {
      providerConfigured: true,
      accountConnectionRequired: true,
      identity: githubState.linked ? { userId: "42", login: "octocat" } : undefined,
      repositories: ["acme/api", "acme/billing", "other/tools"].map((fullName, index) => ({
        repositoryId: `${201 + index}`,
        installationId: "101",
        fullName,
        defaultBranch: "main",
      })),
      pendingRequests: [],
      synchronizing: false,
    },
    isPending: githubState.pending,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
    configureInstallation: { mutateAsync: vi.fn() },
  }),
  useGitHubInstallationChecks: vi.fn(),
}));

const workspaceUsage = { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" };
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: workspaceUsage,
    isSuccess: true,
    isPending: false,
    isError: false,
    refetch: async () => ({ data: workspaceUsage }),
  }),
}));
