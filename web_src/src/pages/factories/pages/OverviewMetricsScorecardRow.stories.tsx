import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { OverviewMetricsScorecardRow } from "./OverviewMetricsScorecardRow";
import { buildOverviewVelocitySummary } from "./overviewVelocitySummary";

/**
 * Isolated scorecard row for Overview: four velocity cards (Merged PRs,
 * Waste, Cost, SuperPlane share) plus a "View velocity" link. Quick visual
 * review of the cards alone — see `Factories/Pages/Overview` for the row
 * mounted on the full page.
 */
const meta = {
  title: "Factories/Components/OverviewMetricsScorecardRow",
  component: OverviewMetricsScorecardRow,
  parameters: { layout: "padded" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-0 max-w-2xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
  },
} satisfies Meta<typeof OverviewMetricsScorecardRow>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Populated: Story = {
  args: {
    summary: buildOverviewVelocitySummary(),
  },
};

export const Empty: Story = {
  name: "No velocity data yet",
  args: {
    summary: null,
  },
};
