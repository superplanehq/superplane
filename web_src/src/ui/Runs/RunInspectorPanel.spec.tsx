import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi, afterEach } from "vitest";
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

afterEach(() => {
  localStorage.clear();
});

describe("RunInspectorPanel", () => {
  it("renders the selected node accordion with backend-provided output sections", () => {
    renderInspector({ selectedNodeId: "action-2" });

    expect(screen.getByTestId("run-inspector-panel")).toBeInTheDocument();
    expect(screen.getByText("Deploy main")).toBeInTheDocument();
    expect(screen.getByText("Save Assessment")).toBeInTheDocument();
    expect(screen.getByText("Output · default · 0.02 KB")).toBeInTheDocument();
    expect(screen.getAllByText("Add Grade Label").length).toBeGreaterThan(0);
  });

  it("shows a pinned error summary and jumps to the failing node", () => {
    const onSelectNode = vi.fn();
    renderInspector({ onSelectNode });

    expect(screen.getByText('Errored at "Add Grade Label"')).toBeInTheDocument();
    expect(screen.getByText("expression evaluation failed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Jump to error" }));

    expect(onSelectNode).toHaveBeenCalledWith("action-1");
  });

  it("persists internal accordion preferences generically", () => {
    renderInspector({ selectedNodeId: "action-2" });

    fireEvent.click(screen.getByRole("button", { name: /Runtime config/i }));

    expect(JSON.parse(localStorage.getItem("superplane.runInspector.internalAccordions") || "{}")).toMatchObject({
      input: true,
      runtime: true,
      output: true,
    });
  });
});

function renderInspector({
  selectedNodeId = null,
  onSelectNode = vi.fn(),
}: {
  selectedNodeId?: string | null;
  onSelectNode?: (nodeId: string) => void;
} = {}) {
  return render(
    <RunInspectorPanel
      canvasId="canvas-1"
      run={run}
      workflowNodes={workflowNodes}
      selectedNodeId={selectedNodeId}
      onSelectNode={onSelectNode}
      onClose={vi.fn()}
    />,
    { wrapper: ThemeProvider },
  );
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
