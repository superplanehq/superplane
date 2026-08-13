import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { LineVelocityPanel } from "./LineVelocityPanel";
import { VELOCITY_BY_PERIOD } from "./lineVelocityMockData";

describe("LineVelocityPanel", () => {
  it("shows Cursor-style period pills, hero run stats, cost rows, and daily velocity", () => {
    render(<LineVelocityPanel />);

    expect(screen.getByTestId("line-velocity-panel")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Velocity" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "7d" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "30d" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "90d" })).toBeInTheDocument();

    const stats = screen.getByTestId("velocity-run-stats");
    expect(stats).toHaveTextContent(String(VELOCITY_BY_PERIOD[7].runs));
    expect(stats).toHaveTextContent("Succeeded");
    expect(stats).toHaveTextContent("Failed");
    expect(stats).toHaveTextContent("Daily velocity");

    const cost = screen.getByTestId("velocity-cost-per-run");
    expect(cost).toHaveTextContent("Tokens");
    expect(cost).toHaveTextContent("VM spend");

    expect(screen.queryByRole("heading", { name: "Stage success" })).not.toBeInTheDocument();
  });

  it("updates run stats when the period changes", async () => {
    const user = userEvent.setup();
    render(<LineVelocityPanel />);

    await user.click(screen.getByRole("tab", { name: "30d" }));

    expect(screen.getByRole("tab", { name: "30d" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("velocity-run-stats")).toHaveTextContent(String(VELOCITY_BY_PERIOD[30].runs));
    expect(screen.getByTestId("velocity-run-stats")).toHaveTextContent("Last 30 days");
  });
});
