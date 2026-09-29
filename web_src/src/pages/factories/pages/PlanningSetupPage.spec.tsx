import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { factoryHomePath, factoryPlanningPath, firstFactoryLineId } from "../lib/factoryPagePaths";
import { PlanningSetupPage } from "./PlanningSetupPage";

let canUpdate = true;

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdate, isLoading: false }),
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="planning-setup-location">{`${location.pathname}${location.search}`}</div>;
}

function renderSetup(lineId: string) {
  return render(
    <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${lineId}/setup/planning`]}>
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org-1",
          factoryId: REFUND_FACTORY.id ?? PRIMARY_FACTORY_ID,
          factoryKey: REFUND_FACTORY.key ?? PRIMARY_FACTORY_KEY,
          factory: REFUND_FACTORY,
          factories: [REFUND_FACTORY],
          openCreateWorkOrder: () => {},
        }}
      >
        <Routes>
          <Route path="/org-1/workspaces/:factoryKey/lines/:lineId/setup/planning" element={<PlanningSetupPage />} />
        </Routes>
        <LocationProbe />
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
}

describe("PlanningSetupPage", () => {
  const boardHref = factoryHomePath("org-1", PRIMARY_FACTORY_KEY, firstFactoryLineId(REFUND_FACTORY));

  beforeEach(() => {
    canUpdate = true;
  });

  it("opens Planning settings for a line the user can update", () => {
    renderSetup(REFUND_LINE_PLAN_ID);

    expect(screen.getByTestId("planning-setup-location")).toHaveTextContent(
      factoryPlanningPath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
  });

  it("returns to the board when the user cannot update the factory", () => {
    canUpdate = false;
    renderSetup(REFUND_LINE_PLAN_ID);

    expect(screen.getByTestId("planning-setup-location")).toHaveTextContent(boardHref);
    expect(screen.getByTestId("planning-setup-location")).not.toHaveTextContent("planning=1");
  });

  it("returns to the board when the line is missing", () => {
    renderSetup("missing-line");

    expect(screen.getByTestId("planning-setup-location")).toHaveTextContent(boardHref);
    expect(screen.getByTestId("planning-setup-location")).not.toHaveTextContent("planning=1");
  });
});
