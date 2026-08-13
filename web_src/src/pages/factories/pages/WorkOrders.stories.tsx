import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyWorkOrdersFactoriesFixture,
  PRIMARY_FACTORY_ID,
} from "../__fixtures__/factoryPageResponses";
import { WorkOrdersPage } from "./WorkOrdersPage";

/**
 * Work Orders list inside the FactoriesLayout: filters, list, and "New Work Order".
 * New Work Order opens the create dialog (requires work_orders create permission in /me).
 */
const meta = {
  title: "Factories/Pages/Work Orders",
  component: WorkOrdersPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkOrdersPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const workOrdersPath = `workspaces/${PRIMARY_FACTORY_ID}/work-orders`;

/** Populated list — active/mine filter is the default. */
export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={workOrdersPath} factoriesFixture={defaultFactoriesFixture} />,
};

/** No active work orders — closed-only fixture. */
export const OnlyClosed: Story = {
  name: "Only closed orders",
  render: () => <FactoriesHarness pathSuffix={workOrdersPath} factoriesFixture={emptyWorkOrdersFactoriesFixture} />,
};
