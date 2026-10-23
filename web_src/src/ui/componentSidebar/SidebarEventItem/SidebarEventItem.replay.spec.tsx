import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi, beforeAll } from "vitest";
import type * as ApiClient from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { SidebarEventItem } from "./SidebarEventItem";
import type { SidebarEvent } from "../types";

const replayNodeMock = vi.fn();
const resolveReplayInputsMock = vi.fn().mockResolvedValue({ data: { inputs: [] } });

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

function pastInputEvent(overrides: Partial<SidebarEvent> = {}): SidebarEvent {
  return {
    id: "exec-1",
    title: "Execution 1",
    state: "success",
    isOpen: false,
    kind: "execution",
    nodeId: "node-1",
    executionId: "exec-1",
    originalExecution: {
      id: "exec-1",
      nodeId: "node-1",
      inputEvent: {
        id: "evt-1",
        nodeId: "source-node-1",
        data: { foo: "bar" },
      },
    },
    ...overrides,
  };
}

function renderItem(event: SidebarEvent, nodeNamesById?: Record<string, string>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <MemoryRouter initialEntries={["/org-1/canvases/canvas-1"]}>
          <Routes>
            <Route
              path="/:organizationId/canvases/:canvasId"
              element={
                <SidebarEventItem
                  event={event}
                  index={0}
                  isOpen={false}
                  onToggleOpen={vi.fn()}
                  nodeNamesById={nodeNamesById}
                />
              }
            />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

async function openReplayModal(user: ReturnType<typeof userEvent.setup>) {
  fireEvent.mouseEnter(screen.getByTestId("sidebar-event-item"));
  await user.click(screen.getByRole("button", { name: "Open actions" }));
  await user.click(screen.getByTestId("replay-item"));
}

describe("SidebarEventItem replay", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  it("shows the selected past input payload in an editable field with replay enabled", async () => {
    const user = userEvent.setup();
    renderItem(pastInputEvent());

    await openReplayModal(user);

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    expect(editor.value).toBe(JSON.stringify({ foo: "bar" }, null, 2));
    expect(screen.getByTestId("launch-replay-button")).not.toBeDisabled();
  });

  it("shows the diff against the original and sends the edited payload, not the original", async () => {
    const user = userEvent.setup();
    replayNodeMock.mockResolvedValue({ data: { runId: "run-1", queueItemIds: ["q-1"] } });
    renderItem(pastInputEvent());

    await openReplayModal(user);

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    fireEvent.change(editor, { target: { value: JSON.stringify({ foo: "baz" }, null, 2) } });

    expect(screen.getByTestId("replay-diff-line-added").textContent).toContain("baz");
    expect(screen.getByTestId("replay-diff-line-removed").textContent).toContain("bar");

    await user.click(screen.getByTestId("launch-replay-button"));

    expect(replayNodeMock).toHaveBeenCalledTimes(1);
    const call = replayNodeMock.mock.calls[0][0];
    expect(call.path).toEqual({ canvasId: "canvas-1", nodeId: "node-1" });
    expect(call.body.inputs).toEqual([{ payload: { foo: "baz" }, sourceNodeId: "source-node-1" }]);
  });

  it("disables replay and shows a parse error for invalid JSON", async () => {
    const user = userEvent.setup();
    renderItem(pastInputEvent());

    await openReplayModal(user);

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    fireEvent.change(editor, { target: { value: "{ not valid json" } });

    expect(screen.getByTestId("launch-replay-button")).toBeDisabled();
    expect(screen.getByTestId("replay-json-error")).toBeInTheDocument();
  });

  it("disables replay for valid JSON that is not an object (boundary)", async () => {
    const user = userEvent.setup();
    renderItem(pastInputEvent());

    await openReplayModal(user);

    const editor = screen.getByTestId("monaco-stub") as HTMLTextAreaElement;
    fireEvent.change(editor, { target: { value: "[1,2]" } });

    expect(screen.getByTestId("launch-replay-button")).toBeDisabled();
    expect(screen.getByTestId("replay-json-error")).toBeInTheDocument();
  });

  it("labels a merge step's editors with source node names when a name map is supplied", async () => {
    const user = userEvent.setup();
    resolveReplayInputsMock.mockResolvedValue({
      data: {
        inputs: [
          { sourceNodeId: "source-node-1", payload: { from: "a" }, status: "STATUS_RECOVERED" },
          { sourceNodeId: "source-node-2", payload: { from: "b" }, status: "STATUS_RECOVERED" },
        ],
      },
    });

    renderItem(pastInputEvent(), { "source-node-1": "Fetch Data", "source-node-2": "Validate Schema" });

    await openReplayModal(user);

    await waitFor(() => expect(screen.getAllByTestId("monaco-stub")).toHaveLength(2));

    expect(screen.getByText("Fetch Data")).toBeInTheDocument();
    expect(screen.getByText("Validate Schema")).toBeInTheDocument();
    expect(screen.queryByText("source-node-1")).not.toBeInTheDocument();
    expect(screen.queryByText("source-node-2")).not.toBeInTheDocument();
  });

  it("offers no Replay for an execution without a single surviving consumed event (boundary, pinned invariant)", async () => {
    const user = userEvent.setup();
    renderItem(pastInputEvent({ originalExecution: { id: "exec-1", nodeId: "node-1" } }));

    fireEvent.mouseEnter(screen.getByTestId("sidebar-event-item"));
    await user.hover(screen.getByTestId("sidebar-event-item"));

    expect(screen.queryByRole("button", { name: "Open actions" })).not.toBeInTheDocument();
  });
});
