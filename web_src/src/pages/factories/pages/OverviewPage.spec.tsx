import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { OverviewMetricsScorecardRowPopulated } from "./OverviewMetricsScorecardRow";
import { OverviewMetricsSlotContext } from "./overviewMetricsSlots";
import { OverviewPage } from "./OverviewPage";
import { buildOverviewVelocitySummary } from "./overviewVelocitySummary";

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({
    data: [],
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
}));

function renderOverview(withMetricsSlot: boolean) {
  const tree = (
    <MemoryRouter initialEntries={["/overview"]}>
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
        <OverviewPage />
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>
  );

  if (!withMetricsSlot) {
    return render(tree);
  }

  return render(
    <OverviewMetricsSlotContext.Provider value={OverviewMetricsScorecardRowPopulated}>
      {tree}
    </OverviewMetricsSlotContext.Provider>,
  );
}

describe("OverviewPage", () => {
  it("keeps today's two-column layout with the Automations box when the metrics slot is empty", () => {
    renderOverview(false);

    expect(screen.getByTestId("overview-work-orders-card")).toBeInTheDocument();
    expect(screen.getByTestId("overview-lines-card")).toBeInTheDocument();
    expect(screen.queryByTestId("overview-metrics-row")).not.toBeInTheDocument();
  });

  it("shows the velocity scorecard row above Work Orders and drops Automations when the slot is filled", () => {
    renderOverview(true);

    const metricsRow = screen.getByTestId("overview-metrics-row");
    const summary = buildOverviewVelocitySummary();

    expect(metricsRow).toHaveTextContent("Merged PRs");
    expect(metricsRow).toHaveTextContent(String(summary.merged));
    expect(metricsRow).toHaveTextContent("Waste");
    expect(metricsRow).toHaveTextContent(`${summary.wastePct}% of SuperPlane output`);
    expect(metricsRow).toHaveTextContent("Cost");
    expect(metricsRow).toHaveTextContent(`${summary.superplaneSharePct}%`);
    expect(screen.getByTestId("overview-metrics-view-velocity")).toHaveAttribute(
      "href",
      `/${"org-1"}/workspaces/${PRIMARY_FACTORY_KEY}/velocity`,
    );

    expect(screen.getByTestId("overview-work-orders-card")).toBeInTheDocument();
    expect(screen.queryByTestId("overview-lines-card")).not.toBeInTheDocument();
  });
});
