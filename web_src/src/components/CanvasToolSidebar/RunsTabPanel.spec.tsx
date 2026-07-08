import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { RunsTabPanel } from "./RunsTabPanel";

const routerWrapper = ({ children }: { children: React.ReactNode }) => <MemoryRouter>{children}</MemoryRouter>;

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

afterEach(() => {
  vi.useRealTimers();
  localStorage.clear();
});

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
    onSelectLiveCanvas: () => {},
    workflowNodes: nodes,
  };

  it("shows the Live Canvas row and selects it when no run is active", () => {
    const onSelectLiveCanvas = vi.fn();
    render(<RunsTabPanel runs={[]} selectedRunId={null} {...baseProps} onSelectLiveCanvas={onSelectLiveCanvas} />, {
      wrapper: routerWrapper,
    });

    const liveCanvas = screen.getByTestId("runs-sidebar-live-canvas");
    expect(liveCanvas).toHaveAttribute("aria-current", "true");

    fireEvent.click(liveCanvas);
    expect(onSelectLiveCanvas).toHaveBeenCalledTimes(1);
  });

  it("shows an empty state when there are no runs", () => {
    render(<RunsTabPanel runs={[]} selectedRunId={null} {...baseProps} />, { wrapper: routerWrapper });

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
      { wrapper: routerWrapper },
    );

    const rows = screen.getAllByTestId("runs-sidebar-row");
    expect(within(rows[0]).getByText("Running run")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Completed run")).toBeInTheDocument();
  });

  it("selects a run when its timestamp is clicked", async () => {
    const user = userEvent.setup();
    const onSelectRun = vi.fn();

    render(<RunsTabPanel runs={[makeRun()]} selectedRunId={null} {...baseProps} onSelectRun={onSelectRun} />, {
      wrapper: routerWrapper,
    });

    const timestamp = document.querySelector('time[datetime="2026-05-01T12:00:00.000Z"]') as HTMLElement;
    await user.click(timestamp);

    expect(onSelectRun).toHaveBeenCalledTimes(1);
    expect(onSelectRun).toHaveBeenCalledWith("run-1");
  });

  it("opens a run in a new tab when its timestamp is modified-clicked", () => {
    const onSelectRun = vi.fn();
    const open = vi.spyOn(window, "open").mockImplementation(() => null);

    render(
      <MemoryRouter initialEntries={["/org-1/apps/app-1"]}>
        <Routes>
          <Route
            path="/:organizationId/apps/:appId"
            element={<RunsTabPanel runs={[makeRun()]} selectedRunId={null} {...baseProps} onSelectRun={onSelectRun} />}
          />
        </Routes>
      </MemoryRouter>,
    );

    const timestamp = document.querySelector('time[datetime="2026-05-01T12:00:00.000Z"]') as HTMLElement;
    fireEvent.click(timestamp, { metaKey: true });

    expect(onSelectRun).not.toHaveBeenCalled();
    expect(open).toHaveBeenCalledWith("/org-1/apps/app-1?run=run-1", "_blank", "noopener,noreferrer");

    open.mockRestore();
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
      { wrapper: routerWrapper },
    );

    fireEvent.click(screen.getByLabelText("Filter runs"));
    const filters = screen.getByRole("dialog");
    expect(within(filters).getByText("Passed")).toBeInTheDocument();
    fireEvent.click(within(filters).getByText("Failed"));
    expect(within(filters).getByText("Cancelled")).toBeInTheDocument();
    expect(within(filters).getByText("Running")).toBeInTheDocument();
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

    const { rerender } = render(<RunsTabPanel runs={runs} selectedRunId={null} {...baseProps} />, {
      wrapper: routerWrapper,
    });
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

  it("keeps the runs list visible on initial deep link", () => {
    render(<RunsTabPanel runs={[makeRun()]} selectedRunId="run-1" initialOpenDetail {...baseProps} />, {
      wrapper: routerWrapper,
    });

    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Filter runs")).toBeVisible();
    expect(screen.getByText("Deploy main")).toBeInTheDocument();
  });

  it("keeps the runs list visible when the selected run is provided outside the loaded list", () => {
    render(<RunsTabPanel runs={[]} selectedRunId="run-1" selectedRun={makeRun()} initialOpenDetail {...baseProps} />, {
      wrapper: routerWrapper,
    });

    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Filter runs")).toBeVisible();
  });

  it("does not replace the list with a loading detail while the selected run is resolving", () => {
    render(<RunsTabPanel runs={[]} selectedRunId="run-1" initialOpenDetail isSelectedRunLoading {...baseProps} />, {
      wrapper: routerWrapper,
    });

    expect(screen.queryByText("Loading run…")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Filter runs")).toBeVisible();
  });

  it("keeps the runs list visible when initialOpenDetail arrives after mount", () => {
    const runs = [makeRun()];

    const { rerender } = render(<RunsTabPanel runs={runs} selectedRunId={null} {...baseProps} />, {
      wrapper: routerWrapper,
    });
    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();

    rerender(<RunsTabPanel runs={runs} selectedRunId="run-1" initialOpenDetail {...baseProps} />);

    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();
    expect(screen.getByText("Deploy main")).toBeInTheDocument();
  });

  it("keeps the selected run highlighted when the selected run changes from the URL", () => {
    const runs = [
      makeRun({ id: "run-1", rootEvent: { ...makeRun().rootEvent, customName: "First run" } }),
      makeRun({ id: "run-2", rootEvent: { ...makeRun().rootEvent, customName: "Second run" } }),
    ];

    const { rerender } = render(<RunsTabPanel runs={runs} selectedRunId={null} {...baseProps} />, {
      wrapper: routerWrapper,
    });
    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();

    rerender(<RunsTabPanel runs={runs} selectedRunId="run-1" {...baseProps} />);
    expect(screen.getByLabelText("Filter runs")).toBeVisible();

    rerender(<RunsTabPanel runs={runs} selectedRunId="run-2" {...baseProps} />);
    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();
    expect(screen.getByText("Second run")).toBeInTheDocument();
  });

  it("keeps the list visible when URL navigation returns to a dismissed run", () => {
    const runs = [
      makeRun({ id: "run-1", rootEvent: { ...makeRun().rootEvent, customName: "First run" } }),
      makeRun({ id: "run-2", rootEvent: { ...makeRun().rootEvent, customName: "Second run" } }),
    ];

    const { rerender } = render(
      <RunsTabPanel runs={runs} selectedRunId="run-1" detailDismissedForRunId="run-1" {...baseProps} />,
      { wrapper: routerWrapper },
    );

    expect(screen.getByLabelText("Filter runs")).toBeVisible();

    rerender(<RunsTabPanel runs={runs} selectedRunId="run-2" detailDismissedForRunId="run-1" {...baseProps} />);
    expect(screen.queryByTestId("run-detail-panel")).not.toBeInTheDocument();
    expect(screen.getByText("Second run")).toBeInTheDocument();

    rerender(<RunsTabPanel runs={runs} selectedRunId="run-1" detailDismissedForRunId="run-1" {...baseProps} />);
    expect(screen.getByLabelText("Filter runs")).toBeVisible();
  });
});
