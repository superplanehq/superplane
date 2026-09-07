import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { WorkspaceUsageByMachineTypeTable, WorkspaceUsageTotals } from "./WorkspaceUsageBreakdown";

/**
 * Spending tab totals and by-machine-type breakdown. `formatDurationSeconds`
 * shows VM time in seconds, minutes, or hours depending on the amount —
 * these stories exercise each threshold.
 */
const meta = {
  title: "Factories/Settings/WorkspaceUsageBreakdown",
  parameters: { layout: "padded" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-4xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

/** VM time under a minute shows seconds only, e.g. "45 s". */
export const TotalsWithSeconds: Story = {
  name: "Totals — seconds",
  render: () => (
    <WorkspaceUsageTotals periodDays={30} totalTokens={12_400} totalCostCents={532} totalDurationSeconds={45} />
  ),
};

/** VM time between a minute and an hour shows minutes and seconds, e.g. "1 min 30 s". */
export const TotalsWithMinutes: Story = {
  name: "Totals — minutes",
  render: () => (
    <WorkspaceUsageTotals periodDays={30} totalTokens={128_000} totalCostCents={4210} totalDurationSeconds={90} />
  ),
};

/** VM time over an hour now shows hours and minutes, e.g. "1 h 30 min", instead of "90 min". */
export const TotalsWithHours: Story = {
  name: "Totals — hours",
  render: () => (
    <WorkspaceUsageTotals periodDays={30} totalTokens={980_000} totalCostCents={31_450} totalDurationSeconds={5400} />
  ),
};

/** Rows spanning all three VM time formats: seconds, minutes, and hours. */
export const ByMachineTypeAllRanges: Story = {
  name: "By machine type — seconds, minutes, hours",
  render: () => (
    <WorkspaceUsageByMachineTypeTable
      byMachineType={[
        { machineType: "small", durationSeconds: 45, costCents: 120 },
        { machineType: "standard", durationSeconds: 90, costCents: 640 },
        { machineType: "gpu-large", durationSeconds: 5400, costCents: 18_900 },
      ]}
    />
  ),
};

/** Empty state: no VM usage recorded for the period. */
export const ByMachineTypeEmpty: Story = {
  name: "By machine type — empty",
  render: () => <WorkspaceUsageByMachineTypeTable byMachineType={[]} />,
};
