import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesLineMetrics } from "@/api-client";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_METRICS,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LinesPage } from "./LinesPage";

const { lineMetricsQuery, workOrdersQuery } = vi.hoisted(() => ({
  lineMetricsQuery: { data: undefined as Record<string, FactoriesLineMetrics> | undefined, error: null as Error | null },
  workOrdersQuery: { data: [] as unknown[] },
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => workOrdersQuery,
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryLineMetrics", () => ({
  useFactoryLineMetrics: () => lineMetricsQuery,
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

function renderLinesPage() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[`/w/${PRIMARY_FACTORY_KEY}/lines`]}>
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
          <Routes>
            <Route path="w/:factoryKey/lines" element={<LinesPage />} />
          </Routes>
        </FactoriesLayoutContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("LinesPage", () => {
  beforeEach(() => {
    lineMetricsQuery.data = undefined;
    lineMetricsQuery.error = null;
  });

  it("renders the list immediately with dashes while metrics are still loading", () => {
    renderLinesPage();

    expect(screen.getByTestId("lines-list")).toBeInTheDocument();
    const metricsRows = screen.getAllByTestId("lines-card-metrics");
    expect(metricsRows.length).toBeGreaterThan(0);
    for (const row of metricsRows) {
      expect(row).toHaveTextContent("—");
    }
  });

  it("shows success/completions/rework/cost once the metrics query resolves", () => {
    lineMetricsQuery.data = { [REFUND_LINE_PLAN_METRICS.lineId as string]: REFUND_LINE_PLAN_METRICS };
    renderLinesPage();

    const card = screen.getByTestId(`lines-card-${REFUND_LINE_PLAN_METRICS.lineId}`);
    expect(card).toHaveTextContent("87%");
    expect(card).toHaveTextContent("0.7 / day");
    expect(card).toHaveTextContent("1.3 / order");
    expect(card).toHaveTextContent("$8.42");
  });

  it("renders dashes for a line absent from the metrics response", () => {
    // Only the plan-and-implement line has metrics; hotfix has none.
    lineMetricsQuery.data = { [REFUND_LINE_PLAN_METRICS.lineId as string]: REFUND_LINE_PLAN_METRICS };
    renderLinesPage();

    const lines = REFUND_FACTORY.lines ?? [];
    const hotfixLine = lines.find((line) => line.id !== REFUND_LINE_PLAN_METRICS.lineId);
    expect(hotfixLine).toBeDefined();

    const card = screen.getByTestId(`lines-card-${hotfixLine!.id}`);
    expect(card).toHaveTextContent("—");
  });

  it("degrades to dashes rather than breaking the page when the metrics endpoint errors", () => {
    lineMetricsQuery.data = undefined;
    lineMetricsQuery.error = new Error("network");
    renderLinesPage();

    expect(screen.getByTestId("lines-list")).toBeInTheDocument();
    const metricsRows = screen.getAllByTestId("lines-card-metrics");
    for (const row of metricsRows) {
      expect(row).toHaveTextContent("—");
    }
  });
});
