import { describe, expect, it } from "vitest";

import { simpleFactoryRunCanvasSpec, simpleFactoryRunExecutions } from "../../__fixtures__/simpleFactoryRunCanvas";
import { emptySplitRunCanvas } from "./splitRunCanvases";
import { clockLabel } from "./splitRunFormat";
import {
  metricFromExecution,
  nodeStatusFromExecution,
  resolveSplitRunVisual,
  splitRunCanvasFromLive,
  streamFromLiveRun,
} from "./splitRunLiveCanvas";
import { SPLIT_RUN_RUNNING } from "./splitRunMocks";

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
    expect(claude?.status).toBe("pending");
  });

  it("orders log lines by createdAt and stamps the trigger from the root event", () => {
    const stream = streamFromLiveRun(
      {
        spec: {
          nodes: [
            { id: "random-noop", name: "random noop", type: "TYPE_ACTION" },
            { id: "on-run", name: "onRun", type: "TYPE_TRIGGER", component: "onRun" },
            { id: "noop-2", name: "noop 2", type: "TYPE_ACTION" },
            { id: "noop", name: "noop", type: "TYPE_ACTION" },
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

    expect(stream.map((line) => line.componentName)).toEqual(["onRun", "random noop", "noop 2", "noop"]);
    expect(stream[0]?.at).toBe(clockLabel("2026-08-21T13:22:50.000Z"));
    expect(stream[0]?.status).toBe("passed");
  });
});

describe("resolveSplitRunVisual", () => {
  it("uses the live canvas when the phase has a run", () => {
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

    expect(visual.canvas.title).toBe("Refund Implementer");
    expect(visual.stream?.some((line) => line.nodeId === "run-workflow")).toBe(true);
    expect(visual.stream?.some((line) => line.componentName === "Create Branch")).toBe(false);
  });

  it("keeps the YAML canvas when no live run is wired", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const visual = resolveSplitRunVisual(implement!, { enabled: false, stream: [] });

    expect(visual.canvas.title).toBe("Implementation");
    expect(visual.stream?.some((line) => line.componentName === "Create Branch")).toBe(true);
  });

  it("does not fall back to YAML while the live canvas loads", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const visual = resolveSplitRunVisual(implement!, { enabled: true, stream: [] });

    expect(visual.canvas).toEqual(emptySplitRunCanvas(implement!));
    expect(visual.stream).toEqual(implement?.stream);
  });

  it("does not keep YAML logs when the live canvas fails", () => {
    const implement = SPLIT_RUN_RUNNING.phases.find((phase) => phase.id === "implement");
    const visual = resolveSplitRunVisual(implement!, { enabled: true, isError: true, stream: [] });

    expect(visual.canvas).toEqual(emptySplitRunCanvas(implement!));
    expect(visual.stream).toEqual([]);
  });
});
