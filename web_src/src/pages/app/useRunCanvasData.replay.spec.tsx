import { describe, expect, it } from "vitest";
import { render, renderHook, screen } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import type {
  CanvasesCanvas,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
} from "@/api-client";
import { ComponentBase } from "@/ui/componentBase";
import type { ComponentBaseProps } from "@/ui/componentBase";
import { useRunCanvasData } from "./useRunCanvasData";
import type { CanvasNode } from "@/ui/CanvasPage";
import {
  annotateReplayInputSourceNodes,
  collectReplayInputSourceNodeIds,
  REPLAY_INPUT_EMPTY_STATE_DESCRIPTION,
  REPLAY_INPUT_EMPTY_STATE_TITLE,
} from "./lib/replay-input-node";

const DEFAULT_EMPTY_STATE_TEXT = "Waiting for the first run...";

const runCanvas: CanvasesCanvas = {
  metadata: { id: "canvas-1", name: "replay-broken-if" },
  spec: {
    nodes: [
      { id: "start", name: "start", type: "TYPE_TRIGGER", component: "webhook" },
      { id: "get-cat-fact", name: "get cat fact", type: "TYPE_ACTION", component: "http" },
      {
        id: "if-short",
        name: "if",
        type: "TYPE_ACTION",
        component: "if",
        configuration: { expression: "length($.fact) < 10" },
      },
    ],
    edges: [
      { sourceId: "start", targetId: "get-cat-fact", channel: "default" },
      { sourceId: "get-cat-fact", targetId: "if-short", channel: "default" },
    ],
  },
} as CanvasesCanvas;

function buildRun(overrides: Partial<CanvasesCanvasRun>): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "canvas-1",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    ...overrides,
  } as CanvasesCanvasRun;
}

type RunCanvasOptions = {
  canvas?: CanvasesCanvas;
  fullExecutions?: CanvasesCanvasNodeExecution[];
  storeExecutions?: Record<string, CanvasesCanvasNodeExecution[]>;
};

function renderRunCanvas(
  selectedRun: CanvasesCanvasRun,
  { canvas = runCanvas, fullExecutions = [], storeExecutions = {} }: RunCanvasOptions = {},
) {
  const { result } = renderHook(() =>
    useRunCanvasData({
      isRunInspectionMode: true,
      selectedRun,
      selectedRunCanvas: canvas,
      canvasLoading: false,
      triggersLoading: false,
      componentsLoading: false,
      isSelectedRunVersionLoading: false,
      allTriggers: [],
      allComponents: [],
      canvasId: "canvas-1",
      queryClient: new QueryClient(),
      me: null,
      visibleNodeExecutionsMap: storeExecutions,
      selectedRunFullExecutions: fullExecutions,
    }),
  );

  return result.current;
}

function componentPropsFor(
  nodeId: string,
  selectedRun: CanvasesCanvasRun,
  options?: RunCanvasOptions,
): ComponentBaseProps {
  const data = renderRunCanvas(selectedRun, options)!.nodes.find((node) => node.id === nodeId)!.data as {
    component: ComponentBaseProps;
  };

  return data.component;
}

