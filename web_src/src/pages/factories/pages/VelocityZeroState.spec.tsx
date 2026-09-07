import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import { VelocityZeroState } from "./VelocityZeroState";

function renderZeroState(tasksHref = "/org-1/workspaces/refunds/work-orders") {
  return render(
    <MemoryRouter>
      <VelocityZeroState tasksHref={tasksHref} />
    </MemoryRouter>,
  );
}

describe("VelocityZeroState", () => {
  it("explains what appears here after the first tasks close", () => {
    renderZeroState();

    expect(screen.getByTestId("velocity-zero-state")).toBeInTheDocument();
    expect(screen.getByText("No velocity data yet")).toBeInTheDocument();
    expect(
      screen.getByText("Velocity reports merged pull requests, task time, and cost after your first tasks close."),
    ).toBeInTheDocument();
  });

  it("links to the Tasks board instead of offering to create work from a report", () => {
    renderZeroState("/org-1/workspaces/refunds/work-orders");

    const link = screen.getByTestId("velocity-zero-state-tasks");
    expect(link).toHaveAttribute("href", "/org-1/workspaces/refunds/work-orders");
    expect(link).toHaveTextContent("View tasks");
    expect(screen.queryByRole("button", { name: "New task" })).not.toBeInTheDocument();
  });
});
