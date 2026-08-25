import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { executions, renderInspector } from "./RunInspectorPanel.spec.fixtures";

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value, collapsed }: { value: unknown; collapsed?: boolean | number }) => (
    <pre data-testid="json-view" data-collapsed={String(collapsed)}>
      {JSON.stringify(value)}
    </pre>
  ),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReemitTriggerEvent: vi.fn(),
    canvasesCancelExecution: vi.fn(),
    canvasesInvokeNodeExecutionHook: vi.fn(),
    canvasesDescribeRun: vi.fn(),
    canvasesListNodeQueueItems: vi.fn(),
    canvasesDeleteNodeQueueItem: vi.fn(),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions },
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

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

describe("RunInspectorPanel factory timeline", () => {
  it("opens factory node timeline cards for the selected node", () => {
    renderInspector({
      factoryContext: true,
      selectedNodeId: "action-2",
    });

    const panel = screen.getByTestId("run-inspector-panel");
    expect(panel).toHaveAttribute("data-factory-context", "true");
    expect(screen.getByTestId("factory-run-node-detail")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Input/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Runtime Config/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Output/i })).toBeInTheDocument();
  });
});
