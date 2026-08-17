import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyWorkOrdersFactoriesFixture,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";
import { OverviewMetricsScorecardRowEmpty } from "./OverviewMetricsScorecardRow";
import { OverviewPage } from "./OverviewPage";

/**
 * Overview page: compact section header aligned with the sidebar workspace
 * name, plus a velocity scorecard row and the recent work orders list inside
 * FactoriesLayout. Mounted through the real router so the sidebar (nav +
 * recent) appears. Canvas clicks stay on FactoryAppCanvasPage because
 * FactoriesHarness serves a factory-owned canvas by default.
 *
 * The scorecard row and the removed Automations box only show through this
 * Storybook path: `FactoriesHarness` fills `OverviewMetricsSlotContext`,
 * which the live app leaves empty.
 */
const meta = {
  title: "Factories/Pages/Overview",
  component: OverviewPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof OverviewPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const overviewPath = `workspaces/${PRIMARY_FACTORY_KEY}/overview`;

export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={overviewPath} factoriesFixture={defaultFactoriesFixture} />,
};

export const EmptyWorkOrders: Story = {
  name: "No active work orders",
  render: () => <FactoriesHarness pathSuffix={overviewPath} factoriesFixture={emptyWorkOrdersFactoriesFixture} />,
};

export const EmptyVelocity: Story = {
  name: "No velocity data yet",
  render: () => (
    <FactoriesHarness
      pathSuffix={overviewPath}
      factoriesFixture={defaultFactoriesFixture}
      overviewMetricsSlot={OverviewMetricsScorecardRowEmpty}
    />
  ),
};
