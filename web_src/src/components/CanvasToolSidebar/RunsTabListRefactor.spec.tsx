import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
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

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

afterEach(() => {
  vi.useRealTimers();
  localStorage.clear();
});

const nodes: SuperplaneComponentsNode[] = [
  {
    id: "trigger-1",
    name: "Deploy Trigger",
    type: "TYPE_TRIGGER",
    component: "webhook",
  },
];

const baseProps = {
  canvasId: "canvas-1",
  onSelectRun: () => {},
  onSelectLiveCanvas: () => {},
  workflowNodes: nodes,
};

function makeRun(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "canvas-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-01T12:00:00Z",
    finishedAt: "2026-05-01T12:00:01Z",
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

describe("RunsTabList refactor", () => {
  it("filters runs by search text", async () => {
    const user = userEvent.setup();

    render(
      <RunsTabPanel
        runs={[
          makeRun({
            id: "run-deploy",
            rootEvent: { ...makeRun().rootEvent, customName: "Deploy production" },
          }),
          makeRun({
            id: "run-release",
            rootEvent: { ...makeRun().rootEvent, customName: "Release candidate" },
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
      { wrapper: routerWrapper },
    );

    await user.type(screen.getByLabelText("Search runs"), "release");

    expect(screen.queryByText("Deploy production")).not.toBeInTheDocument();
    expect(screen.getByText("Release candidate")).toBeInTheDocument();
    expect(screen.getByText("Showing 1 of 2 loaded")).toBeInTheDocument();
  });

  it("shows redesigned run row details", () => {
    render(
      <RunsTabPanel
        runs={[
          makeRun({
            result: "RESULT_FAILED",
            rootEvent: { ...makeRun().rootEvent, customName: "Broken deploy" },
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
      { wrapper: routerWrapper },
    );

    const row = screen.getByTestId("runs-sidebar-row");
    expect(within(row).getByText("Broken deploy")).toBeInTheDocument();
    expect(within(row).getByText("Failed")).toBeInTheDocument();
    expect(within(row).getByText("Deploy Trigger")).toBeInTheDocument();
    expect(within(row).getByText("1s")).toBeInTheDocument();
    expect(row.querySelector('time[datetime="2026-05-01T12:00:00.000Z"]')).toBeInTheDocument();
  });

  it("updates running run duration while the row is visible", () => {
    vi.useFakeTimers();
    const now = new Date();
    vi.setSystemTime(now);
    const startedAt = new Date(now.getTime() - 1_000).toISOString();

    render(
      <RunsTabPanel
        runs={[
          makeRun({
            id: "run-running",
            state: "STATE_STARTED",
            result: "RESULT_UNKNOWN",
            createdAt: startedAt,
            finishedAt: undefined,
            rootEvent: { ...makeRun().rootEvent, customName: "Running deploy" },
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
      { wrapper: routerWrapper },
    );

    expect(screen.getByText("1s")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(2_000);
    });

    expect(screen.getByText("3s")).toBeInTheDocument();
  });

  it("formats sub-second run durations without milliseconds", () => {
    render(
      <RunsTabPanel
        runs={[
          makeRun({
            createdAt: "2026-05-01T12:00:00.000Z",
            finishedAt: "2026-05-01T12:00:00.250Z",
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
      { wrapper: routerWrapper },
    );

    const row = screen.getByTestId("runs-sidebar-row");
    expect(within(row).getByText("<1s")).toBeInTheDocument();
    expect(within(row).queryByText("250ms")).not.toBeInTheDocument();
  });

  it("formats run durations with minutes and seconds only", () => {
    render(
      <RunsTabPanel
        runs={[
          makeRun({
            createdAt: "2026-05-01T12:00:00.000Z",
            finishedAt: "2026-05-01T13:01:30.500Z",
          }),
        ]}
        selectedRunId={null}
        {...baseProps}
      />,
      { wrapper: routerWrapper },
    );

    const row = screen.getByTestId("runs-sidebar-row");
    expect(within(row).getByText("61m 30s")).toBeInTheDocument();
    expect(within(row).queryByText(/ms/)).not.toBeInTheDocument();
    expect(within(row).queryByText(/h/)).not.toBeInTheDocument();
  });
});
