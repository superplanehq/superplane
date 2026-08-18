import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";
import { ONBOARDING_AVAILABLE_REPOS } from "./onboarding/onboardingMocks";
import { OnboardingWireframe } from "./onboarding/OnboardingWireframe";

/**
 * Storybook-only workspace setup (setup stack + setup.log).
 * Sidebar stays hidden until setup finishes. Production create still
 * navigates straight to overview.
 */
const meta = {
  title: "Factories/Pages/Setup",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryBase = `workspaces/${PRIMARY_FACTORY_KEY}`;

const pendingSeed = {
  pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Refunds Factory" },
};

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/setup`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={pendingSeed}
    />
  ),
};

export const EmptyCreate: Story = {
  name: "Empty → Create",
  render: () => <FactoriesHarness pathSuffix="workspaces" factoriesFixture={emptyFactoriesFixture} />,
};

export const GettingStarted: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/overview`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={{
        enabledReposByWorkspace: {
          [PRIMARY_FACTORY_ID]: ONBOARDING_AVAILABLE_REPOS.filter((repo) => repo.provider === "github").slice(0, 2),
        },
        overviewTipsWorkspaceId: PRIMARY_FACTORY_ID,
      }}
    />
  ),
};

/** Pending setup redirects non-setup routes back to setup (no sidebar). */
export const Gated: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`${factoryBase}/overview`}
      factoriesFixture={defaultFactoriesFixture}
      onboardingSeed={pendingSeed}
    />
  ),
};

void OnboardingWireframe;
