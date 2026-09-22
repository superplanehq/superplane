import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";

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
  FactoryAutomationRunsSidebar: () => <div data-testid="factory-automation-runs-sidebar" />,
}));

describe("SettingsAutomationWorkspace", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useInfiniteCanvasRuns.mockReturnValue({
      data: { pages: [{ runs: [] }] },
    });
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
