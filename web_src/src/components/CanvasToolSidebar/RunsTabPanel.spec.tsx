import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { RunsTabPanel } from "./RunsTabPanel";

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: [] },
    isLoading: false,
  }),
}));

vi.mock("@/components/TimeAgo", () => ({
  TimeAgo: () => <span>time ago</span>,
  renderTimeAgo: () => "time ago",
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn() },
}));

function makeRun(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
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
    ...overrides,
  };
}

const nodes: SuperplaneComponentsNode[] = [
  {
    id: "trigger-1",
    name: "Deploy Trigger",
    type: "TYPE_TRIGGER",
    component: "webhook",
  },
  {
    id: "trigger-2",
    name: "Release Trigger",
    type: "TYPE_TRIGGER",
    component: "webhook",
  },
];

describe("RunsTabPanel", () => {
  const baseProps = {
    canvasId: "canvas-1",
    onSelectRun: () => {},
    workflowNodes: nodes,
  };

  it("shows an empty state when there are no runs", () => {
    render(<RunsTabPanel runs={[]} selectedRunId={null} {...baseProps} />);

    expect(screen.getByText("No Runs")).toBeInTheDocument();
  });

  it("pins running runs above completed runs", () => {
    render(
      <RunsTabPanel
        runs={[
          makeRun({ id: "run-completed", rootEvent: { ...makeRun().rootEvent, customName: "Completed run" } }),
          makeRun({
            id: "run-running",
            state: "STATE_STARTED",
            result: "RESULT_UNKNOWN",
            rootEvent: { ...makeRun().rootEvent, customName: "Running run" },
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
    );

    const rows = screen.getAllByTestId("runs-sidebar-row");
    expect(within(rows[0]).getByText("Running run")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Completed run")).toBeInTheDocument();
  });

  it("filters runs by status", () => {
    render(
      <RunsTabPanel
        runs={[
          makeRun({
            id: "run-failed",
            result: "RESULT_FAILED",
            rootEvent: { ...makeRun().rootEvent, customName: "Broken deploy" },
          }),
          makeRun({
            id: "run-passed",
            result: "RESULT_PASSED",
            rootEvent: { ...makeRun().rootEvent, customName: "Healthy deploy" },
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
    );

    fireEvent.click(screen.getByLabelText("Filter runs"));
    expect(screen.getByText("Passed")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Failed"));
    expect(screen.getByText("Cancelled")).toBeInTheDocument();
    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.queryByText("Completed")).not.toBeInTheDocument();

    expect(screen.getByText("Broken deploy")).toBeInTheDocument();
    expect(screen.queryByText("Healthy deploy")).not.toBeInTheDocument();
  });

  it("loads more runs when the sidebar scroll reaches the end", () => {
    const onLoadMore = vi.fn();
    const runs = Array.from({ length: 25 }, (_, index) =>
      makeRun({
        id: `run-${index}`,
        rootEvent: { ...makeRun().rootEvent, customName: `Deploy ${index}` },
      }),
    );

    const { rerender } = render(<RunsTabPanel runs={runs} selectedRunId={null} {...baseProps} />);
    const scroller = screen.getByTestId("runs-sidebar-scroll");

    Object.defineProperties(scroller, {
      scrollHeight: { configurable: true, value: 1000 },
      clientHeight: { configurable: true, value: 300 },
      scrollTop: { configurable: true, writable: true, value: 0 },
    });

    rerender(
      <RunsTabPanel
        runs={runs}
        selectedRunId={null}
        {...baseProps}
        hasNextPage={true}
        isFetchingNextPage={false}
        onLoadMore={onLoadMore}
      />,
    );

    expect(screen.queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    expect(onLoadMore).not.toHaveBeenCalled();

    scroller.scrollTop = 860;
    fireEvent.scroll(scroller);

    expect(onLoadMore).toHaveBeenCalledTimes(1);
  });

  it("opens run detail on initial deep link", () => {
    render(<RunsTabPanel runs={[makeRun()]} selectedRunId="run-1" initialOpenDetail {...baseProps} />);

    expect(screen.getByTestId("run-detail-panel")).toBeInTheDocument();
  });

  it("returns to the run list when back is clicked", async () => {
    const user = userEvent.setup();
    const onBackToRunList = vi.fn();

    render(
      <RunsTabPanel
        runs={[makeRun()]}
        selectedRunId="run-1"
        initialOpenDetail
        onBackToRunList={onBackToRunList}
        {...baseProps}
      />,
    );

    await user.click(screen.getByTestId("run-detail-back"));
    expect(onBackToRunList).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText("Filter runs")).toBeVisible();
  });
});
