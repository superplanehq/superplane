import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Outlet, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { factoryVelocityPath } from "../lib/factoryPagePaths";
import { MobileBottomBar } from "./MobileBottomBar";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada" } }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Acme" } } }),
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="mobile-test-location">{`${location.pathname}${location.search}`}</div>;
}

function renderBar(initialPath: string) {
  return render(
    <ThemeProvider>
      <TooltipProvider>
        <MemoryRouter initialEntries={[initialPath]}>
          <FactoriesLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: PRIMARY_FACTORY_ID,
              factoryKey: PRIMARY_FACTORY_KEY,
              routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
              factory: REFUND_FACTORY,
              factories: [REFUND_FACTORY],
              openCreateWorkOrder: vi.fn(),
            }}
          >
            <Routes>
              <Route
                path="/org-1/workspaces/:factoryKey/*"
                element={
                  <>
                    <MobileBottomBar canCreateWorkOrder={false} />
                    <Outlet />
                  </>
                }
              >
                <Route path="*" element={<LocationProbe />} />
              </Route>
            </Routes>
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </TooltipProvider>
    </ThemeProvider>,
  );
}

describe("MobileBottomBar", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("keeps the current line when Board is opened again", async () => {
    const user = userEvent.setup();
    const hotfixBoard = `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`;
    renderBar(hotfixBoard);

    expect(screen.getByTestId("mobile-tab-board")).toHaveAttribute("href", hotfixBoard);

    await user.click(screen.getByTestId("mobile-tab-velocity"));

    expect(screen.getByTestId("mobile-test-location")).toHaveTextContent(
      factoryVelocityPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT),
    );
    expect(screen.getByTestId("mobile-tab-board")).toHaveAttribute("href", hotfixBoard);
  });

  it("opens the line from a task URL", () => {
    renderBar(`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/task/42?lineId=${REFUND_LINE_HOTFIX_ID}`);

    expect(screen.getByTestId("mobile-tab-board")).toHaveAttribute(
      "href",
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_HOTFIX_ID}`,
    );
  });

  it("opens the first line when no line has been opened yet", () => {
    renderBar(factoryVelocityPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT));

    expect(screen.getByTestId("mobile-tab-board")).toHaveAttribute(
      "href",
      `/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/lines/${REFUND_LINE_PLAN_ID}`,
    );
  });
});
