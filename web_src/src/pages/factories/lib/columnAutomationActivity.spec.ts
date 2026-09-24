import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrder } from "@/api-client";

import { buildColumnAutomationActivity, emptyColumnAutomationActivity } from "./columnAutomationActivity";

const NOW = new Date("2026-09-08T19:00:00.000Z");

function isoHoursAgo(hours: number): string {
  return new Date(NOW.getTime() - hours * 60 * 60 * 1000).toISOString();
}

function orderWithExecutions(
  executions: Array<{
    appId: string;
    state?: "STATE_STARTED" | "STATE_FINISHED";
    result?: "RESULT_PASSED" | "RESULT_FAILED" | "RESULT_CANCELLED";
    at?: string;
  }>,
): FactoriesWorkOrder {
  return {
    id: "wo-activity",
    lineDispatches: [
      {
        stepExecutions: executions.map((execution, index) => ({
          id: `exec-${index}`,
          state: execution.state ?? "STATE_FINISHED",
          result: execution.result,
          finishedAt: execution.at,
          updatedAt: execution.at,
          createdAt: execution.at,
          run: { appId: execution.appId },
        })),
      },
    ],
  };
}

describe("buildColumnAutomationActivity", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns only the running count when the automation has no canvas", () => {
    expect(
      buildColumnAutomationActivity({
        workOrders: [orderWithExecutions([{ appId: "app-1", result: "RESULT_PASSED", at: isoHoursAgo(1) }])],
        runningCount: 2,
      }),
    ).toEqual(emptyColumnAutomationActivity(2));
  });

  it("uses the newest finished run for last-run status", () => {
    const activity = buildColumnAutomationActivity({
      canvasId: "app-implement",
      runningCount: 1,
      workOrders: [
        orderWithExecutions([
          { appId: "app-implement", result: "RESULT_PASSED", at: isoHoursAgo(3) },
          { appId: "app-implement", result: "RESULT_FAILED", at: isoHoursAgo(0.5) },
          { appId: "app-other", result: "RESULT_PASSED", at: isoHoursAgo(0.1) },
          { appId: "app-implement", state: "STATE_STARTED" },
        ]),
      ],
    });

    expect(activity).toEqual({
      lastRunStatus: "failed",
      lastRunWhen: "30 minutes ago",
      runningCount: 1,
    });
  });
});
