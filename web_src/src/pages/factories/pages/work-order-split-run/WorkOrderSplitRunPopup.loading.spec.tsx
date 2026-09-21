import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as FactoryData from "@/hooks/useFactoryData";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { APPROVAL_WORK_ORDER, DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const lookupState = vi.hoisted(() => ({
  factoryPending: false,
  planning: { enabled: true, clarity: true, confidence: true },
  sessionLoading: false,
  artifactsLoading: false,
  artifactsError: null as Error | null,
}));

vi.mock("@/hooks/useFactoryData", () => {
  const actual = unmockedSrc<typeof FactoryData>("hooks/useFactoryData");
  return {
    ...actual,
    useFactory: () => ({
      data: lookupState.factoryPending ? undefined : { id: "factory-1", planning: lookupState.planning },
      isPending: lookupState.factoryPending,
    }),
  };
});

vi.mock("./useAnalysisPlanningSession", () => ({
  useAnalysisPlanningSession: () => ({
    session: null,
    isLoading: lookupState.sessionLoading,
    queryError: null,
    view: { machineStatus: "waiting", messages: [], executionId: "", canvasId: "" },
    canSend: false,
    onSubmitSurvey: () => undefined,
  }),
}));

vi.mock("./useSplitRunPopupData", () => ({
  useSplitRunPopupData: () => ({
    artifacts: [],
    pullRequests: [],
    sourceDescription: "",
    useLive: true,
    artifactsLoading: lookupState.artifactsLoading,
    artifactsError: lookupState.artifactsError,
    pullRequestsLoading: false,
    pullRequestsError: null,
  }),
}));

vi.mock("./ClassicWorkOrderPopup", () => ({
  ClassicWorkOrderPopup: () => <div data-testid="classic-work-order-popup" />,
}));

function renderPopup(onClose?: () => void, fixture = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER)) {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId="organization-1"
              factoryId="factory-1"
              orderId="work-order-1"
              fixture={fixture}
              onClose={onClose}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("WorkOrderSplitRunPopup loading mode", () => {
  beforeEach(() => {
    lookupState.factoryPending = false;
    lookupState.planning = { enabled: true, clarity: true, confidence: true };
    lookupState.sessionLoading = false;
    lookupState.artifactsLoading = false;
    lookupState.artifactsError = null;
  });

  it("shows a dismissible loading popup while Planning settings load", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    lookupState.factoryPending = true;

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
    expect(screen.queryByTestId("classic-work-order-popup")).not.toBeInTheDocument();
  });

  it("shows a loading popup instead of the classic popup while draft artifacts load", () => {
    lookupState.artifactsLoading = true;

    renderPopup();

    expect(screen.getByTestId("work-order-split-run-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("classic-work-order-popup")).not.toBeInTheDocument();
  });

  it("keeps a started task on the analysis popup while artifacts load", () => {
    lookupState.artifactsLoading = true;
    const fixture = splitRunFixtureForWorkOrder(APPROVAL_WORK_ORDER);
    expect(fixture.footer.kind).not.toBe("draft");

    renderPopup(undefined, fixture);

    expect(screen.queryByTestId("classic-work-order-popup")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run-loading")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
  });

  it("keeps a draft on the analysis popup when artifact lookup fails", () => {
    lookupState.artifactsError = new Error("artifacts unavailable");

    renderPopup();

    expect(screen.queryByTestId("classic-work-order-popup")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run-loading")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
  });
});
