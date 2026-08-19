import type { Meta, StoryObj } from "@storybook/react-vite";

import type { FactoryVelocityFlow } from "../lib/factoryVelocityFlow";
import { subHourVelocityDurationFormat } from "../lib/velocityDurationFormat";
import { VelocityDurationFormatSlotContext } from "../lib/velocityDurationFormatSlot";
import { FACTORY_VELOCITY_FLOW_BY_PERIOD } from "./factoryVelocityFlowMockData";
import { WorkOrderFlowCard } from "./VelocityLoadedView";

/**
 * Support stories for the Work order time card (scorecards plus the Time
 * running / Time in Waiting chart) in isolation, for quick sub-hour /
 * in-range / multi-day unit checks. Not the review home — see
 * Factories/Pages/Velocity → Default for the real screen with these labels.
 */

function toFlow(periodDays: 7 | 30): FactoryVelocityFlow {
  const mock = FACTORY_VELOCITY_FLOW_BY_PERIOD[periodDays];
  return {
    days: periodDays,
    label: mock.label,
    sampleSize: 24,
    medianCycleHours: mock.medianCycleHours,
    medianRunningHours: mock.medianRunningHours,
    medianWaitingHours: mock.medianWaitingHours,
    runningShareOfCyclePct: mock.runningShareOfCyclePct,
    waitingShareOfCyclePct: mock.waitingShareOfCyclePct,
    timeTrend: mock.timeTrend,
  };
}

const MULTI_DAY_FLOW: FactoryVelocityFlow = {
  days: 30,
  label: "Last 30 days",
  sampleSize: 18,
  medianCycleHours: 120,
  medianRunningHours: 46,
  medianWaitingHours: 74,
  runningShareOfCyclePct: 38,
  waitingShareOfCyclePct: 62,
  timeTrend: [
    { day: "1", runningHours: 40, waitingHours: 60 },
    { day: "", runningHours: 44, waitingHours: 64 },
    { day: "", runningHours: 48, waitingHours: 68 },
    { day: "", runningHours: 44, waitingHours: 72 },
    { day: "", runningHours: 50, waitingHours: 76 },
    { day: "", runningHours: 46, waitingHours: 80 },
    { day: "30", runningHours: 48, waitingHours: 78 },
  ],
};

const meta = {
  title: "Factories/Components/Velocity Work Order Time",
  component: WorkOrderFlowCard,
  decorators: [
    (Story) => (
      <VelocityDurationFormatSlotContext.Provider value={subHourVelocityDurationFormat}>
        <div className="max-w-3xl">
          <Story />
        </div>
      </VelocityDurationFormatSlotContext.Provider>
    ),
  ],
} satisfies Meta<typeof WorkOrderFlowCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Every work order closed under an hour: scorecards read in minutes and the axis picks minutes too. */
export const AllSubHourPeriod: Story = {
  args: {
    periodLabel: "Last 7 days",
    periodDays: 7,
    config: { flow: toFlow(7) },
  },
};

/** An hour-scale month with a genuine zero day and one sub-hour day mixed in. */
export const MixedPeriod: Story = {
  args: {
    periodLabel: "Last 30 days",
    periodDays: 30,
    config: { flow: toFlow(30) },
  },
};

/** Medians and trend well past two days: scorecards and axis both read in days. */
export const MultiDayPeriod: Story = {
  args: {
    periodLabel: "Last 30 days",
    periodDays: 30,
    config: { flow: MULTI_DAY_FLOW },
  },
};
