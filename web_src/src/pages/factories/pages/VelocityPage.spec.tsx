import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { FACTORY_VELOCITY_BY_PERIOD, FACTORY_VELOCITY_YESTERDAY } from "./factoryVelocityMockData";
import { VelocityPage } from "./VelocityPage";

describe("VelocityPage", () => {
  it("shows yesterday metrics, period pills, trend, and source split", () => {
    render(<VelocityPage />);

    expect(screen.getByRole("heading", { name: "Velocity" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "7d" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "30d" })).toBeInTheDocument();

    const yesterday = screen.getByTestId("velocity-yesterday");
    expect(yesterday).toHaveTextContent("Yesterday");
    expect(yesterday).toHaveTextContent(String(FACTORY_VELOCITY_YESTERDAY.merged));
    expect(yesterday).toHaveTextContent("Merged PRs");
    expect(yesterday).toHaveTextContent("Waste");
    expect(yesterday).toHaveTextContent("Cost");
    expect(yesterday).toHaveTextContent("Cost per merged PR");

    const trend = screen.getByTestId("velocity-trend");
    expect(trend).toHaveTextContent("Last 7 days");
    expect(trend).toHaveTextContent(String(FACTORY_VELOCITY_BY_PERIOD[7].totals.merged));
    expect(trend).toHaveTextContent("SuperPlane output");

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("Merged PRs by source");
    expect(split).toHaveTextContent("SuperPlane");
    expect(split).toHaveTextContent(`${FACTORY_VELOCITY_BY_PERIOD[7].totals.superplaneSharePct}%`);
  });

  it("updates trend totals when the period changes and keeps yesterday fixed", async () => {
    const user = userEvent.setup();
    render(<VelocityPage />);

    const yesterdayMerged = FACTORY_VELOCITY_YESTERDAY.merged;
    expect(screen.getByTestId("velocity-yesterday")).toHaveTextContent(String(yesterdayMerged));

    await user.click(screen.getByRole("tab", { name: "30d" }));

    expect(screen.getByRole("tab", { name: "30d" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("velocity-trend")).toHaveTextContent("Last 30 days");
    expect(screen.getByTestId("velocity-trend")).toHaveTextContent(
      String(FACTORY_VELOCITY_BY_PERIOD[30].totals.merged),
    );
    expect(screen.getByTestId("velocity-yesterday")).toHaveTextContent(String(yesterdayMerged));
  });
});
