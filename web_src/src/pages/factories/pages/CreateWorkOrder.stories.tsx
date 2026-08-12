import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "../__fixtures__/factoryPageResponses";
import { CreateWorkOrderPage } from "./CreateWorkOrderPage";

/**
 * "New work order" form inside the FactoriesLayout — title, description, assignees.
 */
const meta = {
  title: "Factories/Pages/Create Work Order",
  component: CreateWorkOrderPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof CreateWorkOrderPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const createPath = `workspaces/${PRIMARY_FACTORY_ID}/work-orders/new`;

export const Empty: Story = {
  render: () => <FactoriesHarness pathSuffix={createPath} factoriesFixture={defaultFactoriesFixture} />,
};
