import type { FactoriesFactoryOnboarding, MeDescribeVcsProviderOnboardingResponse } from "@/api-client";
import type { StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";

import { defaultFactoriesFixture, PRIMARY_FACTORY_ID, type FactoriesFixture } from "./factoryPageResponses";
import type { StorybookUsageReport } from "./usageReportFixtures";

/** GitHub connection every workspace story shares. */
export const GITHUB_CONNECTION_ID = "storybook-github-connection";
const CLAUDE_CONNECTION_ID = "storybook-claude-connection";

/** App repository the setup stories continue with. Served by the resources fixture. */
export const SETUP_APP_REPOSITORY = "acme/api";

const READY_GITHUB_ONBOARDING: MeDescribeVcsProviderOnboardingResponse = {
  providerConfigured: true,
  accountConnectionRequired: true,
  identity: { userId: "42", login: "forestileao" },
  repositories: [
    {
      repositoryId: "201",
      installationId: "101",
      fullName: SETUP_APP_REPOSITORY,
      defaultBranch: "main",
      accountLogin: "acme",
      accountType: "Organization",
      private: true,
    },
  ],
  pendingRequests: [],
  synchronizing: false,
};

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

/** Two ready Sentry connections, so the intake wizard can offer a choice. */
export const SENTRY_SETUP_INTEGRATIONS: StorybookOrgIntegration[] = [
  readyConnection("sentry", "storybook-sentry-connection", "acme-sentry"),
  readyConnection("sentry", "storybook-sentry-eu-connection", "acme-sentry-eu"),
];

/** Two ready Jira connections, so the intake wizard can offer a choice. */
export const JIRA_SETUP_INTEGRATIONS: StorybookOrgIntegration[] = [
  readyConnection("jira", "storybook-jira-connection", "acme-jira"),
  readyConnection("jira", "storybook-jira-eu-connection", "acme-jira-eu"),
];

/** GitHub and Claude both installed, the state of an organization that already ships with SuperPlane. */
export const CONNECTED_SETUP_INTEGRATIONS: StorybookOrgIntegration[] = [
  ...GITHUB_SETUP_INTEGRATIONS,
  readyConnection("claude", CLAUDE_CONNECTION_ID, "acme-claude"),
];

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

export function factoriesFixtureWithGithubAccess(fixture = defaultFactoriesFixture): FactoriesFixture {
  return { ...fixture, githubOnboarding: READY_GITHUB_ONBOARDING };
}

/** Default dataset with saved setup answers on the primary workspace. */
export function factoriesFixtureWithSetupAnswers(
  onboarding: FactoriesFactoryOnboarding,
  options?: { organizationWorkspaceUsage?: StorybookUsageReport },
): FactoriesFixture {
  const fixture = factoriesFixtureWithGithubAccess();
  return {
    ...fixture,
    organizationWorkspaceUsage: options?.organizationWorkspaceUsage ?? fixture.organizationWorkspaceUsage,
    factories: fixture.factories.map((factory) =>
      factory.id === PRIMARY_FACTORY_ID ? { ...factory, onboarding } : factory,
    ),
  };
}
