import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderEvent } from "@/api-client";

import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";

function stepExecutionEvent(
  type: "step.execution.created" | "step.execution.finished",
  timestamp: string,
  runState: string,
  runResult?: string,
): FactoriesWorkOrderEvent {
  return {
    timestamp,
    type,
    event: {
      stepName: "Build",
      line: { id: "line-1", name: "CI" },
      run: { id: "run-1", state: runState, result: runResult },
      app: { id: "app-1" },
    },
  };
}

describe("buildWorkOrderTimelineViewFromEvents", () => {
  it("keeps finished step state when created and finished share a timestamp", () => {
    const timestamp = "2026-08-04T12:00:00.000Z";
    const apiEvents = [
      stepExecutionEvent("step.execution.finished", timestamp, "finished", "passed"),
      stepExecutionEvent("step.execution.created", timestamp, "pending"),
    ];

    const view = buildWorkOrderTimelineViewFromEvents(apiEvents);
    const step = view.events[0]?.steps?.[0];

    expect(step?.finishedAt).toBe(timestamp);
    expect(step?.execution?.state).toBe("STATE_FINISHED");
    expect(step?.execution?.result).toBe("RESULT_PASSED");
  });
});
