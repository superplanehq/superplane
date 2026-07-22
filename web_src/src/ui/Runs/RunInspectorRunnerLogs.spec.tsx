import { fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { renderInspector, runnerExecution, runnerNode, workflowNodes } from "./RunInspectorPanel.spec.fixtures";

let mockedExecutions: CanvasesCanvasNodeExecution[] = [runnerExecution];
const useLiveLogStreamMock = vi.fn();

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: mockedExecutions },
    isLoading: false,
  }),
  useCanvasVersion: () => ({
    data: undefined,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
}));

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

vi.mock("@/pages/app/mappers", () => ({
  getExecutionDetails: () => ({}),
  getState: () => () => "success",
  getStateMap: () => ({
    success: { badgeColor: "bg-emerald-500", label: "success" },
    triggered: { badgeColor: "bg-blue-500", label: "triggered" },
  }),
  getTriggerRenderer: () => ({
    getTitleAndSubtitle: () => ({ title: "Deploy main", subtitle: "" }),
    getRootEventValues: () => ({ Source: "manual" }),
  }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

beforeEach(() => {
  mockedExecutions = [runnerExecution];
  useLiveLogStreamMock.mockReturnValue({
    sections: [{ index: 0, text: "npm run build", lines: ["> build", "vite build"], status: "passed" }],
    orphanLines: [],
    error: null,
    isStreaming: false,
    toggleSection: vi.fn(),
    scrollRef: { current: null },
  });
});

afterEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("RunInspector runner logs", () => {
  it("lazy loads runner logs only after the internal logs accordion is opened", () => {
    renderRunnerInspector();

    expect(screen.getByRole("button", { name: /Logs.*Run Bash/i })).toBeInTheDocument();
    expect(useLiveLogStreamMock).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /Logs.*Run Bash/i }));

    expect(useLiveLogStreamMock).toHaveBeenCalledWith("execution-runner-1", false);
    expect(screen.getByText(/\$ npm run build/)).toBeInTheDocument();
    expect(screen.getByText(/vite build/)).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem("superplane.runInspector.internalAccordions") || "{}")).toMatchObject({
      logs: true,
    });
  });

  it("hides runner logs before the broker task has started", () => {
    mockedExecutions = [{ ...runnerExecution, metadata: {} }];

    renderRunnerInspector();

    expect(screen.queryByRole("button", { name: /Logs/i })).not.toBeInTheDocument();
    expect(useLiveLogStreamMock).not.toHaveBeenCalled();
  });

  it("shows a loading message while finished-execution logs are still streaming in", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: null,
      isStreaming: true,
      toggleSection: vi.fn(),
      scrollRef: { current: null },
    });

    renderRunnerInspector();
    fireEvent.click(screen.getByRole("button", { name: /Logs.*Run Bash/i }));

    expect(screen.getByText("Waiting for logs...")).toBeInTheDocument();
    expect(screen.queryByText("No log lines yet.")).not.toBeInTheDocument();
  });

  it("shows the empty message only after streaming settles with no lines", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: null,
      isStreaming: false,
      toggleSection: vi.fn(),
      scrollRef: { current: null },
    });

    renderRunnerInspector();
    fireEvent.click(screen.getByRole("button", { name: /Logs.*Run Bash/i }));

    expect(screen.getByText("No log lines yet.")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs...")).not.toBeInTheDocument();
  });
});

function renderRunnerInspector() {
  renderInspector({
    selectedNodeId: "runner-1",
    workflowNodes: [...workflowNodes, runnerNode],
  });
}
