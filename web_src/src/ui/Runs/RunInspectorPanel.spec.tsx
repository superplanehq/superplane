import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createEvent, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { RunInspectorPanel } from "./RunInspectorPanel";

const executions: CanvasesCanvasNodeExecution[] = [
  {
    id: "execution-1",
    nodeId: "action-1",
    state: "STATE_FINISHED",
    result: "RESULT_FAILED",
    resultReason: "RESULT_REASON_ERROR",
    resultMessage: "expression evaluation failed",
    createdAt: "2026-05-01T12:00:01Z",
    updatedAt: "2026-05-01T12:00:02Z",
    outputs: {},
    metadata: {},
    configuration: { retries: 1 },
  },
  {
    id: "execution-2",
    nodeId: "action-2",
    previousExecutionId: "execution-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    createdAt: "2026-05-01T12:00:03Z",
    updatedAt: "2026-05-01T12:00:04Z",
    outputs: { default: [{ data: { ok: true } }] },
    metadata: {},
    configuration: { mode: "create" },
  },
];

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value }: { value: unknown }) => <pre data-testid="json-view">{JSON.stringify(value)}</pre>,
}));

const reemitTriggerEventMock = vi.fn();
const cancelExecutionMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api-client")>();
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEventMock(...args),
    canvasesCancelExecution: (...args: unknown[]) => cancelExecutionMock(...args),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions },
    isLoading: false,
  }),
}));

vi.mock("@/pages/app/mappers", () => ({
  getExecutionDetails: () => ({}),
  getState: () => (execution: CanvasesCanvasNodeExecution) =>
    execution.result === "RESULT_FAILED" ? "error" : "success",
  getStateMap: () => ({
    error: { badgeColor: "bg-red-500", label: "error" },
    success: { badgeColor: "bg-emerald-500", label: "success" },
    triggered: { badgeColor: "bg-blue-500", label: "triggered" },
  }),
  getTriggerRenderer: () => ({
    getTitleAndSubtitle: () => ({ title: "Deploy main", subtitle: "" }),
    getRootEventValues: () => ({ Source: "manual" }),
  }),
}));

