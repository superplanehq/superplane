import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderBoardLane, WorkOrderKanbanBoard } from "./WorkOrderBoardChrome";

describe("WorkOrderKanbanBoard", () => {
  it("scrolls extra lanes on the x axis", () => {
    render(
      <WorkOrderKanbanBoard testId="kanban-board">
        <WorkOrderBoardLane
          title="Plan"
          count={0}
          emptyDescription="Nothing here."
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
        emptyDescription="Nothing here."
        testId="lane-empty"
      />,
    );

    const emptyCopy = screen.getByText("Nothing here.");
    expect(emptyCopy).toBeInTheDocument();
    expect(emptyCopy.className).toContain("flex-1");
    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  it("renders custom empty content instead of the dashed copy", () => {
    render(
      <WorkOrderBoardLane
        title="Backlog"
        count={0}
        emptyDescription="No work orders in the backlog."
        emptyContent={<p data-testid="custom-empty">Tickets land here first.</p>}
        testId="lane-hint"
      />,
    );

    expect(screen.getByTestId("custom-empty")).toHaveTextContent("Tickets land here first.");
    expect(screen.queryByText("No work orders in the backlog.")).not.toBeInTheDocument();
  });

  it("renames the lane title on Enter when canRename is set", async () => {
    const onRename = vi.fn();
    const user = userEvent.setup();
    render(
      <WorkOrderBoardLane
        title="Backlog"
        count={0}
        emptyDescription="No work orders in the backlog."
        canRename
        onRename={onRename}
        titleTestId="lane-title"
        testId="lane-backlog"
      />,
    );

    await user.click(screen.getByTestId("lane-title"));
    const input = await screen.findByTestId("lane-title-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Inbox");
    await user.keyboard("{Enter}");

    expect(onRename).toHaveBeenCalledWith("Inbox");
  });
});
