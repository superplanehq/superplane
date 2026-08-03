import type { Meta, StoryObj } from "@storybook/react-vite";

import { CreateWorkOrderPage } from "./CreateWorkOrderPage";
import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "./__fixtures__/factoryPageResponses";

/**
 * "Describe the work" form for creating a new work order under a factory.
 * Interactively fillable — the assignees popover pulls the seeded org roster
 * from the shared fixture backend.
 */
const meta = {
  title: "Pages/Factories/CreateWorkOrder",
  component: CreateWorkOrderPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof CreateWorkOrderPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const newOrderPath = `factories/${PRIMARY_FACTORY_ID}/orders/new`;

/** Default: blank form ready for input; Create disabled until title is filled. */
export const Default: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={newOrderPath} factoriesFixture={defaultFactoriesFixture} />
  ),
};
