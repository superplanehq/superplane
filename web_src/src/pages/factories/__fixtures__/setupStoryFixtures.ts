import type { FactoriesFactoryOnboarding } from "@/api-client";
import type { StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";

import { defaultFactoriesFixture, PRIMARY_FACTORY_ID, type FactoriesFixture } from "./factoryPageResponses";
import type { StorybookUsageReport } from "./usageReportFixtures";

/** GitHub connection every workspace story shares. */
export const GITHUB_CONNECTION_ID = "storybook-github-connection";
const CLAUDE_CONNECTION_ID = "storybook-claude-connection";

/** App repository the setup stories continue with. Served by the resources fixture. */
export const SETUP_APP_REPOSITORY = "acme/api";

function readyConnection(integrationName: string, id: string, name: string): StorybookOrgIntegration {
  return {
    metadata: { id, name, integrationName },
    status: { state: "ready" },
    spec: { configuration: {} },
  };
}

/** GitHub installed in the organization, no coding agent yet. */
export const GITHUB_SETUP_INTEGRATIONS: StorybookOrgIntegration[] = [
  readyConnection("github", GITHUB_CONNECTION_ID, "acme-github"),
];

/** GitHub and Claude both installed, the state of an organization that already ships with SuperPlane. */
export const CONNECTED_SETUP_INTEGRATIONS: StorybookOrgIntegration[] = [
  ...GITHUB_SETUP_INTEGRATIONS,
  readyConnection("claude", CLAUDE_CONNECTION_ID, "acme-claude"),
];

/**
 * A GitHub connect the storybook user just authorized on GitHub: still
 * pending, with the account picker data the OAuth callback stored.
 */
export const PENDING_PICKER_INTEGRATION: StorybookOrgIntegration = {
  metadata: { id: "storybook-github-pending", name: "github-2", integrationName: "github" },
  status: {
    state: "pending",
    metadata: {
      startedByUserID: "storybook-user",
      startedByGitHubLogin: "forestileao",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [
        { id: "11", accountLogin: "forestigamer" },
        { id: "22", accountLogin: "forestileao" },
      ],
    },
  },
  spec: { configuration: {} },
};

const vcsAnswered: FactoriesFactoryOnboarding = { vcsIntegrationId: GITHUB_CONNECTION_ID };
const repositoryAnswered: FactoriesFactoryOnboarding = { ...vcsAnswered, appRepository: SETUP_APP_REPOSITORY };
const issuesAnswered: FactoriesFactoryOnboarding = {
  ...repositoryAnswered,
  backlogRepository: SETUP_APP_REPOSITORY,
  issuesSource: "ISSUES_SOURCE_VCS",
};
const agentAnswered: FactoriesFactoryOnboarding = {
  ...issuesAnswered,
  agentIntegrationId: CLAUDE_CONNECTION_ID,
  agentHarness: "AGENT_HARNESS_CLAUDE_CODE",
};

/**
 * Answers the API already holds when a story opens a later wizard step, named
 * after the last step that was answered. Setup restores them the same way it
 * does for a user who resumes an unfinished workspace.
 */
export const SETUP_ANSWERS = {
  none: {},
  vcs: vcsAnswered,
  repository: repositoryAnswered,
  issues: issuesAnswered,
  agent: agentAnswered,
} satisfies Record<string, FactoriesFactoryOnboarding>;

/** Default dataset with saved setup answers on the primary workspace. */
export function factoriesFixtureWithSetupAnswers(
  onboarding: FactoriesFactoryOnboarding,
  options?: { organizationWorkspaceUsage?: StorybookUsageReport },
): FactoriesFixture {
  return {
    ...defaultFactoriesFixture,
    organizationWorkspaceUsage:
      options?.organizationWorkspaceUsage ?? defaultFactoriesFixture.organizationWorkspaceUsage,
    factories: defaultFactoriesFixture.factories.map((factory) =>
      factory.id === PRIMARY_FACTORY_ID ? { ...factory, onboarding } : factory,
    ),
  };
}
