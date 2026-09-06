import type { FactoriesFactoryOnboarding } from "@/api-client";
import { accountOrganizationsQueryKey } from "@/hooks/useAccountOrganizations";
import type { StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { useQueryClient } from "@tanstack/react-query";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  GITHUB_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
  soloFactoriesFixtureWithSetupAnswers,
} from "../__fixtures__/setupStoryFixtures";
import { NO_GRANT_USAGE_REPORT, type StorybookUsageReport } from "../__fixtures__/usageReportFixtures";
import type { WizardStepId } from "./onboarding/onboardingFixtures";

/**
 * Workspace setup, one story per step. Every story mounts the same
 * `OnboardingPage` the app routes to, so the stories match production. Setup
 * runs on the first-run screens and saves each answer through a fixture
 * backend. The sidebar stays hidden until setup finishes.
 */
const meta = {
  title: "Factories/Pages/Setup",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const pendingSeed = {
  pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Semaphore" },
};

/**
 * Opens setup on one step, with the earlier answers already saved. `?step=` is
 * the same deep link the app uses when the browser returns from a provider.
 */
function SetupStep({
  step,
  answers = SETUP_ANSWERS.none,
  orgIntegrations = CONNECTED_SETUP_INTEGRATIONS,
  organizationWorkspaceUsage,
}: {
  step: WizardStepId;
  answers?: FactoriesFactoryOnboarding;
  orgIntegrations?: StorybookOrgIntegration[];
  organizationWorkspaceUsage?: StorybookUsageReport;
}) {
  return (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=${step}`}
      factoriesFixture={factoriesFixtureWithSetupAnswers(answers, { organizationWorkspaceUsage })}
      onboardingSeed={pendingSeed}
      orgIntegrations={orgIntegrations}
    />
  );
}

/** A new workspace with no saved answers opens the welcome screen. */
export const Welcome: Story = {
  name: "0 Welcome",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup`}
      factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.none)}
      onboardingSeed={pendingSeed}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
    />
  ),
};

/**
 * A single workspace in a single organization: the account menu's "Quit
 * onboarding" has nowhere to send the user, so it falls back to Sign out
 * only. Every other story in this file has another workspace or
 * organization to leave to, so it shows Quit onboarding instead.
 */
function SoloWorkspaceSetup() {
  const queryClient = useQueryClient();
  queryClient.setQueryData(accountOrganizationsQueryKey, [
    { id: FACTORIES_ORGANIZATION_ID, slug: "superplane", name: "SuperPlane" },
  ]);
  return (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup`}
      factoriesFixture={soloFactoriesFixtureWithSetupAnswers(SETUP_ANSWERS.none)}
      onboardingSeed={pendingSeed}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
    />
  );
}

export const WelcomeSignOutOnly: Story = {
  name: "0 Welcome (sign out only)",
  render: () => <SoloWorkspaceSetup />,
};

/** First workspace in a new organization: nothing is connected yet. */
export const ConnectZero: Story = {
  name: "1a Connect GitHub (zero)",
  render: () => <SetupStep step="vcs" orgIntegrations={[]} />,
};

export const Connect: Story = {
  name: "1b Connect GitHub",
  render: () => <SetupStep step="vcs" orgIntegrations={GITHUB_SETUP_INTEGRATIONS} />,
};

export const Repository: Story = {
  name: "2 Repository",
  render: () => <SetupStep step="repo" answers={SETUP_ANSWERS.vcs} orgIntegrations={GITHUB_SETUP_INTEGRATIONS} />,
};

export const Issues: Story = {
  name: "3 Issues",
  render: () => (
    <SetupStep step="issues" answers={SETUP_ANSWERS.repository} orgIntegrations={GITHUB_SETUP_INTEGRATIONS} />
  ),
};

/** The organization has hosted credit, so the user can continue without keys. */
export const AgentWithGrant: Story = {
  name: "4a Agent (hosted credit)",
  render: () => <SetupStep step="agent" answers={SETUP_ANSWERS.issues} orgIntegrations={GITHUB_SETUP_INTEGRATIONS} />,
};

/** No grant: the user must connect Anthropic, OpenAI, or OpenRouter. */
export const AgentWithoutGrant: Story = {
  name: "4b Agent (connect a provider)",
  render: () => (
    <SetupStep
      step="agent"
      answers={SETUP_ANSWERS.issues}
      orgIntegrations={GITHUB_SETUP_INTEGRATIONS}
      organizationWorkspaceUsage={NO_GRANT_USAGE_REPORT}
    />
  ),
};
