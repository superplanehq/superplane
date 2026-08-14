import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  EMPTY_FACTORY_KEY,
  PRIMARY_FACTORY_KEY,
  defaultFactoriesFixture,
  emptyWorkOrdersFactoriesFixture,
} from "../__fixtures__/factoryPageResponses";
import { WorkOrdersPage } from "./WorkOrdersPage";

/**
 * Work Orders page. The shell renders `WorkOrdersLoadedView`, which owns
 * the title bar + Board/List/Table layouts, empty states, and the shared
 * inline actions.
 */
const meta = {
  title: "Factories/Pages/Work Orders",
  component: WorkOrdersPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof WorkOrdersPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const workOrdersPath = `workspaces/${PRIMARY_FACTORY_KEY}/work-orders`;
const emptyWorkspacePath = `workspaces/${EMPTY_FACTORY_KEY}/work-orders`;

/** Populated dataset. Default layout is Board and default scope is All. */
export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={workOrdersPath} factoriesFixture={defaultFactoriesFixture} />,
};

/** Only closed orders — Done lane holds every entry, other lanes empty. */
export const OnlyClosedOrders: Story = {
  name: "Only closed orders",
  render: () => <FactoriesHarness pathSuffix={workOrdersPath} factoriesFixture={emptyWorkOrdersFactoriesFixture} />,
};

/** New workspace with no orders yet — true empty state with the primary CTA. */
export const EmptyWorkspace: Story = {
  name: "Empty workspace",
  render: () => <FactoriesHarness pathSuffix={emptyWorkspacePath} factoriesFixture={defaultFactoriesFixture} />,
};