// A replay of `if` carries a synthetic input event attributed to `get cat fact`
// (pkg/models/canvas_node_replay.go), so that node is a participant that never ran.
const replayRun = buildRun({
  isReplay: true,
  rootEvent: { id: "event-200c", nodeId: "get-cat-fact", canvasId: "canvas-1" },
  executions: [{ id: "execution-1", nodeId: "if-short", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
});

describe("run canvas empty state on a replay run", () => {
  it("names the replay input instead of claiming the source node is waiting for its first run", () => {
    render(<ComponentBase {...componentPropsFor("get-cat-fact", replayRun)} />);

    expect(screen.getByText(REPLAY_INPUT_EMPTY_STATE_TITLE)).toBeInTheDocument();
    expect(screen.getByText(REPLAY_INPUT_EMPTY_STATE_DESCRIPTION)).toBeInTheDocument();
    expect(screen.queryByText(DEFAULT_EMPTY_STATE_TEXT)).not.toBeInTheDocument();
  });

  it("leaves the replayed node itself alone — it has a real execution to show", () => {
    const props = componentPropsFor("if-short", replayRun);

    expect(props.title).toBe("if");
    expect(props.includeEmptyState).toBe(false);
    expect(props.emptyStateProps?.title).toBeUndefined();
  });

  it("keeps the generic empty state on an ordinary run rooted on its trigger", () => {
    const ordinaryRun = buildRun({
      isReplay: false,
      rootEvent: { id: "event-1", nodeId: "start", canvasId: "canvas-1" },
      executions: [{ id: "execution-1", nodeId: "if-short", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
    });

    render(<ComponentBase {...componentPropsFor("get-cat-fact", ordinaryRun)} />);

    expect(screen.getByText(DEFAULT_EMPTY_STATE_TEXT)).toBeInTheDocument();
    expect(screen.queryByText(REPLAY_INPUT_EMPTY_STATE_TITLE)).not.toBeInTheDocument();
  });

  it("does not annotate a replay source node that did execute in the same run", () => {
    const selfSourcedReplay = buildRun({
      isReplay: true,
      rootEvent: { id: "event-200c", nodeId: "get-cat-fact", canvasId: "canvas-1" },
      executions: [{ id: "execution-1", nodeId: "get-cat-fact", state: "STATE_FINISHED", result: "RESULT_PASSED" }],
    });

    expect(componentPropsFor("get-cat-fact", selfSourcedReplay).emptyStateProps?.title).toBeUndefined();
  });

  it("survives a replay run whose root event is missing", () => {
    const rootlessReplay = buildRun({
      isReplay: true,
      executions: [{ id: "execution-1", nodeId: "if-short", state: "STATE_FINISHED", result: "RESULT_FAILED" }],
    });

    render(<ComponentBase {...componentPropsFor("get-cat-fact", rootlessReplay)} />);

    expect(screen.getByText(DEFAULT_EMPTY_STATE_TEXT)).toBeInTheDocument();
  });
});

const joinCanvas: CanvasesCanvas = {
  metadata: { id: "canvas-2", name: "replay-fanout-merge" },
  spec: {
    nodes: [
      { id: "start", name: "start", type: "TYPE_TRIGGER", component: "webhook" },
      { id: "lane-a", name: "lane A", type: "TYPE_ACTION", component: "http" },
      { id: "lane-b", name: "lane B", type: "TYPE_ACTION", component: "http" },
      { id: "join", name: "join lanes", type: "TYPE_ACTION", component: "merge" },
      { id: "lane-c", name: "lane C", type: "TYPE_ACTION", component: "http" },
      { id: "join-2", name: "join again", type: "TYPE_ACTION", component: "merge" },
    ],
    edges: [
      { sourceId: "start", targetId: "lane-a", channel: "default" },
      { sourceId: "start", targetId: "lane-b", channel: "default" },
      { sourceId: "lane-a", targetId: "join", channel: "default" },
      { sourceId: "lane-b", targetId: "join", channel: "default" },
      { sourceId: "join", targetId: "join-2", channel: "default" },
      { sourceId: "lane-c", targetId: "join-2", channel: "default" },
    ],
  },
} as CanvasesCanvas;

const joinExecution: CanvasesCanvasNodeExecutionRef = {
  id: "execution-join",
  nodeId: "join",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
};

// A multi-input replay writes one input event per incoming source, and the run
// can name only one of them as its root event: every other source is knowable
// only from the replayed node's own metadata.
const multiInputReplayRun = buildRun({
  id: "run-2",
  canvasId: "canvas-2",
  result: "RESULT_PASSED",
  isReplay: true,
  rootEvent: { id: "event-lane-b", nodeId: "lane-b", canvasId: "canvas-2" },
  executions: [joinExecution],
});

// The replayed node's own execution is fed by the synthetic `replay` event; an
// execution further down the run is fed by an ordinary one.
function withMetadata(
  execution: CanvasesCanvasNodeExecutionRef,
  sourceNodes: { nodeId: string; receivedAt?: string }[],
  channel = "replay",
): CanvasesCanvasNodeExecution {
  return {
    ...execution,
    inputEvent: { id: `event-${execution.id}`, channel, canvasId: "canvas-2" },
    metadata: { sourceNodes },
  } as CanvasesCanvasNodeExecution;
}

// The run's own executions carry the metadata; the node execution store is the
// fallback `hydrateRunExecution` uses when they have not loaded.
function joinRunOptions(sourceNodes: { nodeId: string; receivedAt?: string }[]): RunCanvasOptions {
  return { canvas: joinCanvas, fullExecutions: [withMetadata(joinExecution, sourceNodes)] };
}

const bothLanesReceived = [
  { nodeId: "lane-a", receivedAt: "2026-08-14T07:18:09Z" },
  { nodeId: "lane-b", receivedAt: "2026-08-14T07:18:09Z" },
];

describe("run canvas on a multi-input replay run", () => {
  const options = joinRunOptions(bothLanesReceived);

  it("names every source the replay took an input from, not just the one the root event carries", () => {
    render(<ComponentBase {...componentPropsFor("lane-a", multiInputReplayRun, options)} />);

    expect(screen.getByText(REPLAY_INPUT_EMPTY_STATE_TITLE)).toBeInTheDocument();
    expect(screen.queryByText(DEFAULT_EMPTY_STATE_TEXT)).not.toBeInTheDocument();
    expect(componentPropsFor("lane-b", multiInputReplayRun, options).emptyStateProps?.title).toBe(
      REPLAY_INPUT_EMPTY_STATE_TITLE,
    );
  });

  it("falls back to the node execution store while the run's own executions are still loading", () => {
    const fromStore = {
      canvas: joinCanvas,
      storeExecutions: { join: [withMetadata(joinExecution, bothLanesReceived)] },
    };

    expect(componentPropsFor("lane-a", multiInputReplayRun, fromStore).emptyStateProps?.title).toBe(
      REPLAY_INPUT_EMPTY_STATE_TITLE,
    );
  });

  it("counts every source as a run participant so its card body is not blanked", () => {
    const { participantNodeIds } = renderRunCanvas(multiInputReplayRun, options)!;

    expect(participantNodeIds).toEqual(expect.arrayContaining(["lane-a", "lane-b", "join"]));
    expect(participantNodeIds).not.toContain("start");
  });

  it("ignores a source the aggregating node is still waiting for", () => {
    const waitingOnLaneA = joinRunOptions([
      { nodeId: "lane-a" },
      { nodeId: "lane-b", receivedAt: "2026-08-14T07:18:09Z" },
    ]);

    expect(componentPropsFor("lane-a", multiInputReplayRun, waitingOnLaneA).emptyStateProps?.title).toBeUndefined();
    expect(renderRunCanvas(multiInputReplayRun, waitingOnLaneA)!.participantNodeIds).not.toContain("lane-a");
  });

  // A merge further down the run can group in an event produced outside the
  // replay. The node that sent it never ran here either, but it was no input of
  // this replay and must not be labelled as one.
  it("ignores the sources of an aggregating node that is not the replayed one", () => {
    const downstreamJoin: CanvasesCanvasNodeExecutionRef = {
      ...joinExecution,
      id: "execution-join-2",
      nodeId: "join-2",
    };
    const withDownstream = {
      canvas: joinCanvas,
      fullExecutions: [
        withMetadata(joinExecution, bothLanesReceived),
        withMetadata(
          downstreamJoin,
          [
            { nodeId: "join", receivedAt: "2026-08-14T07:18:10Z" },
            { nodeId: "lane-c", receivedAt: "2026-08-14T07:00:00Z" },
          ],
          "default",
        ),
      ],
    };

    expect(componentPropsFor("lane-c", multiInputReplayRun, withDownstream).emptyStateProps?.title).toBeUndefined();
    expect(renderRunCanvas(multiInputReplayRun, withDownstream)!.participantNodeIds).not.toContain("lane-c");
    expect(componentPropsFor("lane-a", multiInputReplayRun, withDownstream).emptyStateProps?.title).toBe(
      REPLAY_INPUT_EMPTY_STATE_TITLE,
    );
  });

  it("leaves an ordinary run's sources alone", () => {
    const ordinaryJoinRun = buildRun({
      id: "run-3",
      canvasId: "canvas-2",
      isReplay: false,
      rootEvent: { id: "event-1", nodeId: "start", canvasId: "canvas-2" },
      executions: [joinExecution],
    });

    expect(componentPropsFor("lane-a", ordinaryJoinRun, options).emptyStateProps?.title).toBeUndefined();
    expect(renderRunCanvas(ordinaryJoinRun, options)!.participantNodeIds).not.toContain("lane-a");
  });
});

// The hook-level boundary above is also guarded by the mapper (a node with an
// execution sets includeEmptyState: false), so these two isolate the
// executedNodeIds check itself: same card, only the executed set differs.
describe("replay input source nodes", () => {
  function emptySourceCard(): CanvasNode {
    return {
      id: "get-cat-fact",
      position: { x: 0, y: 0 },
      data: { type: "component", component: { includeEmptyState: true } },
    } as CanvasNode;
  }

  function titleOf(nodes: CanvasNode[]): string | undefined {
    const { component } = nodes[0].data as { component: ComponentBaseProps };
    return component.emptyStateProps?.title;
  }

  it("annotates a card whose node stayed out of the replay", () => {
    const annotated = annotateReplayInputSourceNodes([emptySourceCard()], new Set(["get-cat-fact"]));

    expect(titleOf(annotated)).toBe(REPLAY_INPUT_EMPTY_STATE_TITLE);
  });

  it("leaves the identical card alone once its node counts as executed", () => {
    const sourceNodeIds = collectReplayInputSourceNodeIds({
      isReplay: true,
      rootEventNodeId: "get-cat-fact",
      executions: [],
      executedNodeIds: new Set(["get-cat-fact"]),
    });

    expect(titleOf(annotateReplayInputSourceNodes([emptySourceCard()], sourceNodeIds))).toBeUndefined();
  });
});