vi.mock("@/pages/app/utils", () => ({
  buildEventInfo: (event: unknown) => event,
  buildExecutionInfo: (execution: unknown) => execution,
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

beforeEach(() => {
  reemitTriggerEventMock.mockResolvedValue({});
  cancelExecutionMock.mockResolvedValue({});
});

afterEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("RunInspectorPanel", () => {
  it("renders the selected node accordion with backend-provided output sections", () => {
    renderInspector({ selectedNodeId: "action-2" });

    expect(screen.getByTestId("run-inspector-panel")).toBeInTheDocument();
    expect(screen.getByText("Deploy main")).toBeInTheDocument();
    expect(screen.getAllByText("Save Assessment").length).toBeGreaterThan(0);
    expect(screen.getByText("OUTPUT · DEFAULT · 0.02 KB")).toBeInTheDocument();
    expect(screen.getByText(/"data":\{"ok":true\}/)).toBeInTheDocument();
    expect(screen.queryByText(/\[\{"data":\{"ok":true\}\}\]/)).not.toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Copy" }).length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByRole("button", { name: "Open fullscreen" }).length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText("Add Grade Label").length).toBeGreaterThan(0);
  });

  it("does not show trigger input and shows the root event payload as trigger output", () => {
    renderInspector({ selectedNodeId: "trigger-1" });

    expect(screen.queryByRole("button", { name: /Input/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Runtime config/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Output · default/i })).toBeInTheDocument();
    expect(screen.getByText(/"repository":"superplane"/)).toBeInTheDocument();
  });

  it("shows a pinned error summary and jumps to the failing node", () => {
    const onSelectNode = vi.fn();
    renderInspector({ onSelectNode });

    expect(screen.getByText('Errored at "Add Grade Label"')).toBeInTheDocument();
    expect(screen.getByText("expression evaluation failed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Jump to error" }));

    expect(onSelectNode).toHaveBeenCalledWith("action-1");
  });

  it("scrolls to the error output when jumping to an already selected failing node", async () => {
    const scrollIntoView = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoView;
    renderInspector({ selectedNodeId: "action-1" });

    expect(screen.getAllByText("Error - Output not emitted")).toHaveLength(1);
    expect(screen.queryByRole("button", { name: /Output/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Jump to error" }));

    await waitFor(() => {
      expect(scrollIntoView).toHaveBeenCalledWith({ block: "center", behavior: "smooth" });
    });
  });

  it("smoothly scrolls an opened node accordion to the steps top", async () => {
    const scrollIntoView = vi.fn();
    window.HTMLElement.prototype.scrollIntoView = scrollIntoView;
    renderInteractiveInspector();

    fireEvent.click(screen.getByRole("button", { name: /Save Assessment/i }));

    await waitFor(() => {
      expect(scrollIntoView).toHaveBeenCalledWith({ block: "start", behavior: "smooth" });
    });
  });

  it("persists internal accordion preferences generically", () => {
    renderInspector({ selectedNodeId: "action-2" });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(JSON.parse(localStorage.getItem("superplane.runInspector.internalAccordions") || "{}")).toMatchObject({
      input: true,
      runtime: false,
      output: true,
    });
  });

  it("opens the upstream input chain in a modal from the more chip", () => {
    renderInspector({ selectedNodeId: "action-2" });

    fireEvent.click(screen.getByRole("button", { name: "+1 more" }));

    const dialog = screen.getByRole("dialog", { name: "Input chain" });
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: /On Pull Request/i })).toBeInTheDocument();
    expect(within(dialog).getAllByText("Add Grade Label").length).toBeGreaterThanOrEqual(2);
    expect(within(dialog).getByTestId("json-view")).toHaveTextContent("{}");
  });

  it("renders a single close button that closes the inspector", () => {
    const onClose = vi.fn();
    renderInspector({ onClose });

    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    expect(closeButtons).toHaveLength(1);

    fireEvent.click(closeButtons[0]);

    expect(onClose).toHaveBeenCalledOnce();
    expect(screen.queryByRole("button", { name: "Back to live canvas" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close run inspector" })).not.toBeInTheDocument();
  });

  it("re-emits the root trigger event from the global rerun button", async () => {
    renderInspector();

    fireEvent.click(screen.getAllByRole("button", { name: /Rerun/i })[0]);

    await waitFor(() => {
      expect(reemitTriggerEventMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            nodeId: "trigger-1",
            eventId: "event-1",
          },
        }),
      );
    });
  });

  it("stores a resized inspector width", () => {
    Object.defineProperty(window, "innerWidth", { value: 1200, configurable: true });
    renderInspector();

    firePointerEvent(screen.getByTestId("run-inspector-resize-handle"), "pointerDown", 700);
    firePointerEvent(window, "pointerMove", 680);
    firePointerEvent(window, "pointerUp", 680);

    expect(localStorage.getItem("superplane.runInspector.width.v3")).toBe("520");
  });
});

function renderInspector({
  selectedNodeId = null,
  onSelectNode = vi.fn(),
  onClose = vi.fn(),
}: {
  selectedNodeId?: string | null;
  onSelectNode?: (nodeId: string) => void;
  onClose?: () => void;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <RunInspectorPanel
          canvasId="canvas-1"
          run={run}
          workflowNodes={workflowNodes}
          selectedNodeId={selectedNodeId}
          onSelectNode={onSelectNode}
          onClose={onClose}
        />
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

function renderInteractiveInspector() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  function InteractiveInspector() {
    const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

    return (
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <RunInspectorPanel
            canvasId="canvas-1"
            run={run}
            workflowNodes={workflowNodes}
            selectedNodeId={selectedNodeId}
            onSelectNode={setSelectedNodeId}
            onClearSelectedNode={() => setSelectedNodeId(null)}
            onClose={vi.fn()}
          />
        </ThemeProvider>
      </QueryClientProvider>
    );
  }

  return render(<InteractiveInspector />);
}

function firePointerEvent(
  target: Window | Element,
  eventName: "pointerDown" | "pointerMove" | "pointerUp",
  clientX: number,
) {
  const event = createEvent[eventName](target, {});
  Object.defineProperty(event, "pointerId", { value: 1 });
  Object.defineProperty(event, "clientX", { value: clientX });
  fireEvent(target, event);
}

const run: CanvasesCanvasRun = {
  id: "run-1",
  canvasId: "canvas-1",
  state: "STATE_FINISHED",
  result: "RESULT_FAILED",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:05Z",
  rootEvent: {
    id: "event-1",
    nodeId: "trigger-1",
    customName: "Deploy main",
    createdAt: "2026-05-01T12:00:00Z",
    data: { repository: "superplane" },
  },
};

const workflowNodes: SuperplaneComponentsNode[] = [
  {
    id: "trigger-1",
    name: "On Pull Request",
    type: "TYPE_TRIGGER",
    component: "github.onPullRequest",
  },
  {
    id: "action-1",
    name: "Add Grade Label",
    type: "TYPE_ACTION",
    component: "github.addLabel",
  },
  {
    id: "action-2",
    name: "Save Assessment",
    type: "TYPE_ACTION",
    component: "upsertMemory",
  },
];
