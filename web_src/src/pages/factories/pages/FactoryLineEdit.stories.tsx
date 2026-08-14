import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../__fixtures__/factoryPageResponses";
import { FactoryLineEditPage } from "./FactoryLineEditPage";

/**
 * Line editor — new and edit routes rendered inside the FactoriesLayout.
 */
const meta = {
  title: "Factories/Pages/Line Editor",
  component: FactoryLineEditPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactoryLineEditPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Create a new line. */
export const NewLine: Story = {
  name: "New line",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/new`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

/** Edit an existing line — pre-fills with the first refund line. */
export const EditLine: Story = {
  name: "Edit line",
  render: () => {
    const line = REFUND_FACTORY_LINES[0];
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}/edit`}
        factoriesFixture={defaultFactoriesFixture}
      />
    );
  },
};
