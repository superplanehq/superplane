import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { describe, expect, it } from "vitest";

import {
  aggregateFactoryVelocityFlow,
  formatDurationHours,
  formatVelocityYesterdayLabel,
  pickVelocityChartUnit,
} from "./factoryVelocityFlow";

const HOUR = 60 * 60 * 1000;
const DAY = 24 * HOUR;
// Fixed anchor used across the file so window math is deterministic.
const NOW = new Date("2026-01-15T12:00:00").getTime();

function iso(offset: number): string {
  return new Date(NOW + offset).toISOString();
}

function execution(overrides: Partial<FactoriesWorkOrderExecution>): FactoriesWorkOrderExecution {
  return {
    id: "exec",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: iso(-2 * HOUR),
    finishedAt: iso(-1 * HOUR),
    ...overrides,
  };
}

/** Fixture-only shape: velocity math flattens dispatches, so tests list step
 * executions directly and `closedOrder` wraps them into one dispatch. */
type ClosedOrderOverrides = Partial<FactoriesWorkOrder> & {
  executions?: FactoriesWorkOrderExecution[];
};

function closedOrder(overrides: ClosedOrderOverrides): FactoriesWorkOrder {
  const { executions = [execution({})], ...rest } = overrides;
  return {
    id: rest.id ?? "order",
    state: "STATE_CLOSED",
    createdAt: iso(-3 * DAY),
    updatedAt: iso(-1 * DAY),
    lineDispatches: [{ id: `dispatch-${rest.id ?? "order"}`, stepExecutions: executions }],
    ...rest,
  };
}

