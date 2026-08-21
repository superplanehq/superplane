import type { FactoriesFactoryOnboarding } from "@/api-client";
import type { StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  GITHUB_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../__fixtures__/setupStoryFixtures";
import type { WizardStepId } from "./onboarding/onboardingFixtures";

/**
 * Workspace setup, one story per wizard step. Every story mounts the same
 * `OnboardingPage` the app routes to, so the stories match production. A
 * fixture backend serves the connect flow, the repository list, and the saved
 * setup answers. The sidebar stays hidden until setup finishes.
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
}: {
  step: WizardStepId;
  answers?: FactoriesFactoryOnboarding;
  orgIntegrations?: StorybookOrgIntegration[];
}) {
  return (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=${step}`}
      factoriesFixture={factoriesFixtureWithSetupAnswers(answers)}
      onboardingSeed={pendingSeed}
      orgIntegrations={orgIntegrations}
    />
  );
}

/** First workspace in a new organization: nothing is connected yet. */
export const ChooseVcsZero: Story = {
  name: "1a Choose VCS (zero)",
  render: () => <SetupStep step="vcs" orgIntegrations={[]} />,
};

export const ChooseVcs: Story = {
  name: "1b Choose VCS",
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

/** The organization has GitHub but no coding agent, so the step asks to connect one. */
export const Agent: Story = {
  name: "4 Agent",
  render: () => <SetupStep step="agent" answers={SETUP_ANSWERS.issues} orgIntegrations={GITHUB_SETUP_INTEGRATIONS} />,
};

export const Name: Story = {
  name: "5 Name",
  render: () => <SetupStep step="name" answers={SETUP_ANSWERS.agent} />,
};
