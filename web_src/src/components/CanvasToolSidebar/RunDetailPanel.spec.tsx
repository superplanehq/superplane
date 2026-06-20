import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { RunDetailPanel } from "./RunDetailPanel";

const { useInfiniteCanvasRunsMock } = vi.hoisted(() => ({
  useInfiniteCanvasRunsMock: vi.fn(() => ({
    data: { pages: [{ runs: [] as CanvasesCanvasRun[] }] },
    isLoading: false,
    hasNextPage: false,
    fetchNextPage: vi.fn(),
    isFetchingNextPage: false,
  })),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: {
      executions: [
        {
          nodeId: "action-1",
          createdAt: "2026-05-01T12:01:00Z",
          outputs: {},
          metadata: {},
        },
      ],
    },
    isLoading: false,
  }),
  useInfiniteCanvasRuns: useInfiniteCanvasRunsMock,
}));

vi.mock("@/components/TimeAgo", () => ({
  TimeAgo: () => <span>time ago</span>,
  renderTimeAgo: () => "time ago",
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn() },
}));

function makeRun(): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "canvas-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-01T12:00:00Z",
    rootEvent: {
      id: "event-1",
      nodeId: "trigger-1",
      customName: "Deploy main",
      createdAt: "2026-05-01T12:00:00Z",
    },
    executions: [],
  };
}

const workflowNodes: SuperplaneComponentsNode[] = [
  {
    id: "trigger-1",
    name: "Deploy Trigger",
    type: "TYPE_TRIGGER",
    component: "schedule",
  },
  {
    id: "action-1",
    name: "Notify Team",
    type: "TYPE_ACTION",
    component: "slack.send_message",
  },
];

describe("RunDetailPanel", () => {
  beforeEach(() => {
    useInfiniteCanvasRunsMock.mockReset();
    useInfiniteCanvasRunsMock.mockReturnValue({
      data: { pages: [{ runs: [] as CanvasesCanvasRun[] }] },
      isLoading: false,
      hasNextPage: false,
      fetchNextPage: vi.fn(),
      isFetchingNextPage: false,
    });
  });

  it("expands a node row with sub-runs and navigates on click", async () => {
    const user = userEvent.setup();
    const onNavigateRun = vi.fn();

    useInfiniteCanvasRunsMock.mockReturnValue({
      data: {
        pages: [
          {
            runs: [
              {
                id: "child-run-1",
                canvasId: "canvas-1",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                spawnedByExecutionId: "exec-action-1",
                createdAt: "2026-05-01T12:03:00Z",
                rootEvent: {
                  id: "child-event-1",
                  nodeId: "action-1",
                  customName: "Child run",
                  createdAt: "2026-05-01T12:03:00Z",
                },
                executions: [],
              },
            ],
          },
        ],
      },
      isLoading: false,
      hasNextPage: false,
      fetchNextPage: vi.fn(),
      isFetchingNextPage: false,
    });

    render(
      <RunDetailPanel
        canvasId="canvas-1"
        run={{
          ...makeRun(),
          executions: [
            {
              id: "exec-action-1",
              nodeId: "action-1",
              state: "STATE_FINISHED",
              result: "RESULT_PASSED",
            },
          ],
        }}
        workflowNodes={workflowNodes}
        componentIconMap={{}}
        selectedNodeId={null}
        onSelectNode={() => {}}
        onBack={() => {}}
        onNavigateRun={onNavigateRun}
      />,
    );

    await user.click(screen.getByTestId("run-execution-row-sub-runs-toggle"));
    expect(screen.getByTestId("run-detail-sub-run-group")).toBeInTheDocument();
    expect(screen.getByTestId("run-detail-sub-run-row")).toHaveTextContent("Child run");
    await user.click(screen.getByText("Child run"));
    expect(onNavigateRun).toHaveBeenCalledWith("child-run-1");
  });

  it("renders run metadata and execution chain rows", () => {
    render(
      <RunDetailPanel
        canvasId="canvas-1"
        run={makeRun()}
        workflowNodes={workflowNodes}
        componentIconMap={{}}
        selectedNodeId={null}
        onSelectNode={() => {}}
        onBack={() => {}}
      />,
    );

    expect(screen.getByTestId("run-detail-panel")).toBeInTheDocument();
    expect(screen.getAllByText("Deploy Trigger").length).toBeGreaterThan(0);
    expect(screen.getByText("Notify Team")).toBeInTheDocument();
    expect(screen.getAllByTestId("run-execution-node-row")).toHaveLength(2);
  });

  it("calls onSelectNode when a node row is clicked", async () => {
    const user = userEvent.setup();
    const onSelectNode = vi.fn();

    render(
      <RunDetailPanel
        canvasId="canvas-1"
        run={makeRun()}
        workflowNodes={workflowNodes}
        componentIconMap={{}}
        selectedNodeId={null}
        onSelectNode={onSelectNode}
        onBack={() => {}}
      />,
    );

    await user.click(screen.getByText("Notify Team"));
    expect(onSelectNode).toHaveBeenCalledWith("action-1");
  });

  it("calls onBack when the back button is clicked", () => {
    const onBack = vi.fn();

    render(
      <RunDetailPanel
        canvasId="canvas-1"
        run={makeRun()}
        workflowNodes={workflowNodes}
        componentIconMap={{}}
        selectedNodeId={null}
        onSelectNode={() => {}}
        onBack={onBack}
      />,
    );

    fireEvent.click(screen.getByTestId("run-detail-back"));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("navigates to adjacent runs when the chevrons are clicked", async () => {
    const user = userEvent.setup();
    const onNavigateRun = vi.fn();

    render(
      <RunDetailPanel
        canvasId="canvas-1"
        run={makeRun()}
        workflowNodes={workflowNodes}
        componentIconMap={{}}
        selectedNodeId={null}
        onSelectNode={() => {}}
        onBack={() => {}}
        newerRunId="run-newer"
        olderRunId="run-older"
        canNavigateOlder
        onNavigateRun={onNavigateRun}
      />,
    );

    await user.click(screen.getByTestId("run-detail-newer"));
    await user.click(screen.getByTestId("run-detail-older"));
    expect(onNavigateRun).toHaveBeenNthCalledWith(1, "run-newer");
    expect(onNavigateRun).toHaveBeenNthCalledWith(2, "run-older");
  });

  it("loads more runs when navigating older at the paginated boundary", async () => {
    const user = userEvent.setup();
    const onNavigateOlder = vi.fn();

    render(
      <RunDetailPanel
        canvasId="canvas-1"
        run={makeRun()}
        workflowNodes={workflowNodes}
        componentIconMap={{}}
        selectedNodeId={null}
        onSelectNode={() => {}}
        onBack={() => {}}
        canNavigateOlder
        onNavigateRun={() => {}}
        onNavigateOlder={onNavigateOlder}
      />,
    );

    await user.click(screen.getByTestId("run-detail-older"));
    expect(onNavigateOlder).toHaveBeenCalledTimes(1);
  });
});
