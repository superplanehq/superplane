import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

function renderPopup(onClose?: () => void) {
  render(
    <WorkOrderSplitRunPopup
      organizationId="organization-1"
      factoryId="factory-1"
      orderId="work-order-1"
      fixture={splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)}
      onClose={onClose}
    />,
  );
}

describe("WorkOrderSplitRunPopup loading mode", () => {
  beforeEach(() => {
    lookupState.featureEnabled = true;
    lookupState.featureLoading = false;
    lookupState.sessionLoading = false;
  });

  it("shows a dismissible loading popup while Task Refinement access loads", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    lookupState.featureLoading = true;

    renderPopup(onClose);

    expect(screen.getByTestId("work-order-split-run-loading")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading task");
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows a loading popup instead of the classic popup while the refinement session loads", () => {
    lookupState.sessionLoading = true;

    renderPopup();

    expect(screen.getByTestId("work-order-split-run-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });
});
