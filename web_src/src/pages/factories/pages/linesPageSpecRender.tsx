import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes, useLocation, useNavigate } from "react-router";

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
  const navigate = useNavigate();
  return (
    <>
      <div data-testid="lines-test-location">{`${location.pathname}${location.search}`}</div>
      <button type="button" data-testid="lines-test-back" onClick={() => navigate(-1)}>
        Back
      </button>
    </>
  );
}

export function LinesBoardSpecHarness({
  path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
  openCreateWorkOrder = () => {},
  factory = REFUND_FACTORY,
  previewFlags = null,
}: {
  path?: string;
  openCreateWorkOrder?: () => void;
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
                  <Route path="/org-1/workspaces/:factoryKey/task/:orderNumber" element={<LinesPage />} />
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
