import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { FactoryPreviewFlagsContext, type FactoryPreviewFlags } from "./factoryPreviewFlagsContext";
import { LinesPage } from "./LinesPage";

export function LocationProbe() {
  const location = useLocation();
  return <div data-testid="lines-test-location">{location.pathname}</div>;
}

export function LinesListSpecHarness({ factory = REFUND_FACTORY }: { factory?: FactoriesFactory }) {
  return (
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={[`/${PRIMARY_FACTORY_KEY}/lines`]}>
        <FactoriesLayoutContext.Provider
          value={{
            organizationId: "org-1",
            factoryId: PRIMARY_FACTORY_ID,
            factoryKey: PRIMARY_FACTORY_KEY,
            factory,
            factories: [factory],
            openCreateWorkOrder: vi.fn(),
          }}
        >
          <LinesPage />
          <LocationProbe />
        </FactoriesLayoutContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

export function LinesBoardSpecHarness({
  path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
  openCreateWorkOrder = vi.fn(),
  factory = REFUND_FACTORY,
  previewFlags = null,
}: {
  path?: string;
  openCreateWorkOrder?: ReturnType<typeof vi.fn>;
  factory?: FactoriesFactory;
  previewFlags?: FactoryPreviewFlags | null;
}) {
  return (
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter initialEntries={[path]}>
            <FactoryPreviewFlagsContext.Provider value={previewFlags}>
              <FactoriesLayoutContext.Provider
                value={{
                  organizationId: "org-1",
                  factoryId: factory.id ?? PRIMARY_FACTORY_ID,
                  factoryKey: factory.key ?? PRIMARY_FACTORY_KEY,
                  factory,
                  factories: [factory],
                  openCreateWorkOrder,
                }}
              >
                <Routes>
                  <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LinesPage />} />
                  <Route path="/org-1/workspaces/:factoryKey/lines/:lineId/edit" element={<div>Edit line</div>} />
                </Routes>
                <LocationProbe />
              </FactoriesLayoutContext.Provider>
            </FactoryPreviewFlagsContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
