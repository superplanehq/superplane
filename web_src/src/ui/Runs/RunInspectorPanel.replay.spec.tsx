import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { renderInspector } from "./RunInspectorPanel.spec.fixtures";

const replayNodeMock = vi.fn();
const resolveReplayInputsMock = vi.fn().mockResolvedValue({ data: { inputs: [] } });

const replayableExecutions: CanvasesCanvasNodeExecution[] = [
  {
    id: "execution-1",
    nodeId: "action-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-01T12:00:01Z",
    updatedAt: "2026-05-01T12:00:02Z",
    outputs: {},
    metadata: {},
    configuration: {},
  },
  {
    id: "execution-2",
    nodeId: "action-2",
    previousExecutionId: "execution-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-01T12:00:03Z",
    updatedAt: "2026-05-01T12:00:04Z",
    outputs: {},
    metadata: {},
    configuration: {},
    inputEvent: {
      id: "event-input-2",
      nodeId: "action-1",
      data: { foo: "bar" },
    },
  },
];

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value }: { value: unknown }) => <pre data-testid="json-view">{JSON.stringify(value)}</pre>,
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea data-testid="monaco-stub" value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReplayNode: (...args: unknown[]) => replayNodeMock(...args),
    canvasesResolveReplayInputs: (...args: unknown[]) => resolveReplayInputsMock(...args),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({ data: { executions: replayableExecutions }, isLoading: false }),
  useCanvasVersion: () => ({ data: undefined, isLoading: false }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
}));

describe("RunInspectorPanel replay", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  beforeEach(() => {
    replayNodeMock.mockReset();
  });

  it("offers replay only on steps that carry a past input", () => {
    renderInspector();

    expect(screen.getAllByRole("button", { name: "Replay" })).toHaveLength(1);
    expect(screen.getAllByRole("button", { name: /Rerun/i }).length).toBeGreaterThan(0);
  });

  it("opens the replay modal with the step's own past input payload", async () => {
    const user = userEvent.setup();
    renderInspector();

    await user.click(screen.getByRole("button", { name: "Replay" }));

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    expect(editor.value).toBe(JSON.stringify({ foo: "bar" }, null, 2));
  });

  it("replays the step's own node with the input event's source node", async () => {
    const user = userEvent.setup();
    replayNodeMock.mockResolvedValue({ data: { runId: "run-2", queueItemIds: ["q-1"] } });
    renderInspector();

    await user.click(screen.getByRole("button", { name: "Replay" }));
    await user.click(screen.getByTestId("launch-replay-button"));

    expect(replayNodeMock).toHaveBeenCalledTimes(1);
    const call = replayNodeMock.mock.calls[0][0];
    expect(call.path).toEqual({ canvasId: "canvas-1", nodeId: "action-2" });
    expect(call.body.inputs).toEqual([{ payload: { foo: "bar" }, sourceNodeId: "action-1" }]);
  });

  it("reports the created replay run so the caller can reveal it", async () => {
    const user = userEvent.setup();
    const onReplayCreated = vi.fn();
    replayNodeMock.mockResolvedValue({ data: { runId: "run-9", queueItemIds: ["q-9"] } });
    renderInspector({ onReplayCreated });

    await user.click(screen.getByRole("button", { name: "Replay" }));
    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => expect(onReplayCreated).toHaveBeenCalledWith("run-9"));
  });

  it("sends the step's source execution id on launch", async () => {
    const user = userEvent.setup();
    const onReplayCreated = vi.fn();
    replayNodeMock.mockResolvedValue({ data: { runId: "run-lineage-1", queueItemIds: ["q-1"] } });
    renderInspector({ onReplayCreated });

    await user.click(screen.getByRole("button", { name: "Replay" }));
    await user.click(screen.getByTestId("launch-replay-button"));

    expect(replayNodeMock).toHaveBeenCalledTimes(1);
    const call = replayNodeMock.mock.calls[0][0];
    expect(call.body.sourceExecutionId).toBe("execution-2");
    await waitFor(() => expect(onReplayCreated).toHaveBeenCalledWith("run-lineage-1"));
  });

  it("reports nothing when the replay launch fails", async () => {
    const user = userEvent.setup();
    const onReplayCreated = vi.fn();
    replayNodeMock.mockRejectedValue(new Error("replay rejected"));
    renderInspector({ onReplayCreated });

    await user.click(screen.getByRole("button", { name: "Replay" }));
    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => expect(replayNodeMock).toHaveBeenCalledTimes(1));
    expect(onReplayCreated).not.toHaveBeenCalled();
  });

  it("reports nothing when the API accepts the replay without returning a run id (boundary)", async () => {
    const user = userEvent.setup();
    const onReplayCreated = vi.fn();
    replayNodeMock.mockResolvedValue({ data: { queueItemIds: ["q-10"] } });
    renderInspector({ onReplayCreated });

    await user.click(screen.getByRole("button", { name: "Replay" }));
    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => expect(replayNodeMock).toHaveBeenCalledTimes(1));
    expect(onReplayCreated).not.toHaveBeenCalled();
  });

  it("sends the edited payload rather than the original", async () => {
    const user = userEvent.setup();
    replayNodeMock.mockResolvedValue({ data: { runId: "run-3", queueItemIds: ["q-2"] } });
    renderInspector();

    await user.click(screen.getByRole("button", { name: "Replay" }));

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    fireEvent.change(editor, { target: { value: JSON.stringify({ foo: "baz" }, null, 2) } });

    await user.click(screen.getByTestId("launch-replay-button"));

    const call = replayNodeMock.mock.calls[0][0];
    expect(call.body.inputs?.[0]?.payload).toEqual({ foo: "baz" });
  });
});
