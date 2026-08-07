import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "../__fixtures__/factoryPageResponses";
import { FactoriesLayout } from "./FactoriesLayout";

/**
 * The Factories layout shell: workspace switcher on top, nav + recent orders,
 * bottom-left user menu, and content area for nested routes. Stories mount
 * the layout with a live route so the sidebar links behave.
 */
const meta = {
  title: "Factories/Layout/FactoriesLayout",
  component: FactoriesLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactoriesLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ID}/overview`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
