import { describe, expect, it } from "bun:test";
import { toAutomationRun, toRunActivities } from "./runHistoryData";

describe("Implement run history data", () => {
  it("keeps the run outcome and exact task identity independent of the current task state", () => {
    const run = toAutomationRun(
      {
        id: "run-1",
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        createdAt: "2026-10-19T10:00:00Z",
        finishedAt: "2026-10-19T10:00:42Z",
      },
      [{ id: "task-1", key: "SP-42", title: "Fix retries", sourceRunId: "run-1" }],
      Date.now(),
    );
    expect(run.status).toBe("Failed");
    expect(run.task).toEqual({ id: "SP-42", title: "Fix retries" });
    expect(run.duration).toBe("42s");
  });
  it("does not invent a passed result or completion time for incomplete records", () => {
    const run = toAutomationRun({ id: "unknown", state: "STATE_FINISHED", result: "RESULT_UNKNOWN" }, [], Date.now());
    expect(run.status).toBe("Unknown");
    expect(run.duration).toBe("—");
    expect(run.task).toBeUndefined();
  });
  it("uses recorded execution IDs and historical step names, retaining retry attempts and outputs", () => {
    const executions = [
      {
        id: "retry",
        nodeId: "agent",
        runId: "run",
        createdAt: "2026-10-19T10:01:00Z",
        state: "STATE_FINISHED" as const,
        result: "RESULT_PASSED" as const,
        outputs: { default: [{ result: "done" }] },
        configuration: { additionalInstructions: "Instructions captured for this run" },
      },
      {
        id: "first",
        nodeId: "agent",
        runId: "run",
        createdAt: "2026-10-19T10:00:00Z",
        result: "RESULT_FAILED" as const,
        resultMessage: "Agent exited",
      },
      { id: "other", nodeId: "agent", runId: "different-run" },
    ];
    const activities = toRunActivities({ id: "run", executions: [{ id: "first" }, { id: "retry" }] }, executions, [
      {
        id: "agent",
        name: "Original implementation step",
        configuration: { additionalInstructions: "Different current instructions" },
      },
    ]);
    expect(activities.map((activity) => activity.id)).toEqual(["first", "retry"]);
    expect(activities[0].name).toBe("Original implementation step");
    expect(activities[0].error).toBe("Agent exited");
    expect(activities[1].outputs).toEqual({ default: [{ result: "done" }] });
    expect(activities[1].additionalInstructions).toBe("Instructions captured for this run");
    expect(activities[0].additionalInstructions).toBeUndefined();
  });
});
