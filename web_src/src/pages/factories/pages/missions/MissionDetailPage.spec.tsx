import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../../layout/factoriesLayoutContext";
import { CHECKOUT_RELIABILITY_MISSION } from "./missionMocks";
import { MissionDetailPage } from "./MissionDetailPage";

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({
    data: [],
    isLoading: false,
    error: new Error("network"),
    refetch: vi.fn(),
  }),
}));

describe("MissionDetailPage", () => {
  it("shows a recoverable error when tasks fail to load", () => {
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter initialEntries={[`/missions/${CHECKOUT_RELIABILITY_MISSION.id}`]}>
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
              <Route path="missions/:missionId" element={<MissionDetailPage />} />
            </Routes>
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getByTestId("work-orders-error-state")).toBeInTheDocument();
    expect(screen.queryByText("This mission has no tasks.")).not.toBeInTheDocument();
  });
});