describe("aggregateFactoryVelocityFlow", () => {
  it("returns zeroed medians when no closed orders exist", () => {
    const flow = aggregateFactoryVelocityFlow([], 14, NOW);

    expect(flow.days).toBe(14);
    expect(flow.label).toBe("Last 14 days");
    expect(flow.sampleSize).toBe(0);
    expect(flow.medianCycleHours).toBe(0);
    expect(flow.medianRunningHours).toBe(0);
    expect(flow.medianWaitingHours).toBe(0);
    expect(flow.runningShareOfCyclePct).toBe(0);
    expect(flow.waitingShareOfCyclePct).toBe(0);
    expect(flow.timeTrend).toHaveLength(14);
  });

  it("skips draft/open orders and orders without executions", () => {
    const orders: FactoriesWorkOrder[] = [
      closedOrder({ id: "no-exec", executions: [] }),
      { ...closedOrder({ id: "draft" }), state: "STATE_DRAFT" },
      { ...closedOrder({ id: "open" }), state: "STATE_OPEN" },
    ];

    const flow = aggregateFactoryVelocityFlow(orders, 14, NOW);

    expect(flow.sampleSize).toBe(0);
  });

  it("derives cycle, running and waiting from multiple executions", () => {
    // Two runs: first ran for 4h, then a 4h gap, then a 4h run.
    // Cycle end is the close instant (updatedAt, -2h), not the last run finish.
    // Cycle start = first exec (-16h). Cycle = 14h, running = 8h, waiting = 6h.
    const order = closedOrder({
      id: "multi",
      updatedAt: iso(-2 * HOUR),
      executions: [
        execution({
          id: "first",
          createdAt: iso(-16 * HOUR),
          finishedAt: iso(-12 * HOUR),
        }),
        execution({
          id: "second",
          createdAt: iso(-8 * HOUR),
          finishedAt: iso(-4 * HOUR),
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, NOW);

    expect(flow.sampleSize).toBe(1);
    expect(flow.medianCycleHours).toBeCloseTo(14, 5);
    expect(flow.medianRunningHours).toBeCloseTo(8, 5);
    expect(flow.medianWaitingHours).toBeCloseTo(6, 5);
    expect(flow.runningShareOfCyclePct).toBe(57);
    expect(flow.waitingShareOfCyclePct).toBe(43);
  });

  it("falls back to the last run finish when updatedAt is missing", () => {
    const order = closedOrder({
      id: "no-updated-at",
      updatedAt: undefined,
      executions: [
        execution({
          createdAt: iso(-16 * HOUR),
          finishedAt: iso(-6 * HOUR),
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, NOW);

    expect(flow.sampleSize).toBe(1);
    expect(flow.medianCycleHours).toBeCloseTo(10, 5);
    expect(flow.medianRunningHours).toBeCloseTo(10, 5);
    expect(flow.medianWaitingHours).toBeCloseTo(0, 5);
  });

  it("uses updatedAt when the order has no finished executions", () => {
    const order = closedOrder({
      id: "no-finish",
      updatedAt: iso(-6 * HOUR),
      executions: [
        execution({
          id: "pending",
          state: "STATE_PENDING",
          result: "RESULT_UNKNOWN",
          createdAt: iso(-16 * HOUR),
          finishedAt: undefined,
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, NOW);

    expect(flow.sampleSize).toBe(1);
    expect(flow.medianCycleHours).toBeCloseTo(10, 5);
    expect(flow.medianRunningHours).toBeCloseTo(0, 5);
    expect(flow.medianWaitingHours).toBeCloseTo(10, 5);
  });

  it("ignores non-finished executions when summing running time", () => {
    const start = -10 * HOUR;
    const finish = -6 * HOUR;

    const order = closedOrder({
      id: "with-pending",
      updatedAt: iso(-5 * HOUR),
      executions: [
        execution({ createdAt: iso(start), finishedAt: iso(finish) }),
        execution({
          id: "pending",
          state: "STATE_PENDING",
          createdAt: iso(-4 * HOUR),
          finishedAt: undefined,
          result: "RESULT_UNKNOWN",
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, NOW);

    expect(flow.medianRunningHours).toBeCloseTo(4, 5);
  });

  it("includes a closed order whose last run finished before the period", () => {
    // Last run finished 20 days ago; the task closed yesterday.
    // Windowing must use the close instant, not the last run finish.
    const order = closedOrder({
      id: "closed-later",
      updatedAt: iso(-1 * DAY),
      executions: [
        execution({
          createdAt: iso(-20 * DAY - 4 * HOUR),
          finishedAt: iso(-20 * DAY),
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, NOW);

    expect(flow.sampleSize).toBe(1);
  });

  it("buckets a sample on the close day, not the last run finish day", () => {
    const close = new Date(2026, 0, 15, 9, 0, 0);
    const now = new Date(2026, 0, 15, 12, 0, 0).getTime();
    const order = closedOrder({
      id: "late-close",
      updatedAt: close.toISOString(),
      executions: [
        execution({
          createdAt: new Date(2026, 0, 14, 9, 0, 0).toISOString(),
          finishedAt: new Date(2026, 0, 14, 11, 0, 0).toISOString(),
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, now);

    expect(flow.sampleSize).toBe(1);
    const lastBucket = flow.timeTrend[flow.timeTrend.length - 1];
    expect(lastBucket.runningHours).toBeGreaterThan(0);
    const earlierBuckets = flow.timeTrend.slice(0, -1);
    expect(earlierBuckets.every((point) => point.runningHours === 0 && point.waitingHours === 0)).toBe(true);
  });

  it("excludes orders that closed outside the window", () => {
    const orders: FactoriesWorkOrder[] = [
      closedOrder({
        id: "too-old",
        updatedAt: iso(-20 * DAY),
        executions: [
          execution({
            createdAt: iso(-20 * DAY - 4 * HOUR),
            finishedAt: iso(-20 * DAY),
          }),
        ],
      }),
      closedOrder({
        id: "in-window",
        updatedAt: iso(-2 * DAY),
        executions: [
          execution({
            createdAt: iso(-2 * DAY - 4 * HOUR),
            finishedAt: iso(-2 * DAY),
          }),
        ],
      }),
    ];

    const flow = aggregateFactoryVelocityFlow(orders, 14, NOW);

    expect(flow.sampleSize).toBe(1);
  });

  it("emits a bucket per day and groups samples on their local close day", () => {
    const orders: FactoriesWorkOrder[] = [
      closedOrder({
        id: "yesterday-a",
        updatedAt: iso(-1 * DAY),
        executions: [execution({ createdAt: iso(-1 * DAY - 4 * HOUR), finishedAt: iso(-1 * DAY - 1 * HOUR) })],
      }),
      closedOrder({
        id: "yesterday-b",
        updatedAt: iso(-1 * DAY),
        executions: [execution({ createdAt: iso(-1 * DAY - 8 * HOUR), finishedAt: iso(-1 * DAY - 3 * HOUR) })],
      }),
    ];

    const flow = aggregateFactoryVelocityFlow(orders, 14, NOW);

    expect(flow.timeTrend).toHaveLength(14);
    // Two samples on the same day = one bucket with two entries;
    // other buckets stay at 0.
    const nonZeroBuckets = flow.timeTrend.filter((point) => point.runningHours > 0 || point.waitingHours > 0);
    expect(nonZeroBuckets).toHaveLength(1);
  });

  it("names the weekday and the date instead of numbering the days", () => {
    // NOW is Thursday 15 January 2026, so a fortnight starts on Friday 2 January.
    const flow = aggregateFactoryVelocityFlow([], 14, NOW);

    expect(flow.timeTrend[0].day).toBe("Fri 2");
    expect(flow.timeTrend[1].day).toBe("Sat 3");
    expect(flow.timeTrend[13].day).toBe("Thu 15");
  });

  it("repeats the month when the window crosses into a new one", () => {
    // A fortnight to 5 February 2026 starts on 23 January.
    const now = new Date(2026, 1, 5, 12, 0, 0).getTime();

    const flow = aggregateFactoryVelocityFlow([], 14, now);

    expect(flow.timeTrend[0].day).toBe("Fri 23");
    expect(flow.timeTrend[9].day).toBe("Sun Feb 1");
    expect(flow.timeTrend[10].day).toBe("Mon 2");
  });

  it("labels every fifth day on a month so the ticks stay legible", () => {
    const flow = aggregateFactoryVelocityFlow([], 30, NOW);

    const labelled = flow.timeTrend.filter((point) => point.day !== "");

    expect(labelled).toHaveLength(7);
  });

  it("supports the 30-day period label", () => {
    const flow = aggregateFactoryVelocityFlow([], 30, NOW);

    expect(flow.days).toBe(30);
    expect(flow.label).toBe("Last 30 days");
    expect(flow.timeTrend).toHaveLength(30);
  });

  it("keeps a close on its local calendar day across a short DST day", () => {
    // 8 March 2026 is the US spring-forward. Bucket keys must follow
    // local calendar midnights, not fixed 24-hour offsets.
    const now = new Date(2026, 2, 9, 12, 0, 0).getTime();
    const close = new Date(2026, 2, 7, 16, 0, 0);
    const start = new Date(2026, 2, 7, 10, 0, 0);
    const order = closedOrder({
      id: "dst",
      updatedAt: close.toISOString(),
      executions: [
        execution({
          createdAt: start.toISOString(),
          finishedAt: close.toISOString(),
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 14, now);

    expect(flow.sampleSize).toBe(1);
    const nonZeroBuckets = flow.timeTrend.filter((point) => point.runningHours > 0 || point.waitingHours > 0);
    expect(nonZeroBuckets).toHaveLength(1);
  });
});

describe("formatDurationHours", () => {
  it("formats sub-hour values in minutes", () => {
    expect(formatDurationHours(0.2)).toBe("12m");
    expect(formatDurationHours(1 / 60)).toBe("1m");
  });

  it("treats genuine zero and defensive inputs as 0h", () => {
    expect(formatDurationHours(0)).toBe("0h");
    expect(formatDurationHours(-5)).toBe("0h");
    expect(formatDurationHours(NaN)).toBe("0h");
  });

  it("guards the minute rollover so a value just under 1h renders as 1h, not 60m", () => {
    expect(formatDurationHours(0.999)).toBe("1h");
  });

  it("formats in-range hours as whole hours", () => {
    expect(formatDurationHours(1)).toBe("1h");
    expect(formatDurationHours(5)).toBe("5h");
  });

  it("formats multi-day durations in days", () => {
    expect(formatDurationHours(48)).toBe("2d");
    expect(formatDurationHours(60)).toBe("2.5d");
  });
});

describe("pickVelocityChartUnit", () => {
  it("picks minutes when every value in the period is sub-hour", () => {
    const chartUnit = pickVelocityChartUnit([0.1, 0.2, 0.3]);

    expect(chartUnit.unit).toBe("m");
    expect(chartUnit.formatTick(0.2)).toBe("12m");
    expect(chartUnit.formatTick(0.05)).toBe("3m");
  });

  it("picks hours when the largest value in the period is in hour range", () => {
    const chartUnit = pickVelocityChartUnit([0.2, 5, 20]);

    expect(chartUnit.unit).toBe("h");
    expect(chartUnit.formatTick(0.2)).toBe("0h");
    expect(chartUnit.formatTick(5)).toBe("5h");
  });

  it("picks days when the largest value in the period is multi-day", () => {
    const chartUnit = pickVelocityChartUnit([5, 60, 72]);

    expect(chartUnit.unit).toBe("d");
    expect(chartUnit.formatTick(5)).toBe("0.2d");
    expect(chartUnit.formatTick(72)).toBe("3d");
  });

  it("falls back to hours without throwing for empty or all-zero input", () => {
    expect(pickVelocityChartUnit([]).unit).toBe("h");
    expect(pickVelocityChartUnit([0, 0, 0]).unit).toBe("h");
    expect(pickVelocityChartUnit([]).formatTick(0)).toBe("0h");
  });
});

describe("formatVelocityYesterdayLabel", () => {
  it("formats the UTC-noon calendar stamp so the day does not shift by timezone", () => {
    const label = formatVelocityYesterdayLabel("2026-08-17T12:00:00.000Z");
    expect(label).toMatch(/^Yesterday · /);
    expect(label).toMatch(/17/);
    expect(label).not.toMatch(/16/);
  });
});
