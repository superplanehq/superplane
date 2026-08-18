import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { FACTORY_VELOCITY_BY_PERIOD, FACTORY_VELOCITY_YESTERDAY } from "./factoryVelocityMockData";
import { FACTORY_VELOCITY_FLOW_BY_PERIOD } from "./factoryVelocityFlowMockData";
import { VelocityPage } from "./VelocityPage";

function renderWithFactoriesLayout(ui: ReactElement) {
  return render(
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org-1",
        factoryId: PRIMARY_FACTORY_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        factory: REFUND_FACTORY,
        factories: [REFUND_FACTORY],
        openCreateWorkOrder: vi.fn(),
      }}
    >
      {ui}
    </FactoriesLayoutContext.Provider>,
  );
}

describe("VelocityPage", () => {
  it("sets the document title from the page and workspace name", () => {
    renderWithFactoriesLayout(<VelocityPage />);

    expect(document.title).toBe(`Velocity · ${REFUND_FACTORY.name} · SuperPlane`);
  });

  it("shows yesterday metrics, period pills, trend, and source split", () => {
    renderWithFactoriesLayout(<VelocityPage />);

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

    expect(screen.queryByTestId("velocity-work-order-flow")).not.toBeInTheDocument();
  });

  it("updates trend totals when the period changes and keeps yesterday fixed", async () => {
    const user = userEvent.setup();
    renderWithFactoriesLayout(<VelocityPage />);

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

  it("shows work-order time metrics when includeWorkOrderFlow is set", () => {
    renderWithFactoriesLayout(<VelocityPage includeWorkOrderFlow />);

    const flow = screen.getByTestId("velocity-work-order-flow");
    const period = FACTORY_VELOCITY_FLOW_BY_PERIOD[7];

    expect(flow).toHaveTextContent("Work order time");
    expect(flow).not.toHaveTextContent("Closed work orders");
    expect(flow).toHaveTextContent("Last 7 days");
    expect(flow).toHaveTextContent("Cycle time");
    expect(flow).toHaveTextContent("Time running");
    expect(flow).toHaveTextContent("Time in Waiting");
    expect(flow).toHaveTextContent(`${period.runningShareOfCyclePct}% of cycle time`);
    expect(flow).toHaveTextContent(`${period.waitingShareOfCyclePct}% of cycle time`);
    expect(flow).toHaveTextContent("Time running and Time in Waiting by day");
    expect(flow).not.toHaveTextContent("Work orders in Waiting");
    expect(flow).not.toHaveTextContent("Open in Waiting");
    expect(flow).not.toHaveTextContent("Waiting now");
    expect(flow).not.toHaveTextContent("Median age");
    expect(flow).not.toHaveTextContent("Lead time");
  });

  it("updates closed-period metrics when the period changes", async () => {
    const user = userEvent.setup();
    renderWithFactoriesLayout(<VelocityPage includeWorkOrderFlow />);

    expect(screen.getByTestId("velocity-work-order-flow")).toHaveTextContent(
      `${FACTORY_VELOCITY_FLOW_BY_PERIOD[7].waitingShareOfCyclePct}% of cycle time`,
    );

    await user.click(screen.getByRole("tab", { name: "30d" }));

    const flow = screen.getByTestId("velocity-work-order-flow");
    expect(flow).toHaveTextContent("Last 30 days");
    expect(flow).toHaveTextContent(`${FACTORY_VELOCITY_FLOW_BY_PERIOD[30].waitingShareOfCyclePct}% of cycle time`);
  });
});
