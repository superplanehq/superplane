import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { SettingsAutomationRunsSidebar } from "./SettingsAutomationRunsSidebar";

const { useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns,
}));

function makeRun(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    id: "run-1",
    canvasId: "app-github-issues-intake",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: "2026-05-01T12:00:00Z",
    rootEvent: {
      id: "event-1",
      nodeId: "trigger",
      customName: "feat: Add console empty-state",
      createdAt: "2026-05-01T12:00:00Z",
    },
    executions: [],
    ...overrides,
  };
}

function mockRuns(runs: CanvasesCanvasRun[]) {
  useInfiniteCanvasRuns.mockReturnValue({
    data: { pages: [{ runs, totalCount: runs.length }] },
    isPending: false,
    isError: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
    refetch: vi.fn(),
  });
}

const workflowNodes: SuperplaneComponentsNode[] = [
  { id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "webhook" },
];

function renderSidebar(props: Partial<Parameters<typeof SettingsAutomationRunsSidebar>[0]> = {}) {
  return render(
    <MemoryRouter>
      <div className="flex h-[32rem]">
        <SettingsAutomationRunsSidebar
          canvasId="app-github-issues-intake"
          workflowNodes={workflowNodes}
          selectedRunId={null}
          onSelectRun={vi.fn()}
          {...props}
        />
      </div>
    </MemoryRouter>,
  );
}

afterEach(() => {
  localStorage.clear();
});

describe("SettingsAutomationRunsSidebar", () => {
  it("lists canvas runs beside the automation", () => {
    mockRuns([
      makeRun(),
      makeRun({
        id: "run-2",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        rootEvent: { id: "event-2", nodeId: "trigger", customName: "On Issue Labeled" },
      }),
    ]);

    renderSidebar();

    expect(useInfiniteCanvasRuns).toHaveBeenCalledWith("app-github-issues-intake", {}, true);
    const sidebar = screen.getByTestId("canvas-runs-sidebar");
    expect(within(sidebar).getByText("Runs")).toBeInTheDocument();
    expect(within(sidebar).getByLabelText("Search runs")).toBeInTheDocument();
    expect(within(sidebar).getByText("feat: Add console empty-state")).toBeInTheDocument();
    expect(within(sidebar).getByText("On Issue Labeled")).toBeInTheDocument();
    expect(within(sidebar).getByText("Passed")).toBeInTheDocument();
    expect(within(sidebar).getByText("Running")).toBeInTheDocument();
    expect(within(sidebar).queryByRole("button", { name: "Live Canvas" })).not.toBeInTheDocument();
  });

  it("keeps the selected run in the sidebar without leaving the page", async () => {
    const user = userEvent.setup();
    mockRuns([makeRun()]);

    function ControlledSidebar() {
      const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
      return (
        <SettingsAutomationRunsSidebar
          canvasId="app-github-issues-intake"
          workflowNodes={workflowNodes}
          selectedRunId={selectedRunId}
          onSelectRun={setSelectedRunId}
          runHrefFor={(runId) => `/org-1/workspaces/RF/apps/app-github-issues-intake?run=${runId}&from=lines`}
        />
      );
    }

    render(
      <MemoryRouter>
        <div className="flex h-[32rem]">
          <ControlledSidebar />
        </div>
      </MemoryRouter>,
    );

    const row = screen.getByTestId("runs-sidebar-row");
    expect(within(row).getByRole("link", { name: "feat: Add console empty-state" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?run=run-1&from=lines",
    );

    await user.click(within(row).getByRole("link", { name: "feat: Add console empty-state" }));

    expect(row).toHaveClass("bg-sky-100");
    expect(screen.queryByRole("button", { name: "Live Canvas" })).not.toBeInTheDocument();
  });

  it("shows an empty state when the canvas has no runs", () => {
    mockRuns([]);

    renderSidebar();

    expect(screen.getByText("No Runs")).toBeInTheDocument();
  });
});
