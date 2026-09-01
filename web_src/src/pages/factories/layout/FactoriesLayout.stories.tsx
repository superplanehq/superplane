import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { FactoriesLayout } from "./FactoriesLayout";

/**
 * The Factories layout shell: a thin icon rail (workspace, intake, board,
 * velocity, settings, new task, user) and the line board as the main pane.
 * Stories mount the layout with a live route so the rail controls behave.
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
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/line-plan-and-implement`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};
