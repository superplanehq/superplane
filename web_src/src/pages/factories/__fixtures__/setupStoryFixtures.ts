import type { FactoriesFactoryOnboarding } from "@/api-client";
import type { StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";

import { defaultFactoriesFixture, PRIMARY_FACTORY_ID, type FactoriesFixture } from "./factoryPageResponses";
import type { StorybookUsageReport } from "./usageReportFixtures";

const GITHUB_CONNECTION_ID = "storybook-github-connection";
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
  options?: { organizationLlmSpend?: StorybookUsageReport },
): FactoriesFixture {
  return {
    ...defaultFactoriesFixture,
    organizationLlmSpend: options?.organizationLlmSpend ?? defaultFactoriesFixture.organizationLlmSpend,
    factories: defaultFactoriesFixture.factories.map((factory) =>
      factory.id === PRIMARY_FACTORY_ID ? { ...factory, onboarding } : factory,
    ),
  };
}
