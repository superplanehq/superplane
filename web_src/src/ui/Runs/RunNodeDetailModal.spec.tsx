import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { RunNodeDetailModal } from "./RunNodeDetailModal";

vi.mock("@uiw/react-json-view", () => ({
  default: () => <div data-testid="json-view" />,
}));

vi.mock("@/components/TimeAgo", () => ({
  TimeAgo: () => <span>time ago</span>,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: {
      executions: [
        {
          nodeId: "node-1",
          createdAt: "2026-05-18T12:00:00Z",
          outputs: {},
          metadata: {},
          configuration: {},
        },
      ],
    },
  }),
}));

vi.mock("@/pages/app/mappers", () => ({
  getExecutionDetails: () => ({
    "Workflow URL": "https://semaphore.example/workflows/123",
  }),
  getState: () => () => "success",
  getStateMap: () => ({
    success: { badgeColor: "bg-emerald-500", label: "passed" },
    triggered: { badgeColor: "bg-blue-500", label: "triggered" },
  }),
}));

vi.mock("@/pages/app/utils", () => ({
  buildExecutionInfo: (execution: unknown) => execution,
}));

describe("RunNodeDetailModal", () => {
  it("renders URL details as clickable links", () => {
    const run = {
      id: "run-1",
      rootEvent: {
        id: "root-event-1",
        nodeId: "trigger-node",
      },
    } as CanvasesCanvasRun;

    const workflowNodes = [
      {
        id: "node-1",
        name: "Semaphore",
        component: "semaphore.run_workflow",
        type: "TYPE_ACTION",
      },
    ] as SuperplaneComponentsNode[];

    render(
      <RunNodeDetailModal
        canvasId="canvas-1"
        run={run}
        nodeId="node-1"
        workflowNodes={workflowNodes}
        onClose={vi.fn()}
      />,
      { wrapper: ThemeProvider },
    );

    const link = screen.getByRole("link", { name: "https://semaphore.example/workflows/123" });
    expect(link).toHaveAttribute("href", "https://semaphore.example/workflows/123");
    expect(link).toHaveAttribute("target", "_blank");
  });
});
