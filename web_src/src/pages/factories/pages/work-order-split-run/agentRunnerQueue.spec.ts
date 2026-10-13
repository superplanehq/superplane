import { describe, expect, it } from "bun:test";

import { agentRunnerQueue } from "./agentRunnerQueue";

function agentCanvas(component = "runnerOpenRouter") {
  return {
    spec: {
      nodes: [
        { id: "trigger", name: "On run", type: "TYPE_TRIGGER" as const, component: "onRun" },
        { id: "agent", name: "Capture screenshots", type: "TYPE_ACTION" as const, component },
        { id: "access", name: "Update activity", type: "TYPE_ACTION" as const, component: "updatePullRequestActivity" },
      ],
    },
  };
}

function startedRun(overrides: Record<string, unknown> = {}) {
  return {
    state: "STATE_STARTED" as const,
    executions: [{ nodeId: "trigger", state: "STATE_FINISHED" as const, result: "RESULT_PASSED" as const }],
    queueItems: [{ id: "queue-1", nodeId: "agent" }],
    ...overrides,
  };
}

describe("agentRunnerQueue", () => {
  it("shows an active run as queued when the agent has a queue item and no active execution", () => {
    const queue = agentRunnerQueue(agentCanvas("runnerClaudeCode"), startedRun());

    expect(queue).toEqual({ queued: true });
    expect(queue.workStartedAt).toBeUndefined();
  });

  it("does not treat a finished agent execution as working", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          executions: [
            { nodeId: "trigger", state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { nodeId: "agent", state: "STATE_FINISHED", result: "RESULT_PASSED" },
          ],
        }),
      ),
    ).toEqual({ queued: true });
  });

  it("shows the run as working when the agent execution has started", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          executions: [
            { nodeId: "trigger", state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { nodeId: "agent", state: "STATE_STARTED", createdAt: "2026-10-13T11:10:00.000Z" },
          ],
        }),
      ),
    ).toEqual({ queued: false, workStartedAt: "2026-10-13T11:10:00.000Z" });
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          executions: [
            { nodeId: "trigger", state: "STATE_FINISHED", result: "RESULT_PASSED" },
            { nodeId: "agent", state: "STATE_STARTED", createdAt: "2026-10-13T11:10:00.000Z" },
          ],
        }),
      ).workEndedAt,
    ).toBeUndefined();
  });

  it("shows a pending agent execution as queued before it starts", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          queueItems: [],
          executions: [{ nodeId: "agent", state: "STATE_PENDING", createdAt: "2026-10-13T11:10:00.000Z" }],
        }),
      ),
    ).toEqual({ queued: true });
  });

  it("does not treat a cancelling agent execution as queued", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          executions: [{ nodeId: "agent", state: "STATE_CANCELLING", createdAt: "2026-10-13T11:10:00.000Z" }],
        }),
      ).queued,
    ).toBe(false);
  });

  it("does not treat a run reference without queue items as queued", () => {
    const { queueItems: _queueItems, ...runRef } = startedRun();

    expect(agentRunnerQueue(agentCanvas(), runRef).queued).toBe(false);
  });

  it("does not treat a queue item on a non-agent component as queued", () => {
    expect(
      agentRunnerQueue(agentCanvas(), startedRun({ queueItems: [{ id: "queue-access", nodeId: "access" }] })).queued,
    ).toBe(false);
  });

  it("does not mark a finished run queued", () => {
    expect(agentRunnerQueue(agentCanvas(), startedRun({ state: "STATE_FINISHED" })).queued).toBe(false);
  });

  it("keeps the agent working interval after the execution finishes", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          queueItems: [],
          executions: [
            {
              nodeId: "agent",
              state: "STATE_FINISHED",
              createdAt: "2026-10-13T11:10:00.000Z",
              updatedAt: "2026-10-13T11:12:00.000Z",
            },
          ],
        }),
      ),
    ).toEqual({
      queued: false,
      workStartedAt: "2026-10-13T11:10:00.000Z",
      workEndedAt: "2026-10-13T11:12:00.000Z",
    });
  });

  it("sums each agent execution and skips the wait between them", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          queueItems: [],
          executions: [
            {
              nodeId: "access",
              state: "STATE_FINISHED",
              createdAt: "2026-10-13T11:02:00.000Z",
              updatedAt: "2026-10-13T11:40:00.000Z",
            },
            {
              nodeId: "agent",
              state: "STATE_FINISHED",
              createdAt: "2026-10-13T11:00:00.000Z",
              updatedAt: "2026-10-13T11:02:00.000Z",
            },
            { nodeId: "agent", state: "STATE_STARTED", createdAt: "2026-10-13T11:40:00.000Z" },
          ],
        }),
      ),
    ).toEqual({
      queued: false,
      workStartedAt: "2026-10-13T11:40:00.000Z",
      completedWorkMs: 2 * 60 * 1000,
    });
  });

  it("adds each open agent execution instead of the earliest start", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          queueItems: [],
          executions: [
            { nodeId: "agent", state: "STATE_STARTED", createdAt: "2026-10-13T11:58:00.000Z" },
            { nodeId: "agent", state: "STATE_STARTED", createdAt: "2026-10-13T11:59:00.000Z" },
          ],
        }),
      ),
    ).toEqual({
      queued: false,
      workStartedAt: "2026-10-13T11:58:00.000Z",
      openStartedAt: ["2026-10-13T11:58:00.000Z", "2026-10-13T11:59:00.000Z"],
    });
  });

  it("sums finished agent executions instead of the span that includes the wait", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          state: "STATE_FINISHED",
          finishedAt: "2026-10-13T12:00:00.000Z",
          queueItems: [],
          executions: [
            {
              nodeId: "agent",
              state: "STATE_FINISHED",
              createdAt: "2026-10-13T11:00:00.000Z",
              updatedAt: "2026-10-13T11:02:00.000Z",
            },
            {
              nodeId: "agent",
              state: "STATE_FINISHED",
              createdAt: "2026-10-13T11:40:00.000Z",
              updatedAt: "2026-10-13T11:43:00.000Z",
            },
          ],
        }),
      ),
    ).toEqual({
      queued: false,
      completedWorkMs: 5 * 60 * 1000,
      workEndedAt: "2026-10-13T11:43:00.000Z",
    });
  });

  it("keeps the agent working interval after the run finishes", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          state: "STATE_FINISHED",
          finishedAt: "2026-10-13T12:00:00.000Z",
          queueItems: [],
          executions: [
            {
              nodeId: "agent",
              state: "STATE_FINISHED",
              createdAt: "2026-10-13T11:10:00.000Z",
              updatedAt: "2026-10-13T11:12:00.000Z",
            },
          ],
        }),
      ),
    ).toEqual({
      queued: false,
      workStartedAt: "2026-10-13T11:10:00.000Z",
      workEndedAt: "2026-10-13T11:12:00.000Z",
    });
  });
});
