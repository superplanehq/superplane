import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  RUNNING_WORK_ORDER,
  CLOSED_WORK_ORDER,
} from "../__fixtures__/factoryPageResponses";
import { WorkOrderDetailPage } from "./WorkOrderDetailPage";

/**
 * Work Order Detail — full detail view inside the FactoriesLayout.
 * Uses the shared fixture so the sidebar and detail render together.
 */
const meta = {
  title: "Factories/Pages/Work Order Detail",
  component: WorkOrderDetailPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkOrderDetailPage>;

export default meta;

type Story = StoryObj<typeof meta>;

function detailPath(orderId?: string) {
  return `workspaces/${PRIMARY_FACTORY_ID}/work-orders/${orderId ?? OPEN_WORK_ORDER.id}`;
}

export const Open: Story = {
  render: () => <FactoriesHarness pathSuffix={detailPath()} factoriesFixture={defaultFactoriesFixture} />,
};

export const Running: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={detailPath(RUNNING_WORK_ORDER.id)} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Closed: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={detailPath(CLOSED_WORK_ORDER.id)} factoriesFixture={defaultFactoriesFixture} />
  ),
};
