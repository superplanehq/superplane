import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, useNavigate } from "react-router";

import { SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import { useAutomationRunNavigation } from "./useAutomationRunNavigation";

const { useCanvasRuntimeWebsocket, useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useCanvasRuntimeWebsocket: vi.fn(),
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns,
}));

vi.mock("./useSettingsAutomationRunCanvas", () => ({
  useSettingsAutomationRunCanvas: ({ liveGraph }: { liveGraph: { nodes: unknown[]; edges: unknown[] } }) => ({
    graph: liveGraph,
    isRunInspectionMode: false,
    runCanvasLoading: false,
    selectedRun: null,
    fitAllRequest: null,
  }),
}));

vi.mock("./SettingsAutomationCanvas", () => ({
  SettingsAutomationCanvas: () => <div data-testid="settings-automation-canvas" />,
}));

vi.mock("./factoryAutomationRunsSidebar/FactoryAutomationRunsSidebar", () => ({
  FactoryAutomationRunsSidebar: ({ onSelectRun }: { onSelectRun: (id: string) => void }) => (
    <button data-testid="select-older-run" onClick={() => onSelectRun("older")}>
      Older run
    </button>
  ),
}));

function NavigableWorkspace() {
  const { tab, setTab, selectedRunId, setSelectedRunId } = useAutomationRunNavigation();
  const navigate = useNavigate();
  return (
    <>
      <button data-testid="runs" onClick={() => setTab("runs")}>
        Runs
      </button>
      <button data-testid="automation" onClick={() => setTab("automation")}>
        Automation
      </button>
      <button data-testid="back" onClick={() => navigate(-1)}>
        Back
      </button>
      <button data-testid="forward" onClick={() => navigate(1)}>
        Forward
      </button>
      <output data-testid="view">
        {tab}:{selectedRunId ?? "index"}
      </output>
      {tab === "runs" && (
        <SettingsAutomationWorkspace
          graph={{ nodes: [], edges: [] }}
          canvasId="canvas"
          testId="workspace"
          selectLatestRun
          selectedRunId={selectedRunId}
          onSelectedRunIdChange={setSelectedRunId}
        />
      )}
    </>
  );
}

describe("SettingsAutomationWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useInfiniteCanvasRuns.mockReturnValue({
      data: { pages: [{ runs: [] }] },
    });
  });

  it("keeps Back and Forward intact after automatic and manual run selections", async () => {
    useInfiniteCanvasRuns.mockReturnValue({ data: { pages: [{ runs: [{ id: "latest" }, { id: "older" }] }] } });
    render(
      <MemoryRouter initialEntries={["/?automationView=canvas"]}>
        <NavigableWorkspace />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByTestId("runs"));
    await waitFor(() => expect(screen.getByTestId("view")).toHaveTextContent("runs:latest"));
    fireEvent.click(screen.getByTestId("select-older-run"));
    fireEvent.click(screen.getByTestId("automation"));
    for (const expected of ["runs:older", "runs:latest", "automation:index"]) {
      fireEvent.click(screen.getByTestId("back"));
      await waitFor(() => expect(screen.getByTestId("view")).toHaveTextContent(expected));
    }
    for (const expected of ["runs:latest", "runs:older", "automation:older"]) {
      fireEvent.click(screen.getByTestId("forward"));
      await waitFor(() => expect(screen.getByTestId("view")).toHaveTextContent(expected));
    }
  });

  it("subscribes to run updates for its canvas", () => {
    render(
      <SettingsAutomationWorkspace
        graph={{ nodes: [], edges: [], organizationId: "org-1", factoryId: "factory-1" }}
        testId="settings-automation"
        canvasId="canvas-1"
      />,
    );

    expect(useCanvasRuntimeWebsocket).toHaveBeenCalledWith("canvas-1", "org-1", true);
  });

  it("does not open a run subscription without a canvas", () => {
    render(
      <SettingsAutomationWorkspace
        graph={{ nodes: [], edges: [], organizationId: "org-1", factoryId: "factory-1" }}
        testId="settings-automation"
      />,
    );

    expect(useCanvasRuntimeWebsocket).toHaveBeenCalledWith("", "org-1", false);
  });
});
