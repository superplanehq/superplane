import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { subHourVelocityDurationFormat } from "../lib/velocityDurationFormat";
import { VelocityDurationFormatSlotContext } from "../lib/velocityDurationFormatSlot";
import { FACTORY_VELOCITY_BY_PERIOD, FACTORY_VELOCITY_YESTERDAY } from "./factoryVelocityMockData";
import { FACTORY_VELOCITY_FLOW_BY_PERIOD } from "./factoryVelocityFlowMockData";
import { VelocityLoadedView, type VelocityData, type VelocitySourceSplitConfig } from "./VelocityLoadedView";

function toData(): VelocityData {
  const period = FACTORY_VELOCITY_BY_PERIOD[7];
  return {
    yesterday: {
      dateLabel: FACTORY_VELOCITY_YESTERDAY.dateLabel,
      merged: FACTORY_VELOCITY_YESTERDAY.merged,
      waste: FACTORY_VELOCITY_YESTERDAY.waste,
      wastePct: FACTORY_VELOCITY_YESTERDAY.wastePct,
    },
    totals: {
      merged: period.totals.merged,
      waste: period.totals.waste,
      wastePct: period.totals.wastePct,
      superplaneMerged: period.totals.superplaneMerged,
      peopleMerged: period.totals.peopleMerged,
      superplaneSharePct: period.totals.superplaneSharePct,
    },
    points: period.points.map((point) => ({
      day: point.day,
      merged: point.merged,
      waste: point.waste,
      peopleMerged: point.peopleMerged,
      superplaneMerged: point.superplaneMerged,
    })),
  };
}

