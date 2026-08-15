import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type * as ApiClient from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { ReplayNodeModal } from "./ReplayNodeModal";

const replayNodeMock = vi.fn();
const resolveReplayInputsMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReplayNode: (...args: unknown[]) => replayNodeMock(...args),
    canvasesResolveReplayInputs: (...args: unknown[]) => resolveReplayInputsMock(...args),
  };
});

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea data-testid="monaco-stub" value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

function renderModal(overrides: Partial<ComponentProps<typeof ReplayNodeModal>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ReplayNodeModal
          onClose={vi.fn()}
          canvasId="canvas-1"
          nodeId="merge-1"
          originalPayload={{ foo: "bar" }}
          sourceNodeId="lane-a"
          sourceExecutionId="exec-9"
          {...overrides}
        />
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("ReplayNodeModal multi-input", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows one editor per input, labeled by resolved source node name and filled from history", async () => {
    resolveReplayInputsMock.mockResolvedValue({
      data: {
        inputs: [
          { sourceNodeId: "lane-a", payload: { from: "a" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "lane-b", payload: { from: "b" }, status: "STATUS_RECOVERED" },
        ],
      },
    });

    renderModal({ nodeNamesById: { "lane-a": "Fetch Data", "lane-b": "Validate Schema" } });

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(2));

    expect(screen.getByText("Fetch Data")).toBeInTheDocument();
    expect(screen.getByText("Validate Schema")).toBeInTheDocument();
    expect(screen.queryByText("lane-a")).not.toBeInTheDocument();
    expect(screen.queryByText("lane-b")).not.toBeInTheDocument();

    const editors = screen.getAllByTestId("monaco-stub") as HTMLTextAreaElement[];
    expect(editors[0].value).toBe(JSON.stringify({ from: "a" }, null, 2));
    expect(editors[1].value).toBe(JSON.stringify({ from: "b" }, null, 2));
  });

  it("falls back to the raw source id when no name map is supplied", async () => {
    resolveReplayInputsMock.mockResolvedValue({
      data: {
        inputs: [
          { sourceNodeId: "lane-a", payload: { from: "a" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "lane-b", payload: { from: "b" }, status: "STATUS_RECOVERED" },
        ],
      },
    });

    renderModal();

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(2));

    expect(screen.getByText("lane-a")).toBeInTheDocument();
    expect(screen.getByText("lane-b")).toBeInTheDocument();
  });

  it("sends the edited value for one input and the original for the rest", async () => {
    const user = userEvent.setup();
    resolveReplayInputsMock.mockResolvedValue({
      data: {
        inputs: [
          { sourceNodeId: "lane-a", payload: { from: "a" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "lane-b", payload: { from: "b" }, status: "STATUS_RECOVERED" },
        ],
      },
    });
    replayNodeMock.mockResolvedValue({ data: { runId: "run-2", queueItemIds: ["q-2"] } });

    renderModal();

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(2));

    const editors = screen.getAllByTestId("monaco-stub") as HTMLTextAreaElement[];
    fireEvent.change(editors[0], { target: { value: JSON.stringify({ from: "a-edited" }, null, 2) } });

    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => expect(replayNodeMock).toHaveBeenCalledTimes(1));
    const call = replayNodeMock.mock.calls[0][0];
    expect(call.body.inputs).toEqual([
      { payload: { from: "a-edited" }, sourceNodeId: "lane-a" },
      { payload: { from: "b" }, sourceNodeId: "lane-b" },
    ]);
  });

  it("points invalid JSON in the second of two editors at that editor, not the first", async () => {
    const user = userEvent.setup();
    resolveReplayInputsMock.mockResolvedValue({
      data: {
        inputs: [
          { sourceNodeId: "lane-a", payload: { from: "a" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "lane-b", payload: { from: "b" }, status: "STATUS_RECOVERED" },
        ],
      },
    });

    renderModal();

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(2));

    const editors = screen.getAllByTestId("monaco-stub") as HTMLTextAreaElement[];
    fireEvent.change(editors[1], { target: { value: "not valid json" } });

    await waitFor(() => expect(screen.getByTestId("replay-input-1-json-error")).toBeInTheDocument());
    expect(screen.queryByTestId("replay-input-0-json-error")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("launch-replay-button"));
    expect(replayNodeMock).not.toHaveBeenCalled();
  });

  it("renders exactly one editor for an ordinary one-input step", async () => {
    resolveReplayInputsMock.mockResolvedValue({
      data: { inputs: [{ sourceNodeId: "lane-a", payload: { foo: "bar" }, status: "STATUS_RECOVERED" }] },
    });

    renderModal();

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(1));

    expect(screen.getByTestId("replay-payload-editor")).toBeInTheDocument();
    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    expect(editor.value).toBe(JSON.stringify({ foo: "bar" }, null, 2));
  });

  it("blocks launch and names the source when a single-source step's only input is missing", async () => {
    resolveReplayInputsMock.mockResolvedValue({
      data: { inputs: [{ sourceNodeId: "lane-a", status: "STATUS_MISSING" }] },
    });

    renderModal();

    await waitFor(() => expect(resolveReplayInputsMock).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByTestId("launch-replay-button")).toBeDisabled());

    expect(screen.getByTestId("replay-missing-error")).toHaveTextContent("Missing payload for Payload");
  });

  it("unblocks launch once a single-source step's missing input has been filled in", async () => {
    const user = userEvent.setup();
    resolveReplayInputsMock.mockResolvedValue({
      data: { inputs: [{ sourceNodeId: "lane-a", status: "STATUS_MISSING" }] },
    });
    replayNodeMock.mockResolvedValue({ data: { runId: "run-4", queueItemIds: ["q-4"] } });

    renderModal();

    // Wait for the resolved state, not merely for a disabled button: while the
    // lookup is still in flight the button is disabled too, and editing that early
    // would be discarded when the resolved inputs replace the fallback.
    await waitFor(() => expect(screen.getByTestId("replay-missing-error")).toBeInTheDocument());
    expect(screen.getByTestId("launch-replay-button")).toBeDisabled();

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    fireEvent.change(editor, { target: { value: JSON.stringify({ filled: true }) } });

    await waitFor(() => expect(screen.getByTestId("launch-replay-button")).not.toBeDisabled());
    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => expect(replayNodeMock).toHaveBeenCalledTimes(1));
    const call = replayNodeMock.mock.calls[0][0];
    expect(call.body.inputs).toEqual([{ payload: { filled: true }, sourceNodeId: "lane-a" }]);
  });

  it("surfaces a failed input lookup instead of passing the step off as single-input", async () => {
    resolveReplayInputsMock.mockRejectedValue(new Error("resolve boom"));

    renderModal();

    await waitFor(() => expect(screen.getByTestId("replay-api-error")).toBeInTheDocument());
    expect(screen.getByTestId("launch-replay-button")).toBeDisabled();
    expect(replayNodeMock).not.toHaveBeenCalled();
  });

  it("blocks the launch when the only input is detached, rather than replaying it anyway", async () => {
    resolveReplayInputsMock.mockResolvedValue({
      data: { inputs: [{ sourceNodeId: "lane-a", payload: { stale: true }, status: "STATUS_DETACHED" }] },
    });
    replayNodeMock.mockResolvedValue({ data: { runId: "run-3", queueItemIds: ["q-3"] } });

    renderModal();

    await waitFor(() => expect(screen.getByTestId("replay-detached-badge")).toBeInTheDocument());

    expect(screen.getByTestId("launch-replay-button")).toBeDisabled();
    expect(screen.getByTestId("replay-api-error")).toHaveTextContent("nothing to replay");
    expect(replayNodeMock).not.toHaveBeenCalled();
  });

  it("shows a detached input marked as such, but excludes it from the launch request", async () => {
    const user = userEvent.setup();
    resolveReplayInputsMock.mockResolvedValue({
      data: {
        inputs: [
          { sourceNodeId: "lane-a", payload: { from: "a" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "lane-b", payload: { from: "b" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "lane-old", payload: { stale: true }, status: "STATUS_DETACHED" },
        ],
      },
    });
    replayNodeMock.mockResolvedValue({ data: { runId: "run-1", queueItemIds: ["q-1"] } });

    renderModal();

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(3));

    expect(screen.getByTestId("replay-input-2-detached-badge")).toBeInTheDocument();
    expect(screen.getByText("lane-old")).toBeInTheDocument();

    await user.click(screen.getByTestId("launch-replay-button"));

    await waitFor(() => expect(replayNodeMock).toHaveBeenCalledTimes(1));
    const call = replayNodeMock.mock.calls[0][0];
    expect(call.body.inputs).toHaveLength(2);
    expect(call.body.inputs.some((input: { sourceNodeId?: string }) => input.sourceNodeId === "lane-old")).toBe(false);
    expect(call.body.sourceExecutionId).toBe("exec-9");
  });
});
