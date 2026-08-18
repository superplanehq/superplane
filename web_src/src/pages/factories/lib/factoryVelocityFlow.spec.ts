import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { describe, expect, it } from "vitest";

import { aggregateFactoryVelocityFlow, formatVelocityYesterdayLabel } from "./factoryVelocityFlow";

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

function closedOrder(overrides: Partial<FactoriesWorkOrder>): FactoriesWorkOrder {
  return {
    id: overrides.id ?? "order",
    state: "STATE_CLOSED",
    createdAt: iso(-3 * DAY),
    updatedAt: iso(-1 * DAY),
    executions: [execution({})],
    ...overrides,
  };
}

describe("aggregateFactoryVelocityFlow", () => {
  it("returns zeroed medians when no closed orders exist", () => {
    const flow = aggregateFactoryVelocityFlow([], 7, NOW);

    expect(flow.days).toBe(7);
    expect(flow.label).toBe("Last 7 days");
    expect(flow.sampleSize).toBe(0);
    expect(flow.medianCycleHours).toBe(0);
    expect(flow.medianRunningHours).toBe(0);
    expect(flow.medianWaitingHours).toBe(0);
    expect(flow.runningShareOfCyclePct).toBe(0);
    expect(flow.waitingShareOfCyclePct).toBe(0);
    expect(flow.timeTrend).toHaveLength(7);
  });

  it("skips draft/open orders and orders without executions", () => {
    const orders: FactoriesWorkOrder[] = [
      { ...closedOrder({ id: "no-exec" }), executions: [] },
      { ...closedOrder({ id: "draft" }), state: "STATE_DRAFT" },
      { ...closedOrder({ id: "open" }), state: "STATE_OPEN" },
    ];

    const flow = aggregateFactoryVelocityFlow(orders, 7, NOW);

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

    const flow = aggregateFactoryVelocityFlow([order], 7, NOW);

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

    const flow = aggregateFactoryVelocityFlow([order], 7, NOW);

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

    const flow = aggregateFactoryVelocityFlow([order], 7, NOW);

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

    const flow = aggregateFactoryVelocityFlow([order], 7, NOW);

    expect(flow.medianRunningHours).toBeCloseTo(4, 5);
  });

  it("includes a closed order whose last run finished before the period", () => {
    // Last run finished 10 days ago; the work order closed yesterday.
    // Windowing must use the close instant, not the last run finish.
    const order = closedOrder({
      id: "closed-later",
      updatedAt: iso(-1 * DAY),
      executions: [
        execution({
          createdAt: iso(-10 * DAY - 4 * HOUR),
          finishedAt: iso(-10 * DAY),
        }),
      ],
    });

    const flow = aggregateFactoryVelocityFlow([order], 7, NOW);

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

    const flow = aggregateFactoryVelocityFlow([order], 7, now);

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
        updatedAt: iso(-10 * DAY),
        executions: [
          execution({
            createdAt: iso(-10 * DAY - 4 * HOUR),
            finishedAt: iso(-10 * DAY),
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

    const flow = aggregateFactoryVelocityFlow(orders, 7, NOW);

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

    const flow = aggregateFactoryVelocityFlow(orders, 7, NOW);

    expect(flow.timeTrend).toHaveLength(7);
    // Two samples on the same day = one bucket with two entries;
    // other buckets stay at 0.
    const nonZeroBuckets = flow.timeTrend.filter((point) => point.runningHours > 0 || point.waitingHours > 0);
    expect(nonZeroBuckets).toHaveLength(1);
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

    const flow = aggregateFactoryVelocityFlow([order], 7, now);

    expect(flow.sampleSize).toBe(1);
    const nonZeroBuckets = flow.timeTrend.filter((point) => point.runningHours > 0 || point.waitingHours > 0);
    expect(nonZeroBuckets).toHaveLength(1);
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
