import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { MobileBoardPage } from "./MobileBoardPage";
import { MobileMorePage } from "./MobileMorePage";

export const DRAFT_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-draft",
  number: "42",
  title: "Fix refund rounding",
  state: "STATE_DRAFT",
  lineDispatches: [],
};

export function stubElementHeights(measurements: { scrollHeight: number; clientHeight: number }) {
  for (const [name, value] of Object.entries(measurements)) {
    Object.defineProperty(HTMLElement.prototype, name, { configurable: true, value });
  }
  return () => {
    for (const name of Object.keys(measurements)) Reflect.deleteProperty(HTMLElement.prototype, name);
  };
}

export function createMobileBoardRenderer(openCreateWorkOrder: () => void, LocationProbe: () => ReactNode) {
  function boardElement(factory: FactoriesFactory = REFUND_FACTORY, more = false) {
    return (
      <QueryClientProvider client={new QueryClient()}>
        <ThemeProvider>
          <TooltipProvider>
            <MemoryRouter
              initialEntries={[
                more
                  ? `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/more?lineId=${REFUND_LINE_PLAN_ID}`
                  : `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
              ]}
            >
              <FactoriesLayoutContext.Provider
                value={{
                  organizationId: "org-1",
                  factoryId: PRIMARY_FACTORY_ID,
                  factoryKey: PRIMARY_FACTORY_KEY,
                  routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                  factory,
                  factories: [factory],
                  openCreateWorkOrder,
                }}
              >
                <Routes>
                  <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<MobileBoardPage />} />
                  <Route path="/org-1/workspaces/:factoryKey/more" element={<MobileMorePage />} />
                  <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<div>Task page</div>} />
                </Routes>
                <LocationProbe />
              </FactoriesLayoutContext.Provider>
            </MemoryRouter>
          </TooltipProvider>
        </ThemeProvider>
      </QueryClientProvider>
    );
  }

  function renderBoard(factory: FactoriesFactory = REFUND_FACTORY, more = false) {
    return render(boardElement(factory, more));
  }

  return { boardElement, renderBoard };
}
