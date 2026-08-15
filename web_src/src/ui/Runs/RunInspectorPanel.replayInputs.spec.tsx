import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { renderInspector } from "./RunInspectorPanel.spec.fixtures";

const RECEIVED_AT = "2026-08-15T12:52:55Z";

const joinExecutions: CanvasesCanvasNodeExecution[] = [
  {
    id: "execution-join",
    nodeId: "join",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: RECEIVED_AT,
    updatedAt: RECEIVED_AT,
    outputs: { success: [{ data: { merged: true } }] },
    metadata: {
      sourceNodes: [
        { nodeId: "lane-a", receivedAt: RECEIVED_AT },
        { nodeId: "lane-b", receivedAt: RECEIVED_AT },
      ],
    },
    configuration: {},
    inputEvent: {
      id: "event-lane-b",
      nodeId: "lane-b",
      channel: "replay",
      createdAt: RECEIVED_AT,
      data: { type: "noop.finished1" },
    },
  },
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

const joinWorkflowNodes: SuperplaneComponentsNode[] = [
  { id: "lane-a", name: "lane A", type: "TYPE_ACTION", component: "noop" },
  { id: "lane-b", name: "lane B", type: "TYPE_ACTION", component: "noop" },
  { id: "join", name: "join lanes", type: "TYPE_ACTION", component: "merge" },
];

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value }: { value: unknown }) => <pre data-testid="json-view">{JSON.stringify(value)}</pre>,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({ data: { executions: joinExecutions }, isLoading: false }),
  useCanvasVersion: () => ({ data: undefined, isLoading: false }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
}));

describe("RunInspectorPanel inputs of a multi-input replay", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  it("offers the source that did not run in the replay next to the one the run is named after", async () => {
    const user = userEvent.setup();
    renderInspector({ run: replayRun, workflowNodes: joinWorkflowNodes, selectedNodeId: "join" });

    await user.click(screen.getByRole("button", { name: "+1 more" }));

    await user.click(screen.getByRole("button", { name: "lane B" }));
    expect(screen.getAllByTestId("json-view").map((node) => node.textContent)).toContain(
      JSON.stringify({ type: "noop.finished1" }),
    );
  });
});
