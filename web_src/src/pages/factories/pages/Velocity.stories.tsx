import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { VelocityPage, VelocityPrototypePage } from "./VelocityPage";

/**
 * Velocity page: PR output plus Storybook work-order time (cycle and Waiting).
 */
const meta = {
  title: "Factories/Pages/Velocity",
  component: VelocityPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof VelocityPage>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/velocity`}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ velocity: VelocityPrototypePage }}
    />
  ),
};
