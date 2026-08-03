import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import {
  CLOSED_WORK_ORDER,
  defaultFactoriesFixture,
  FAILED_WORK_ORDER,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  RUNNING_WORK_ORDER,
} from "./__fixtures__/factoryPageResponses";
import type { WorkOrderDetailLoadedView } from "./WorkOrderDetailLoadedView";

/**
 * The composed loaded view (header, description, timeline, assignees sidebar)
 * mounted via the shared harness so the assignees popover fetch works.
 * Same underlying component the page uses, exposed here so reviewers can
 * cycle through work order states without navigating the list.
 */
const meta = {
  title: "Factories/WorkOrderDetailLoadedView",
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkOrderDetailLoadedView>;

export default meta;

type Story = StoryObj<typeof meta>;

const orderPath = (orderId: string) => `factories/${PRIMARY_FACTORY_ID}/orders/${orderId}`;

/** Open — Dispatch/Complete/Reject visible. */
export const Open: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(OPEN_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Running — activity timeline shows one running dispatch step. */
export const Running: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(RUNNING_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Failed — status badge in red, failed step surfaced in the timeline. */
export const Failed: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(FAILED_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

/** Closed — action row hidden, "Closed as completed" footer visible. */
export const Closed: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={orderPath(CLOSED_WORK_ORDER.id ?? "")} factoriesFixture={defaultFactoriesFixture} />
  ),
};
