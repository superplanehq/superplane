import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi, afterEach, beforeEach } from "vitest";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import {
  executions,
  firePointerEvent,
  renderInspector,
  renderInteractiveInspector,
  runningExecutions,
  runningRun,
} from "./RunInspectorPanel.spec.fixtures";

let mockedExecutions = executions;
let mockedExecutionsLoading = false;

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value }: { value: unknown }) => <pre data-testid="json-view">{JSON.stringify(value)}</pre>,
}));

const reemitTriggerEventMock = vi.fn();
const cancelExecutionMock = vi.fn();
const invokeExecutionHookMock = vi.fn();
const listNodeQueueItemsMock = vi.fn();
const deleteNodeQueueItemMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEventMock(...args),
    canvasesCancelExecution: (...args: unknown[]) => cancelExecutionMock(...args),
    canvasesInvokeNodeExecutionHook: (...args: unknown[]) => invokeExecutionHookMock(...args),
    canvasesListNodeQueueItems: (...args: unknown[]) => listNodeQueueItemsMock(...args),
    canvasesDeleteNodeQueueItem: (...args: unknown[]) => deleteNodeQueueItemMock(...args),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: mockedExecutions },
    isLoading: mockedExecutionsLoading,
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
  mockedExecutions = executions;
  mockedExecutionsLoading = false;
  reemitTriggerEventMock.mockResolvedValue({});
  cancelExecutionMock.mockResolvedValue({});
  invokeExecutionHookMock.mockResolvedValue({});
  listNodeQueueItemsMock.mockResolvedValue({ data: { items: [] } });
  deleteNodeQueueItemMock.mockResolvedValue({});
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
    expect(screen.queryByText(/"data":\{"ok":true\}/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Output · default/i }));

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
    expect(screen.queryByText(/"repository":"superplane"/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Output · default/i }));

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
      input: false,
      runtime: true,
      output: false,
    });
  });

  it("shows applied runtime config as a read-only form with a JSON switch", () => {
    renderInspector({ selectedNodeId: "action-2" });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(screen.getByRole("button", { name: "Form" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("Mode")).toBeInTheDocument();
    expect(screen.getByText("Create")).toBeInTheDocument();
    expect(screen.getByText("Approvers")).toBeInTheDocument();
    expect(screen.getByText("Request approval from")).toBeInTheDocument();
    expect(screen.getByText("Any one")).toBeInTheDocument();
    expect(screen.queryByText(/"type":"anyone"/)).not.toBeInTheDocument();
    expect(screen.queryByTestId("json-view")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "JSON" }));

    expect(screen.getByRole("button", { name: "JSON" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("json-view")).toHaveTextContent(/"mode":"create"/);
    expect(screen.getByTestId("json-view")).toHaveTextContent(/"type":"anyone"/);
  });

  it("honors stored internal accordion preferences", () => {
    localStorage.setItem(
      "superplane.runInspector.internalAccordions",
      JSON.stringify({ input: false, runtime: false, output: true }),
    );

    renderInspector({ selectedNodeId: "action-2" });

    expect(screen.getByText(/"data":\{"ok":true\}/)).toBeInTheDocument();
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

  it("stops running executions and queued items for the inspected run", async () => {
    mockedExecutions = runningExecutions;
    listNodeQueueItemsMock.mockImplementation(({ path }: { path: { nodeId: string } }) => ({
      data: {
        items:
          path.nodeId === "action-2"
            ? [
                { id: "queue-1", rootEvent: { id: "event-running" } },
                { id: "queue-other", rootEvent: { id: "other-event" } },
              ]
            : [],
      },
    }));

    renderInspector({ run: runningRun });

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);

    await waitFor(() => {
      expect(cancelExecutionMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            executionId: "execution-running",
          },
        }),
      );
      expect(deleteNodeQueueItemMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            nodeId: "action-2",
            itemId: "queue-1",
          },
        }),
      );
    });

    expect(deleteNodeQueueItemMock).toHaveBeenCalledTimes(1);
  });

  it("stops an active node execution from the node accordion header", async () => {
    mockedExecutions = runningExecutions;

    renderInspector({ run: runningRun, selectedNodeId: "action-2" });

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" }).at(-1)!);

    await waitFor(() => {
      expect(cancelExecutionMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            executionId: "execution-running",
          },
        }),
      );
    });
  });

  it("shows approval actions for actionable pending approval records", async () => {
    mockedExecutions = [
      {
        id: "execution-approval",
        nodeId: "approval-1",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        resultReason: "RESULT_REASON_OK",
        resultMessage: "",
        createdAt: "2026-05-01T12:00:03Z",
        updatedAt: "2026-05-01T12:00:04Z",
        outputs: {},
        metadata: {
          records: [{ index: 0, state: "pending", type: "user", user: { id: "account-1", email: "me@example.com" } }],
        },
        configuration: {},
      },
    ];

    renderInspector({
      run: runningRun,
      selectedNodeId: "approval-1",
      account: { id: "account-1", name: "Me", email: "me@example.com", avatar_url: "", installation_admin: false },
    });

    fireEvent.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => {
      expect(invokeExecutionHookMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            executionId: "execution-approval",
            hookName: "approve",
          },
          body: {
            parameters: { index: 0, comment: "" },
          },
        }),
      );
    });
  });

  it("shows approval nodes from run execution refs when the current user cannot approve", () => {
    mockedExecutions = [];

    renderInspector({
      run: {
        ...runningRun,
        executions: [
          {
            id: "execution-approval",
            nodeId: "approval-1",
            state: "STATE_STARTED",
            result: "RESULT_UNKNOWN",
            resultReason: "RESULT_REASON_OK",
            createdAt: "2026-05-01T12:00:03Z",
            updatedAt: "2026-05-01T12:00:04Z",
          },
        ],
      },
      selectedNodeId: "approval-1",
      account: {
        id: "account-other",
        name: "Other user",
        email: "other@example.com",
        avatar_url: "",
        installation_admin: false,
      },
    });

    expect(screen.getByRole("button", { name: /Await Approval/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
  });

  it("keeps the stop action disabled while executions are loading", () => {
    mockedExecutions = [];
    mockedExecutionsLoading = true;

    renderInspector({ run: runningRun });

    const stopButton = screen.getByRole("button", { name: "Stop" });
    expect(stopButton).toBeDisabled();

    fireEvent.click(stopButton);

    expect(cancelExecutionMock).not.toHaveBeenCalled();
    expect(listNodeQueueItemsMock).not.toHaveBeenCalled();
  });

  it("keeps the stop action disabled when no action steps are loaded", () => {
    mockedExecutions = [];
    mockedExecutionsLoading = false;

    renderInspector({ run: runningRun });

    const stopButton = screen.getByRole("button", { name: "Stop" });
    expect(stopButton).toBeDisabled();

    fireEvent.click(stopButton);

    expect(cancelExecutionMock).not.toHaveBeenCalled();
    expect(listNodeQueueItemsMock).not.toHaveBeenCalled();
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
