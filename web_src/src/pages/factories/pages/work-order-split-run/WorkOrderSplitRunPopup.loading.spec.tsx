import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const lookupState = vi.hoisted(() => ({
  featureEnabled: true,
  featureLoading: false,
  sessionLoading: false,
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => lookupState.featureEnabled,
    enabledExperimentalFeatures: [],
    isLoading: lookupState.featureLoading,
  }),
}));

vi.mock("./useAnalysisPlanningSession", () => ({
  useAnalysisPlanningSession: () => ({
    session: null,
    isLoading: lookupState.sessionLoading,
    queryError: null,
  }),
}));

vi.mock("./useSplitRunPopupData", () => ({
  useSplitRunPopupData: () => ({ artifacts: [] }),
}));

function renderPopup() {
  render(
    <WorkOrderSplitRunPopup
      organizationId="organization-1"
      factoryId="factory-1"
      orderId="work-order-1"
      fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
    />,
  );
}

describe("WorkOrderSplitRunPopup loading mode", () => {
  beforeEach(() => {
    lookupState.featureEnabled = true;
    lookupState.featureLoading = false;
    lookupState.sessionLoading = false;
  });

  it("does not show a popup while Task Refinement access loads", () => {
    lookupState.featureLoading = true;

    renderPopup();

    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("does not show the classic popup while the refinement session loads", () => {
    lookupState.sessionLoading = true;

    renderPopup();

    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });
});
