import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  EMPTY_FACTORY_KEY,
  PRIMARY_FACTORY_KEY,
  defaultFactoriesFixture,
  emptyWorkOrdersFactoriesFixture,
} from "../__fixtures__/factoryPageResponses";
import { CHECKOUT_RELIABILITY_MISSION, REFUNDS_V2_MISSION } from "./missions/missionMocks";
import { MissionsWorkOrdersPage } from "./missions/MissionsWorkOrdersPage";
import { WorkOrdersPage } from "./WorkOrdersPage";

/**
 * Work Orders page. Storybook uses the missions-enhanced page: missions
 * follow the board, list, and table layouts. The live app still renders
 * `WorkOrdersPage`.
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

function WorkOrdersWithMissionsPage() {
  return <MissionsWorkOrdersPage />;
}

const missionsPage = { workOrders: WorkOrdersWithMissionsPage };

/** Populated dataset. Default layout is Board and default scope is All. */
export const Populated: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={workOrdersPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={missionsPage}
    />
  ),
};

/** Only closed orders — Done lane holds every entry, other lanes empty. */
export const OnlyClosedOrders: Story = {
  name: "Only closed orders",
  render: () => (
    <FactoriesHarness
      pathSuffix={workOrdersPath}
      factoriesFixture={emptyWorkOrdersFactoriesFixture}
      pageOverrides={missionsPage}
    />
  ),
};

/** New workspace with no orders yet — true empty state with the primary CTA. */
export const EmptyWorkspace: Story = {
  name: "Empty workspace",
  render: () => (
    <FactoriesHarness
      pathSuffix={emptyWorkspacePath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={missionsPage}
    />
  ),
};

export const MissionDetailCheckout: Story = {
  name: "Mission detail — Checkout reliability",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/missions/${CHECKOUT_RELIABILITY_MISSION.id}`}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={missionsPage}
    />
  ),
};

export const MissionDetailRefunds: Story = {
  name: "Mission detail — Refunds v2",
  render: () => (
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/missions/${REFUNDS_V2_MISSION.id}`}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={missionsPage}
    />
  ),
};
