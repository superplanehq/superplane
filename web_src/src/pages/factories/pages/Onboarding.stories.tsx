import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyFactoriesFixture,
  PRIMARY_FACTORY_ID,
} from "../__fixtures__/factoryPageResponses";
import { ONBOARDING_AVAILABLE_REPOS } from "./onboarding/onboardingMocks";
import { OnboardingWireframe } from "./onboarding/OnboardingWireframe";

/**
 * Storybook-only workspace onboarding funnel (v3 parity).
 * Production create still navigates straight to overview.
 */
const meta = {
  title: "Factories/Pages/Onboarding",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryBase = `workspaces/${PRIMARY_FACTORY_ID}`;

const pendingSeed = {
  pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Refunds Factory" },
};

export const EmptyCreate: Story = {
  name: "Empty → Create",
  render: () => <FactoriesHarness pathSuffix="workspaces" factoriesFixture={emptyFactoriesFixture} />,
};

export const ConnectProviders: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/onboarding`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={pendingSeed}
    />
  ),
};

export const SelectRepositories: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/onboarding`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={{
        ...pendingSeed,
        connectedProviders: ["github", "gitlab"],
      }}
    />
  ),
};

export const GettingStarted: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/overview`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={{
        connectedProviders: ["github"],
        enabledReposByWorkspace: {
          [PRIMARY_FACTORY_ID]: ONBOARDING_AVAILABLE_REPOS.filter((repo) => repo.provider === "github").slice(0, 2),
        },
        overviewTipsWorkspaceId: PRIMARY_FACTORY_ID,
      }}
    />
  ),
};

export const Gated: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/missions`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={pendingSeed}
    />
  ),
};

void OnboardingWireframe;
