import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "bun:test";

import type { CanvasesCanvasRun } from "@/api-client";
import type * as CanvasDataModule from "@/hooks/useCanvasData";

import { useSettingsAutomationRunCanvas } from "./useSettingsAutomationRunCanvas";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

const { useCanvas, useDescribeRun, useEventExecutions, useTriggers, useComponents, useAvailableIntegrations } =
  vi.hoisted(() => ({
    useCanvas: vi.fn(),
    useDescribeRun: vi.fn(),
    useEventExecutions: vi.fn(),
    useTriggers: vi.fn(),
    useComponents: vi.fn(),
    useAvailableIntegrations: vi.fn(),
  }));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal<typeof CanvasDataModule>();
  return {
    ...actual,
    useCanvas,
    useDescribeRun,
    useEventExecutions,
    useTriggers,
  };
});

vi.mock("@/hooks/useComponentData", () => ({
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useAvailableIntegrations,
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
}));

const liveGraph: IntakeAutomationGraph = {
  nodes: [{ id: "trigger", position: { x: 0, y: 0 }, data: { type: "trigger", label: "On Issue", state: "pending" } }],
  edges: [],
  factoryId: "factory-1",
  organizationId: "org-1",
};

const selectedRun: CanvasesCanvasRun = {
  id: "run-1",
  canvasId: "canvas-1",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  rootEvent: { id: "event-1", nodeId: "trigger", customName: "On mention on Issue" },
  executions: [
    {
      id: "exec-1",
      nodeId: "filter",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
    },
  ],
};

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      {children}
    </QueryClientProvider>
  );
}

describe("useSettingsAutomationRunCanvas", () => {
  it("keeps the live graph until a run is selected", () => {
    useCanvas.mockReturnValue({
      data: { metadata: { id: "canvas-1" }, spec: { nodes: [], edges: [] } },
      isPending: false,
    });
    useDescribeRun.mockReturnValue({ data: undefined, isLoading: false, isFetched: true });
    useEventExecutions.mockReturnValue({ data: { executions: [] }, isLoading: false });
    useTriggers.mockReturnValue({ data: [], isLoading: false });
    useComponents.mockReturnValue({ data: [], isLoading: false });
    useAvailableIntegrations.mockReturnValue({ data: [], isLoading: false });

    const { result } = renderHook(
      () =>
        useSettingsAutomationRunCanvas({
          organizationId: "org-1",
          canvasId: "canvas-1",
          selectedRunId: null,
          selectedRunFromList: null,
          liveGraph,
        }),
      { wrapper },
    );

    expect(result.current.isRunInspectionMode).toBe(false);
    expect(result.current.graph.nodes).toEqual(liveGraph.nodes);
    expect(result.current.selectedRun).toBeNull();
  });

  it("uses the selected list run as the canvas run", () => {
    useCanvas.mockReturnValue({
      data: {
        metadata: { id: "canvas-1", liveVersionId: "version-1" },
        spec: {
          nodes: [{ id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" }],
          edges: [],
        },
      },
      isPending: false,
    });
    useDescribeRun.mockReturnValue({ data: undefined, isLoading: false, isFetched: true });
    useEventExecutions.mockReturnValue({ data: { executions: [] }, isLoading: false });
    useTriggers.mockReturnValue({ data: [{ name: "github.onIssue", label: "On Issue" }], isLoading: false });
    useComponents.mockReturnValue({ data: [], isLoading: false });
    useAvailableIntegrations.mockReturnValue({ data: [], isLoading: false });

    const { result } = renderHook(
      () =>
        useSettingsAutomationRunCanvas({
          organizationId: "org-1",
          canvasId: "canvas-1",
          selectedRunId: "run-1",
          selectedRunFromList: selectedRun,
          liveGraph,
        }),
      { wrapper },
    );

    expect(result.current.isRunInspectionMode).toBe(true);
    expect(result.current.selectedRun).toEqual(selectedRun);
    expect(result.current.runParticipantNodeIds).toEqual(["trigger", "filter"]);
  });
});
