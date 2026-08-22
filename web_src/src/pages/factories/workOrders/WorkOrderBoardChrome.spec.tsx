import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkOrderBoardLane, WorkOrderKanbanBoard } from "./WorkOrderBoardChrome";

describe("WorkOrderKanbanBoard", () => {
  it("scrolls extra lanes on the x axis", () => {
    render(
      <WorkOrderKanbanBoard testId="kanban-board">
        <WorkOrderBoardLane
          title="Plan"
          count={0}
          emptyDescription="No work orders in this phase."
          testId="lane-plan"
        />
      </WorkOrderKanbanBoard>,
    );

    expect(screen.getByTestId("kanban-board").className).toContain("overflow-x-auto");
    expect(screen.getByTestId("lane-plan").className).toContain("min-w-72");
    expect(screen.getByTestId("lane-plan").className).toContain("shrink-0");
    expect(screen.getByRole("heading", { name: "Plan" })).toBeInTheDocument();
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  it("keeps empty-lane copy and stretches the empty body", () => {
    render(
      <WorkOrderBoardLane
        title="Verify"
        count={0}
        emptyDescription="No work orders in this phase."
        testId="lane-empty"
      />,
    );

    const emptyCopy = screen.getByText("No work orders in this phase.");
    expect(emptyCopy).toBeInTheDocument();
    expect(emptyCopy.className).toContain("flex-1");
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });
});
