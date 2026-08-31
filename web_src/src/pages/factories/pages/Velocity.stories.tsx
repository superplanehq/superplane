import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { VelocityPage } from "./VelocityPage";
import {
  VelocityPrototypeEarlyUsagePage,
  VelocityPrototypePage,
  VelocityPrototypeZeroStatePage,
} from "./VelocityPrototypePage";

const meta = {
  title: "Factories/Pages/Velocity",
  component: VelocityPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof VelocityPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const velocityPath = `workspaces/${PRIMARY_FACTORY_KEY}/velocity`;

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ velocity: VelocityPrototypePage }}
    />
  ),
};

/** New workspace with no closed tasks — empty state that points at the board. */
export const ZeroState: Story = {
  name: "Zero state",
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ velocity: VelocityPrototypeZeroStatePage }}
    />
  ),
};

/**
 * A few hours after setup: people merges fill the period, SuperPlane has one
 * day of output. No period comparison, and medians name their small sample.
 */
export const EarlyUsage: Story = {
  name: "Early usage",
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ velocity: VelocityPrototypeEarlyUsagePage }}
    />
  ),
};
