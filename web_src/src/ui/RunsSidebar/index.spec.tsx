import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { RunsSidebar } from ".";

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

describe("RunsSidebar", () => {
  it("shows an empty state when there are no runs", () => {
    render(<RunsSidebar runs={[]} selectedRunId={null} onSelectRun={() => {}} workflowNodes={nodes} />);

    expect(screen.getByText("No runs yet")).toBeInTheDocument();
  });

  it("pins running runs above completed runs", () => {
    render(
      <RunsSidebar
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
        onSelectRun={() => {}}
        workflowNodes={nodes}
      />,
    );

    const rows = screen.getAllByRole("button").filter((button) => within(button).queryByText(/run$/i));
    expect(within(rows[0]).getByText("Running run")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Completed run")).toBeInTheDocument();
  });

  it("filters runs by search text and status", () => {
    render(
      <RunsSidebar
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
        onSelectRun={() => {}}
        workflowNodes={nodes}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText("Search runs..."), { target: { value: "broken" } });
    expect(screen.getByText("Broken deploy")).toBeInTheDocument();
    expect(screen.queryByText("Healthy deploy")).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("Search runs..."), { target: { value: "" } });
    fireEvent.click(screen.getByLabelText("Filter runs"));
    expect(screen.getByText("Passed")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Failed"));
    expect(screen.getByText("Cancelled")).toBeInTheDocument();
    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.queryByText("Completed")).not.toBeInTheDocument();

    expect(screen.getByText("Broken deploy")).toBeInTheDocument();
    expect(screen.queryByText("Healthy deploy")).not.toBeInTheDocument();
  });
});
