import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_KEY,
  RUNNING_WORK_ORDER,
  CLOSED_WORK_ORDER,
} from "../__fixtures__/factoryPageResponses";
import { WorkOrderDetailPage } from "./WorkOrderDetailPage";

/**
 * Work Order Detail — full detail view inside the FactoriesLayout.
 * Storybook Overview includes a Mission picker. The live page does not.
 */
const meta = {
  title: "Factories/Pages/Work Order Detail",
  component: WorkOrderDetailPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkOrderDetailPage>;

export default meta;

type Story = StoryObj<typeof meta>;

function detailPath(orderNumber?: string) {
  return `workspaces/${PRIMARY_FACTORY_KEY}/work-order/${orderNumber ?? OPEN_WORK_ORDER.number}`;
}

export const Open: Story = {
  render: () => <FactoriesHarness pathSuffix={detailPath()} factoriesFixture={defaultFactoriesFixture} />,
};

export const Running: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={detailPath(RUNNING_WORK_ORDER.number)} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Closed: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={detailPath(CLOSED_WORK_ORDER.number)} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Draft order with no mission — use Overview to assign one. */
export const NoMission: Story = {
  name: "No mission",
  render: () => (
    <FactoriesHarness pathSuffix={detailPath(DRAFT_WORK_ORDER.number)} factoriesFixture={defaultFactoriesFixture} />
  ),
};
