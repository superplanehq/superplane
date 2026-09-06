import { describe, expect, it } from "vitest";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  ComponentsEdge,
  SuperplaneComponentsNode,
} from "@/api-client";
import { buildRunInspectorNodeSections } from "./runNodeDetailModel";

const RECEIVED_AT = "2026-08-15T12:52:55Z";

const workflowNodes: SuperplaneComponentsNode[] = [
  { id: "lane-a", name: "lane A", type: "TYPE_ACTION", component: "noop" },
  { id: "lane-b", name: "lane B", type: "TYPE_ACTION", component: "noop" },
  { id: "join", name: "join lanes", type: "TYPE_ACTION", component: "merge" },
];

const workflowEdges: ComponentsEdge[] = [
  { sourceId: "lane-a", targetId: "join" },
  { sourceId: "lane-b", targetId: "join" },
];

const replayRun: CanvasesCanvasRun = {
  id: "run-replay",
  isReplay: true,
  rootEvent: {
    id: "event-lane-a",
    nodeId: "lane-a",
    channel: "replay",
    createdAt: RECEIVED_AT,
    data: { type: "noop.finished11" },
  },
};

function joinExecution(overrides: Partial<CanvasesCanvasNodeExecution> = {}): CanvasesCanvasNodeExecution {
  return {
    id: "execution-join",
    nodeId: "join",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: RECEIVED_AT,
    updatedAt: RECEIVED_AT,
    outputs: { success: [{ data: { merged: true } }] },
    metadata: {
      groupKey: "event-lane-a",
      eventIDs: ["event-lane-b", "event-lane-a"],
      sourceNodes: [
        { nodeId: "lane-a", receivedAt: RECEIVED_AT },
        { nodeId: "lane-b", receivedAt: RECEIVED_AT },
      ],
    },
    inputEvent: {
      id: "event-lane-b",
      nodeId: "lane-b",
      channel: "replay",
      createdAt: RECEIVED_AT,
      data: { type: "noop.finished1" },
    },
    ...overrides,
  };
}

function joinUpstreamSections(run: CanvasesCanvasRun, executions: CanvasesCanvasNodeExecution[]) {
  const sections = buildRunInspectorNodeSections({ run, executions, workflowNodes, workflowEdges });
  return sections.find((section) => section.nodeId === "join")?.upstreamSections ?? [];
}

describe("buildRunInspectorNodeSections on a multi-input replay", () => {
  it("lists every source the replayed step received, not only the run's root event", () => {
    const upstreamSections = joinUpstreamSections(replayRun, [joinExecution()]);

    expect(upstreamSections.map((section) => section.nodeName)).toEqual(["lane A", "lane B"]);
  });

  it("shows the replay payload of a source that never ran in the replay", () => {
    const upstreamSections = joinUpstreamSections(replayRun, [joinExecution()]);

    expect(upstreamSections.find((section) => section.nodeId === "lane-b")?.output).toEqual({
      type: "noop.finished1",
    });
  });

  it("ignores a source the replayed step is still waiting for", () => {
    const stillWaiting = joinExecution({
      metadata: {
        sourceNodes: [{ nodeId: "lane-a", receivedAt: RECEIVED_AT }, { nodeId: "lane-b" }],
      },
      inputEvent: {
        id: "event-lane-a",
        nodeId: "lane-a",
        channel: "replay",
        createdAt: RECEIVED_AT,
        data: { type: "noop.finished11" },
      },
    });

    expect(joinUpstreamSections(replayRun, [stillWaiting]).map((section) => section.nodeName)).toEqual(["lane A"]);
  });

  it("leaves an ordinary run alone, where both lanes are upstream steps of their own", () => {
    const ordinaryRun: CanvasesCanvasRun = {
      id: "run-ordinary",
      rootEvent: { id: "event-trigger", nodeId: "lane-a", createdAt: RECEIVED_AT, data: { type: "noop.finished11" } },
    };
    const laneB: CanvasesCanvasNodeExecution = {
      id: "execution-lane-b",
      nodeId: "lane-b",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      createdAt: RECEIVED_AT,
      updatedAt: RECEIVED_AT,
      outputs: { default: [{ data: { type: "noop.finished1" } }] },
      metadata: {},
    };
    const join = joinExecution({
      inputEvent: {
        id: "event-lane-b-output",
        nodeId: "lane-b",
        channel: "default",
        createdAt: RECEIVED_AT,
        data: { type: "noop.finished1" },
      },
    });

    expect(joinUpstreamSections(ordinaryRun, [laneB, join]).map((section) => section.nodeName)).toEqual([
      "lane A",
      "lane B",
    ]);
  });
});
