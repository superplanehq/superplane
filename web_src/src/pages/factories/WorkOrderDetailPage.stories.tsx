import type { Meta, StoryObj } from "@storybook/react-vite";

import { WorkOrderDetailPage } from "./WorkOrderDetailPage";
import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import {
  CLOSED_WORK_ORDER,
  defaultFactoriesFixture,
  FAILED_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  RUNNING_WORK_ORDER,
} from "./__fixtures__/factoryPageResponses";

/**
 * Work order detail: header (badge, title, dispatch/complete/reject),
 * description, activity timeline, and the assignees sidebar. One story
 * per representative state (open / running / failed / closed).
 */
const meta = {
  title: "Pages/Factories/WorkOrderDetail",
  component: WorkOrderDetailPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof WorkOrderDetailPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const orderPath = (orderId: string) => `factories/${PRIMARY_FACTORY_ID}/orders/${orderId}`;

/** Open, not yet dispatched — Dispatch/Complete/Reject actions available. */
export const Open: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(OPEN_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Currently running on a line — activity timeline shows in-flight steps. */
export const Running: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={orderPath(RUNNING_WORK_ORDER.id ?? "")}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

/** A line step failed — status badge and timeline flag the failed step. */
export const Failed: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(FAILED_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Closed as completed — action row hidden, "Closed as completed" footer visible. */
export const Closed: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(CLOSED_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};
