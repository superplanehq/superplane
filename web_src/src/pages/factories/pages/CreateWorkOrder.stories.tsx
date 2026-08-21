import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, EMPTY_FACTORY_KEY, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";

/**
 * Linear-style create dialog. `/work-orders/new` opens it over the list.
 * Click New Work Order on Work Orders stories to open the same dialog.
 */
const meta = {
  title: "Factories/Pages/Create Work Order",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const createPath = (factoryKey: string) => `workspaces/${factoryKey}/work-orders/new`;

export const Empty: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={createPath(PRIMARY_FACTORY_KEY)} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Assigned: Story = {
  name: "Assignees available",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const NoLines: Story = {
  name: "No lines",
  render: () => (
    <FactoriesHarness pathSuffix={createPath(EMPTY_FACTORY_KEY)} factoriesFixture={defaultFactoriesFixture} />
  ),
};
