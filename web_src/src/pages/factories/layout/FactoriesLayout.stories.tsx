import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { FactoriesLayout } from "./FactoriesLayout";

/**
 * The Factories layout shell: workspace name (opens Overview), nav + recent
 * orders, bottom-left user menu, and a compact section header in the pane.
 * Stories mount the layout with a live route so the sidebar links behave.
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
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/overview`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
