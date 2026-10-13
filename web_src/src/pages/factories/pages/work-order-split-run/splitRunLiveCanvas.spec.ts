import { describe, expect, it } from "bun:test";

import { RUNNING_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { simpleFactoryRunCanvasSpec, simpleFactoryRunExecutions } from "../../__fixtures__/simpleFactoryRunCanvas";
import { intakeTicketAnalysisFixture } from "../lineIntakeModel";
import { agentRunnerQueue } from "./agentRunnerQueue";
import { emptySplitRunCanvas, splitRunCanvasForPhase } from "./splitRunCanvases";
import { clockLabel } from "./splitRunFormat";
import {
  metricFromExecution,
  nodeStatusFromExecution,
  orderCanvasNodesTopologically,
  resolveSplitRunVisual,
  splitRunCanvasFromLive,
  streamFromLiveRun,
  streamStatusFromNode,
} from "./splitRunLiveCanvas";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "./splitRunMocks";

const LIVE_CANVAS = {
  metadata: { id: "app-refund-implementer", name: "Refund Implementer" },
  spec: simpleFactoryRunCanvasSpec(),
};

describe("nodeStatusFromExecution", () => {
  it("maps run execution state onto factory node chrome", () => {
    expect(nodeStatusFromExecution(undefined, false)).toBe("did_not_run");
    expect(nodeStatusFromExecution({ state: "STATE_STARTED" }, false)).toBe("running");
    expect(nodeStatusFromExecution({ state: "STATE_FINISHED", result: "RESULT_PASSED" }, true)).toBe("triggered");
    expect(nodeStatusFromExecution({ state: "STATE_FINISHED", result: "RESULT_FAILED" }, false)).toBe("failed");
    expect(nodeStatusFromExecution({ state: "STATE_FINISHED", result: "RESULT_CANCELLED" }, false)).toBe("cancelled");
  });
});

describe("streamStatusFromNode", () => {
  it("keeps a cancelled node as canceled, not pending", () => {
    expect(streamStatusFromNode("cancelled")).toBe("cancelled");
  });
});

describe("metricFromExecution", () => {
  it("formats the execution window", () => {
    expect(metricFromExecution(undefined)).toBe("—");
    expect(
      metricFromExecution({
        createdAt: "2026-08-21T12:00:00.000Z",
        updatedAt: "2026-08-21T12:00:12.000Z",
      }),
    ).toBe("12s");
  });
});

describe("splitRunCanvasFromLive", () => {
  it("paints the factory canvas from the automation run", () => {
    const model = splitRunCanvasFromLive({
      canvas: LIVE_CANVAS,
      run: { executions: simpleFactoryRunExecutions() },
      fallbackTitle: "Implement",
    });

    expect(model?.title).toBe("Refund Implementer");
    expect(model?.nodes.map((node) => node.id)).toContain("run-workflow");
    expect(model?.statuses["run-workflow"]).toBe("passed");
    expect(model?.statuses["run-claude"]).toBe("did_not_run");
    expect(model?.metrics["run-claude"]).toBe("—");
  });
});

describe("streamFromLiveRun", () => {
  it("writes one log line per canvas node from the run", () => {
    const stream = streamFromLiveRun(LIVE_CANVAS, { executions: simpleFactoryRunExecutions() });
    const workflow = stream.find((line) => line.nodeId === "run-workflow");
    const claude = stream.find((line) => line.nodeId === "run-claude");

    expect(stream.map((line) => line.nodeId)).toContain("on-run");
    expect(workflow?.status).toBe("passed");
    expect(workflow?.component).toBe("semaphore.runWorkflow");
    expect(workflow?.executionId).toBe("exec-run-workflow");
    expect(claude?.status).toBe("pending");
    expect(claude?.component).toBe("runnerClaudeCode");
    expect(claude?.executionId).toBeUndefined();
  });

  it("puts the node error on the stream line", () => {
    const stream = streamFromLiveRun(
      {
        spec: {
          nodes: [{ id: "broken-node", name: "Broken node", type: "TYPE_ACTION", component: "http.request" }],
        },
      },
      {
        executions: [
          {
            id: "exec-broken",
            nodeId: "broken-node",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            resultMessage: "unknown name asdad (1:1)\n | asdad\n | ^",
          },
        ],
      },
    );

    expect(stream.find((line) => line.nodeId === "broken-node")).toMatchObject({
      status: "failed",
      detail: "unknown name asdad (1:1)\n | asdad\n | ^",
    });
  });

  it("orders log lines by canvas topology, not execution time", () => {
    const stream = streamFromLiveRun(
      {
        spec: {
          nodes: [
            { id: "random-noop", name: "random noop", type: "TYPE_ACTION" },
            { id: "on-run", name: "onRun", type: "TYPE_TRIGGER", component: "onRun" },
            { id: "noop-2", name: "noop 2", type: "TYPE_ACTION" },
            { id: "noop", name: "noop", type: "TYPE_ACTION" },
          ],
          edges: [
            { sourceId: "on-run", targetId: "noop" },
            { sourceId: "noop", targetId: "noop-2" },
            { sourceId: "noop-2", targetId: "random-noop" },
          ],
        },
      },
      {
        createdAt: "2026-08-21T13:22:50.000Z",
        rootEvent: { nodeId: "on-run", createdAt: "2026-08-21T13:22:50.000Z" },
        executions: [
          {
            nodeId: "random-noop",
            state: "STATE_FINISHED" as const,
            result: "RESULT_PASSED" as const,
            createdAt: "2026-08-21T13:22:56.000Z",
          },
          {
            nodeId: "noop",
            state: "STATE_FINISHED" as const,
            result: "RESULT_PASSED" as const,
            createdAt: "2026-08-21T13:22:58.000Z",
          },
          {
            nodeId: "noop-2",
            state: "STATE_FINISHED" as const,
            result: "RESULT_PASSED" as const,
            createdAt: "2026-08-21T13:22:57.000Z",
          },
        ],
      },
    );

    expect(stream.map((line) => line.componentName)).toEqual(["onRun", "noop", "noop 2", "random noop"]);
    expect(stream[0]?.at).toBe(clockLabel("2026-08-21T13:22:50.000Z"));
    expect(stream[0]?.status).toBe("passed");
  });
});

describe("orderCanvasNodesTopologically", () => {
  it("puts triggers first and follows edges", () => {
    const ordered = orderCanvasNodesTopologically(
      [
        { id: "b", type: "TYPE_ACTION" },
        { id: "a", type: "TYPE_TRIGGER" },
        { id: "c", type: "TYPE_ACTION" },
      ],
      [
        { sourceId: "a", targetId: "b" },
        { sourceId: "b", targetId: "c" },
      ],
    );

    expect(ordered.map((node) => node.id)).toEqual(["a", "b", "c"]);
  });

  it("keeps spec order among siblings and after a cycle", () => {
    const ordered = orderCanvasNodesTopologically(
      [
        { id: "loop", type: "TYPE_ACTION" },
        { id: "start", type: "TYPE_TRIGGER" },
        { id: "again", type: "TYPE_ACTION" },
      ],
      [
        { sourceId: "start", targetId: "loop" },
        { sourceId: "loop", targetId: "again" },
        { sourceId: "again", targetId: "loop" },
      ],
    );

    expect(ordered.map((node) => node.id)).toEqual(["start", "loop", "again"]);
  });
});

describe("resolveSplitRunVisual", () => {
  it("keeps the line automation when the live canvas is a different graph", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    expect(implement).toBeDefined();
    const liveCanvas = splitRunCanvasFromLive({
      canvas: LIVE_CANVAS,
      run: { executions: simpleFactoryRunExecutions() },
      fallbackTitle: "Implement",
    });
    const liveStream = streamFromLiveRun(LIVE_CANVAS, { executions: simpleFactoryRunExecutions() });
    const visual = resolveSplitRunVisual(implement!, {
      enabled: true,
      canvas: liveCanvas,
      stream: liveStream,
    });

    expect(visual.canvas.title).toBe("Implement");
    expect(visual.stream?.some((line) => line.componentName === "Implement From Task Description")).toBe(true);
    expect(visual.stream?.some((line) => line.nodeId === "run-workflow")).toBe(false);
  });

  it("uses the live canvas for a custom app when demo artifacts are off", () => {
    const phase = {
      id: "build-0",
      name: "Build",
      status: "running" as const,
      duration: "1s",
      componentName: "Wide chain",
      artifacts: [],
      stream: [],
      canvasSteps: [],
      appId: "app-custom",
      runId: "run-wide",
    };
    const liveCanvas = splitRunCanvasFromLive({
      canvas: {
        metadata: { id: "app-custom", name: "Wide chain" },
        spec: {
          nodes: [
            { id: "kickoff", name: "Kickoff", type: "TYPE_TRIGGER" },
            { id: "step-11", name: "Step 11", type: "TYPE_ACTION" },
          ],
          edges: [{ sourceId: "kickoff", targetId: "step-11" }],
        },
      },
      fallbackTitle: "Wide chain",
    });
    const visual = resolveSplitRunVisual(
      phase,
      { enabled: true, canvas: liveCanvas, stream: [] },
      { demoArtifacts: false },
    );

    expect(visual.canvas.nodes.map((node) => node.id)).toEqual(["kickoff", "step-11"]);
  });

  it("uses the live canvas when it is the same line automation", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const yaml = splitRunCanvasForPhase(implement!);
    const visual = resolveSplitRunVisual(implement!, {
      enabled: true,
      canvas: { ...yaml, title: "Implementation (live)" },
      stream: [
        {
          id: "live-implementation-agent",
          nodeId: "implementation-agent-no-issue",
          at: "00:00:01",
          componentName: "Implement From Task Description",
          status: "passed",
        },
      ],
    });

    expect(visual.canvas.title).toBe("Implementation (live)");
    expect(visual.stream?.some((line) => line.id === "live-implementation-agent")).toBe(true);
  });

  it("replaces the canned implement branch and adds the order pull request", () => {
    const implement = splitRunFixtureForWorkOrder(RUNNING_WORK_ORDER).phases.find(
      (phase) => phase.id === "implement-0",
    );
    const visual = resolveSplitRunVisual(implement!, { enabled: false, stream: [] });
    const branches = (visual.stream ?? [])
      .map((line) => line.artifact)
      .filter((artifact) => artifact?.type === "TYPE_BRANCH")
      .map((artifact) => artifact?.data?.name);
    const pullRequests = (visual.stream ?? []).filter((line) => line.pullRequest);

    expect(branches).toEqual(["feature/rf-103"]);
    expect(pullRequests).toHaveLength(1);
    expect(pullRequests[0]?.pullRequest).toMatchObject({ number: "503" });
  });

  it("does not copy demo YAML artifacts onto a live stream", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const yaml = splitRunCanvasForPhase(implement!);
    const visual = resolveSplitRunVisual(
      implement!,
      {
        enabled: true,
        canvas: yaml,
        stream: [
          {
            id: "add-branch-artifact",
            nodeId: "add-branch-artifact",
            at: "00:00:01",
            componentName: "Add Branch Artifact",
            status: "passed",
          },
        ],
      },
      { demoArtifacts: false },
    );

    expect(visual.stream?.some((line) => line.artifact)).toBe(false);
  });

  it("keeps the YAML canvas when no live run is wired", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const visual = resolveSplitRunVisual(implement!, { enabled: false, stream: [] });

    expect(visual.canvas.title).toBe("Implement");
    expect(visual.stream?.some((line) => line.componentName === "Implement From Task Description")).toBe(true);
  });

  it("shows the line automation while the live canvas loads", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const visual = resolveSplitRunVisual(implement!, { enabled: true, stream: [] });

    expect(visual.canvas.title).toBe("Implement");
    expect(visual.stream?.some((line) => line.componentName === "Implement From Task Description")).toBe(true);
  });

  it("does not keep YAML logs when the live canvas fails", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const visual = resolveSplitRunVisual(implement!, { enabled: true, isError: true, stream: [] });

    expect(visual.canvas).toEqual(emptySplitRunCanvas(implement!));
    expect(visual.stream).toEqual([]);
  });

  it("writes the Score check onto the report line", () => {
    const fixture = intakeTicketAnalysisFixture(
      {
        id: "wo-review-pay-842",
        title: "Add retry handling to webhook delivery",
        confidenceScore: 5,
      },
      { complete: true },
    );
    const score = fixture.phases.find((phase) => phase.id === "score");
    expect(score).toBeDefined();
    const visual = resolveSplitRunVisual(score!, { enabled: false, stream: [] });

    expect(visual.stream?.find((line) => line.nodeId === "ticket-score")).toMatchObject({
      kind: "check",
      action: "5/5",
    });
  });

  it("keeps the manual create log when there is no automation canvas", () => {
    const backlog = {
      id: "backlog",
      name: "Backlog",
      status: "passed" as const,
      duration: "2s",
      componentName: "Created manually",
      artifacts: [],
      stream: [
        {
          id: "backlog-created",
          at: "12:24:02",
          componentName: "Leonardo DiCaprio created this task.",
          status: "passed" as const,
        },
      ],
      canvasSteps: [],
      canvasKey: null,
    };
    const visual = resolveSplitRunVisual(backlog, { enabled: false, stream: [] });

    expect(visual.canvas.nodes).toEqual([]);
    expect(visual.stream).toEqual(backlog.stream);
  });
});

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

  it("does not treat a pending agent execution as queued", () => {
    expect(
      agentRunnerQueue(
        agentCanvas(),
        startedRun({
          executions: [{ nodeId: "agent", state: "STATE_PENDING", createdAt: "2026-10-13T11:10:00.000Z" }],
        }),
      ),
    ).toEqual({ queued: false });
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
