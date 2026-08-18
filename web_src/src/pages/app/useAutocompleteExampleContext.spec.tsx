import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasEvent, SuperplaneComponentsNode } from "@/api-client";
import { useAutocompleteExampleContext } from "./useAutocompleteExampleContext";

const triggerNode: SuperplaneComponentsNode = {
  id: "trigger-1",
  name: "GitHub Check Run",
  type: "TYPE_TRIGGER",
  component: "github.onCheckRun",
};

const finishedExecution: CanvasesCanvasNodeExecution = {
  id: "exec-1",
  nodeId: "trigger-1",
  state: "STATE_FINISHED",
  resultReason: "RESULT_REASON_OK",
  outputs: { default: [{ data: { fired: true } }] },
  createdAt: "2026-07-31T00:00:00Z",
};

const realEvent: CanvasesCanvasEvent = {
  id: "evt-1",
  nodeId: "trigger-1",
  data: { data: { check_run: { conclusion: "success" } } },
  createdAt: "2026-07-31T00:00:00Z",
};

describe("useAutocompleteExampleContext", () => {
  // AppPage feeds this hook the raw execution/event maps. The hook must pass
  // them through to the authoring context without any visibility filtering;
  // the canvas handles presentation separately.
  it("propagates the raw execution/event maps into the authoring context by reference", () => {
    const nodeExecutionsMap = { "trigger-1": [finishedExecution] };
    const nodeEventsMap = { "trigger-1": [realEvent] };

    const { result } = renderHook(() =>
      useAutocompleteExampleContext({
        canvas: { metadata: { id: "canvas-1", name: "Deploy" } },
        canvasNodes: [triggerNode],
        canvasNodesById: new Map([[triggerNode.id!, triggerNode]]),
        incomingNodeIdsByTargetId: new Map(),
        nodeExecutionsMap,
        nodeEventsMap,
        allComponentsByName: new Map(),
        allTriggersByName: new Map(),
      }),
    );

    const context = result.current;
    expect(context.nodeExecutionsMap).toBe(nodeExecutionsMap);
    expect(context.nodeEventsMap).toBe(nodeEventsMap);
    expect(context.app).toEqual({ id: "canvas-1", name: "Deploy", description: "" });
  });

  it("does not expose any visibility/presentation filter on the authoring context", () => {
    // The hook takes no showLiveActivity input. Visibility filtering lives in
    // AppPage, which keeps separate `visible*` maps for the canvas.
    const { result } = renderHook(() =>
      useAutocompleteExampleContext({
        canvas: null,
        canvasNodes: [],
        canvasNodesById: new Map(),
        incomingNodeIdsByTargetId: new Map(),
        nodeExecutionsMap: {},
        nodeEventsMap: {},
        allComponentsByName: new Map(),
        allTriggersByName: new Map(),
      }),
    );

    expect("showLiveActivity" in result.current).toBe(false);
    expect(result.current.nodeExecutionsMap).toEqual({});
  });
});
