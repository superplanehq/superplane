import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[], isLoading: false }));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: (...args: unknown[]) => useFactoryWorkOrders(...(args as [])),
}));

import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LegacyWorkOrderDetailRedirect } from "./WorkOrderDetailPage";

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{`${location.pathname}${location.search}`}</div>;
}

const ORDER: FactoriesWorkOrder = { id: "wo-open-refunds", number: "842", title: "Reconcile refunds" };

function renderAt(path: string) {
  return render(
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org-1",
        factoryId: "factory-1",
        factoryKey: "SP",
        factory: { id: "factory-1", key: "SP", lines: [{ id: "line-hotfix" }] } as FactoriesFactory,
        factories: [],
        openCreateWorkOrder: () => {},
      }}
    >
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/org-1/workspaces/:factoryKey/tasks/:orderId" element={<LegacyWorkOrderDetailRedirect />} />
        </Routes>
        <LocationProbe />
      </MemoryRouter>
    </FactoriesLayoutContext.Provider>,
  );
}

describe("LegacyWorkOrderDetailRedirect", () => {
  it("canonicalizes a legacy id onto the task number permalink", () => {
    useFactoryWorkOrders.mockReturnValue({ data: [ORDER], isLoading: false });
    renderAt("/org-1/workspaces/SP/tasks/wo-open-refunds");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/task/842");
  });

  it("preserves the lineId query through the id-to-number redirect", () => {
    useFactoryWorkOrders.mockReturnValue({ data: [ORDER], isLoading: false });
    renderAt("/org-1/workspaces/SP/tasks/wo-open-refunds?lineId=line-hotfix");
    expect(screen.getByTestId("location")).toHaveTextContent("/org-1/workspaces/SP/task/842?lineId=line-hotfix");
  });
});