describe("VelocityLoadedView", () => {
  it("renders yesterday and SuperPlane output cards without cost by default", () => {
    const data = toData();
    const sourceSplit: VelocitySourceSplitConfig = {
      hasPeopleCohort: true,
      repositoryLabel: "acme/refunds",
    };

    render(<VelocityLoadedView periodLabel="Last 7 days" periodDays={7} data={data} sourceSplit={sourceSplit} />);

    const yesterday = screen.getByTestId("velocity-yesterday");
    expect(yesterday).toHaveTextContent(String(FACTORY_VELOCITY_YESTERDAY.merged));
    expect(yesterday).toHaveTextContent("Merged PRs");
    expect(yesterday).toHaveTextContent("Waste");
    expect(yesterday).not.toHaveTextContent("Cost");

    const trend = screen.getByTestId("velocity-trend");
    expect(trend).toHaveTextContent("Last 7 days");
    expect(trend).toHaveTextContent(String(FACTORY_VELOCITY_BY_PERIOD[7].totals.merged));
    expect(trend).not.toHaveTextContent("Cost");

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("SuperPlane");
    expect(split).toHaveTextContent("People");
    expect(split).toHaveTextContent(`${FACTORY_VELOCITY_BY_PERIOD[7].totals.superplaneSharePct}%`);
    expect(split).toHaveTextContent("acme/refunds");

    expect(screen.queryByTestId("velocity-work-order-flow")).not.toBeInTheDocument();
  });

  it("hides People metrics and shows the empty-state slot when hasPeopleCohort is false", () => {
    const data = toData();
    const sourceSplit: VelocitySourceSplitConfig = {
      hasPeopleCohort: false,
      emptyState: <span>Pick a repository first.</span>,
    };

    render(<VelocityLoadedView periodLabel="Last 7 days" periodDays={7} data={data} sourceSplit={sourceSplit} />);

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("Pick a repository first.");
    expect(split).not.toHaveTextContent(String(FACTORY_VELOCITY_BY_PERIOD[7].totals.peopleMerged));
  });

  it("renders the work-order flow card when a flow config is provided", () => {
    const mock = FACTORY_VELOCITY_FLOW_BY_PERIOD[7];
    const data = toData();
    const sourceSplit: VelocitySourceSplitConfig = { hasPeopleCohort: true };

    render(
      <VelocityLoadedView
        periodLabel="Last 7 days"
        periodDays={7}
        data={data}
        sourceSplit={sourceSplit}
        workOrderFlow={{
          flow: {
            days: 7,
            label: "Last 7 days",
            sampleSize: 5,
            medianCycleHours: mock.medianCycleHours,
            medianRunningHours: mock.medianRunningHours,
            medianWaitingHours: mock.medianWaitingHours,
            runningShareOfCyclePct: mock.runningShareOfCyclePct,
            waitingShareOfCyclePct: mock.waitingShareOfCyclePct,
            timeTrend: mock.timeTrend,
          },
        }}
      />,
    );

    const flow = screen.getByTestId("velocity-work-order-flow");
    expect(flow).toHaveTextContent("Work order time");
    expect(flow).toHaveTextContent("Cycle time");
    expect(flow).toHaveTextContent("Time running");
    expect(flow).toHaveTextContent("Time in Waiting");
    expect(flow).toHaveTextContent(`${mock.runningShareOfCyclePct}% of cycle time`);
    expect(flow).toHaveTextContent(`${mock.waitingShareOfCyclePct}% of cycle time`);
  });

  it("reads sub-hour medians in minutes when the duration format slot is provided (Storybook prototype)", () => {
    const mock = FACTORY_VELOCITY_FLOW_BY_PERIOD[7];
    const data = toData();
    const sourceSplit: VelocitySourceSplitConfig = { hasPeopleCohort: true };

    render(
      <VelocityDurationFormatSlotContext.Provider value={subHourVelocityDurationFormat}>
        <VelocityLoadedView
          periodLabel="Last 7 days"
          periodDays={7}
          data={data}
          sourceSplit={sourceSplit}
          workOrderFlow={{
            flow: {
              days: 7,
              label: "Last 7 days",
              sampleSize: 5,
              medianCycleHours: mock.medianCycleHours,
              medianRunningHours: mock.medianRunningHours,
              medianWaitingHours: mock.medianWaitingHours,
              runningShareOfCyclePct: mock.runningShareOfCyclePct,
              waitingShareOfCyclePct: mock.waitingShareOfCyclePct,
              timeTrend: mock.timeTrend,
            },
          }}
        />
      </VelocityDurationFormatSlotContext.Provider>,
    );

    const flow = screen.getByTestId("velocity-work-order-flow");
    // The mock's sub-hour medians (0.45h / 0.2h / 0.25h) must read as minutes,
    // not collapse to "0h" the way the default whole-hour formatter would.
    expect(flow).toHaveTextContent("27m");
    expect(flow).toHaveTextContent("12m");
    expect(flow).toHaveTextContent("15m");
    expect(flow).not.toHaveTextContent("0h");
  });

  it("shows the empty label when the flow has no closed samples", () => {
    const data = toData();
    const sourceSplit: VelocitySourceSplitConfig = { hasPeopleCohort: true };

    render(
      <VelocityLoadedView
        periodLabel="Last 7 days"
        periodDays={7}
        data={data}
        sourceSplit={sourceSplit}
        workOrderFlow={{
          flow: {
            days: 7,
            label: "Last 7 days",
            sampleSize: 0,
            medianCycleHours: 0,
            medianRunningHours: 0,
            medianWaitingHours: 0,
            runningShareOfCyclePct: 0,
            waitingShareOfCyclePct: 0,
            timeTrend: [],
          },
        }}
      />,
    );

    const flow = screen.getByTestId("velocity-work-order-flow");
    expect(flow).toHaveTextContent("No work orders closed in this period.");
    // Metric cells (`Cycle time` label, `%` share hints) must not render.
    expect(flow).not.toHaveTextContent("From start to close");
    expect(flow).not.toHaveTextContent("% of cycle time");
  });

  it("renders cost cells when a cost config is provided (Storybook prototype)", () => {
    const data = toData();
    const sourceSplit: VelocitySourceSplitConfig = { hasPeopleCohort: true };

    render(
      <VelocityLoadedView
        periodLabel="Last 7 days"
        periodDays={7}
        data={data}
        sourceSplit={sourceSplit}
        cost={{
          yesterdayCostUsd: FACTORY_VELOCITY_YESTERDAY.costUsd,
          yesterdayTokens: FACTORY_VELOCITY_YESTERDAY.tokens,
          yesterdayCostPerMerged: FACTORY_VELOCITY_YESTERDAY.costPerMergedPr,
          totalCostUsd: FACTORY_VELOCITY_BY_PERIOD[7].totals.costUsd,
          seriesUsd: FACTORY_VELOCITY_BY_PERIOD[7].points.map((point) => point.costUsd),
        }}
      />,
    );

    const yesterday = screen.getByTestId("velocity-yesterday");
    expect(yesterday).toHaveTextContent("Cost");
    expect(yesterday).toHaveTextContent("Cost per merged PR");
  });
});
