import { fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasNodeExecution, SuperplaneMeUser } from "@/api-client";
import { executions, renderInspector } from "./RunInspectorPanel.spec.fixtures";

let mockedExecutions = executions;
let mockedMe: SuperplaneMeUser | null = null;

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value, collapsed }: { value: unknown; collapsed?: boolean | number }) => (
    <pre data-testid="json-view" data-collapsed={String(collapsed)}>
      {JSON.stringify(value)}
    </pre>
  ),
}));

const reemitTriggerEventMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEventMock(...args),
    canvasesCancelExecution: vi.fn(),
    canvasesInvokeNodeExecutionHook: vi.fn(),
    canvasesDescribeRun: vi.fn().mockResolvedValue({ data: { run: { queueItems: [] } } }),
    canvasesListNodeQueueItems: vi.fn().mockResolvedValue({ data: { items: [] } }),
    canvasesDeleteNodeQueueItem: vi.fn(),
  };
});

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
  useMe: () => ({ data: mockedMe }),
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

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => false, isLoading: false }),
}));

beforeEach(() => {
  mockedExecutions = executions;
  mockedMe = null;
  reemitTriggerEventMock.mockResolvedValue({});
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("RunInspectorPanel without canvases:update permission", () => {
  it("disables the global Restart button and never fires a request when clicked", async () => {
    renderInspector();

    const restartButtons = screen.getAllByRole("button", { name: /Rerun/i });
    const globalRestartButton = restartButtons[0];
    expect(globalRestartButton).toBeDisabled();

    fireEvent.click(globalRestartButton);

    expect(reemitTriggerEventMock).not.toHaveBeenCalled();
  });

  it("disables the per-trigger inline Rerun button inside the node accordion", () => {
    renderInspector({ selectedNodeId: "trigger-1" });

    const restartButtons = screen.getAllByRole("button", { name: /Rerun/i });
    expect(restartButtons.length).toBeGreaterThan(1);
    restartButtons.forEach((button) => expect(button).toBeDisabled());
  });
});
