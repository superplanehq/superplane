import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { editFactoryLinePath, factoryAppConfigurePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  GITHUB_ISSUES_INTAKE_APP,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { withPlanLinePhases } from "../__fixtures__/lineMetricsPlanLine";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LINE_LIST_METRICS_BY_ID } from "./lineListMetricsMockData";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "./onboarding/first-run/reviewCandidates";
import { LinesPage } from "./LinesPage";

const createFactoryLineMutateAsync = vi.fn();
const updateFactoryLineMutateAsync = vi.fn();
const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));
const useFactoryApps = vi.fn(() => ({ data: [] as Array<{ id?: string; name?: string }> }));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryApps: () => useFactoryApps(),
  useCreateFactoryLine: () => ({ mutateAsync: createFactoryLineMutateAsync, isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: updateFactoryLineMutateAsync, isPending: false }),
}));

vi.mock("@/hooks/useWorkOrderCardActions", () => ({
  useWorkOrderCardActions: () => ({
    isDispatching: false,
    isAssigneesSaving: false,
    onDispatch: vi.fn(),
    onAssigneesSave: vi.fn(),
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: () => undefined,
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="lines-test-location">{location.pathname}</div>;
}

function renderList(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
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
    </QueryClientProvider>,
  );
}

describe("LinesPage metrics", () => {
  it("shows zero success rate and completions when a line has no nested metrics", () => {
    renderList();
    const cards = screen.getAllByTestId("lines-card-metrics");
    expect(cards[0]).toHaveTextContent("0%");
    expect(cards[0]).toHaveTextContent("0 per day");
    expect(cards[0]).toHaveTextContent("—");
  });

  it("shows live numbers for a line that has nested metrics", () => {
    const factory: FactoriesFactory = {
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map((line) =>
        line.id === REFUND_LINE_PLAN_ID ? { ...line, metrics: LINE_LIST_METRICS_BY_ID[REFUND_LINE_PLAN_ID]! } : line,
      ),
    };
    renderList(factory);
    expect(screen.getByTestId(`lines-card-${REFUND_LINE_PLAN_ID}`)).toHaveTextContent("82%");
    expect(screen.getByTestId("lines-card-line-hotfix")).toHaveTextContent("0%");
    expect(screen.getByTestId("lines-card-line-hotfix")).toHaveTextContent("0 per day");
  });
});

describe("LinesPage card menu", () => {
  it("duplicates a line with its steps and a unique copy name, then opens the new line", async () => {
    const sourceLine = REFUND_FACTORY.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID);
    const newLine = { id: "line-new", name: "plan-and-implement copy", steps: sourceLine?.steps ?? [] };
    createFactoryLineMutateAsync.mockResolvedValueOnce(newLine);

    const user = userEvent.setup();
    renderList();

    const card = screen.getByTestId(`lines-card-${REFUND_LINE_PLAN_ID}`);
    await user.click(within(card).getByTestId("lines-card-menu"));
    await user.click(screen.getByTestId("lines-card-duplicate"));

    await waitFor(() => {
      expect(createFactoryLineMutateAsync).toHaveBeenCalledWith({
        name: "plan-and-implement copy",
        steps: sourceLine?.steps ?? [],
      });
    });
    await waitFor(() => {
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
        factoryLineDetailPath("org-1", PRIMARY_FACTORY_KEY, "line-new"),
      );
    });
  });
});

describe("LinesPage board", () => {
  beforeEach(() => {
    updateFactoryLineMutateAsync.mockReset();
    useFactoryWorkOrders.mockReturnValue({ data: [] });
    useFactoryApps.mockReturnValue({ data: [] });
  });

  function renderBoard(
    path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
    openCreateWorkOrder = vi.fn(),
    factory: FactoriesFactory = REFUND_FACTORY,
  ) {
    return render(
      <QueryClientProvider client={new QueryClient()}>
        <ThemeProvider>
          <MemoryRouter initialEntries={[path]}>
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
          </MemoryRouter>
        </ThemeProvider>
      </QueryClientProvider>,
    );
  }

  it("does not show a back link to the lines list", () => {
    renderBoard();

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-back-to-list")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-drawer")).not.toBeInTheDocument();
  });

  it("creates work orders from the backlog header plus", async () => {
    const openCreateWorkOrder = vi.fn();
    const user = userEvent.setup();
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`, openCreateWorkOrder);

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(within(backlog).queryByRole("button", { name: "Add work order" })).not.toBeInTheDocument();

    await user.click(within(backlog).getByTestId("lines-backlog-create"));
    expect(openCreateWorkOrder).toHaveBeenCalledTimes(1);
  });

  it("sets a pastel colour on the backlog from circular swatches", async () => {
    const user = userEvent.setup();
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`);

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-color-lime"));

    expect(screen.getByTestId("lines-backlog-column").className).toContain("bg-lime-300");
  });

  it("shows a score on a review-candidate backlog card and opens the plan review", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderBoard();

    const card = screen.getByTestId("work-order-card-wo-review-pay-842");
    expect(within(card).getByTestId("work-order-card-score-wo-review-pay-842")).toHaveTextContent("95%");
    expect(within(card).queryByRole("button", { name: "Start" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

    const dialog = screen.getByTestId("review-candidate-modal");
    expect(within(dialog).getByRole("heading", { name: "Review candidate" })).toBeInTheDocument();
    expect(within(dialog).getByText("PAY-842")).toBeInTheDocument();
    expect(within(dialog).getByTestId("review-candidate-section-04")).toHaveTextContent("Implementation plan");
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("opens the Intake drawer beside the board when the intake query is set", () => {
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    expect(screen.getByTestId("line-intake-drawer")).toBeInTheDocument();
    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Plan and Implement" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-close")).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-add")).toHaveTextContent("Add intake");
    expect(screen.getByTestId("line-intake-source-sentry-exceptions")).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-source-pagerduty-incidents")).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-analyzing")).not.toBeInTheDocument();
  });

  it("shows a backlog onboarding card on Acme when the backlog is empty", () => {
    renderBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}`,
      vi.fn(),
      ACME_ONBOARDING_FACTORY,
    );

    expect(screen.getByTestId("backlog-onboarding-card")).toBeInTheDocument();
    expect(screen.queryByText("No work orders in the backlog.")).not.toBeInTheDocument();
  });

  it("shows GitHub issues only on Acme onboarding intake", () => {
    renderBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&source=github-issues`,
      vi.fn(),
      ACME_ONBOARDING_FACTORY,
    );

    expect(screen.getByTestId("line-intake-source-github-issues")).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-sentry-exceptions")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-source-pagerduty-incidents")).not.toBeInTheDocument();
  });

  it("opens the factory canvas editor from Edit automation", async () => {
    useFactoryApps.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE_APP] });
    const user = userEvent.setup();
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&source=github-issues`);

    await user.click(screen.getByRole("button", { name: "Open GitHub issues settings" }));
    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(screen.getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, GITHUB_ISSUES_INTAKE_APP.id!, {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("nests analyzing tickets under GitHub issues when that source is open", () => {
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&source=github-issues`);

    expect(screen.getByTestId("line-intake-analyzing")).toBeInTheDocument();
    expect(screen.getByText("Handle duplicate refunds on retry")).toBeInTheDocument();
  });

  it("renames the board title on Enter", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("lines-board-title"));
    const input = await screen.findByTestId("lines-board-title-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Refund line");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith({
        lineId: REFUND_LINE_PLAN_ID,
        name: "Refund line",
      });
    });
  });

  it("renames a column title on Enter", async () => {
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("lines-column-title-backlog"));
    const input = await screen.findByTestId("lines-column-title-backlog-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Inbox");
    await user.keyboard("{Enter}");

    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Inbox");
  });

  it("opens backlog settings in a modal and does not open a canvas", async () => {
    const user = userEvent.setup();
    const linePath = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`;
    renderBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));

    expect(screen.getByTestId("lines-backlog-settings")).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toHaveValue("Backlog");
    expect(screen.getByLabelText("Size")).toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(linePath);
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent("configure=1");
  });

  it("saves the backlog name from the settings modal", async () => {
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("lines-backlog-menu"));
    await user.click(screen.getByTestId("lines-backlog-menu-edit"));
    const name = screen.getByLabelText("Name");
    await user.clear(name);
    await user.type(name, "Inbox");
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(screen.queryByTestId("lines-backlog-settings")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Inbox");
  });

  it("labels phase Edit as Edit Automation", async () => {
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-edit")).toHaveTextContent("Edit Automation");
    await user.keyboard("{Escape}");

    await user.click(screen.getByTestId("lines-backlog-menu"));
    expect(screen.getByTestId("lines-backlog-menu-edit")).toHaveTextContent("Edit");
    expect(screen.queryByTestId("lines-backlog-menu-parallelism")).not.toBeInTheDocument();
  });

  it("opens Set parallelism and saves a new cap", async () => {
    updateFactoryLineMutateAsync.mockResolvedValueOnce({});
    const user = userEvent.setup();
    renderBoard();

    await user.click(screen.getByTestId("lines-phase-menu-0"));
    expect(screen.getByTestId("lines-phase-menu-0-parallelism")).toHaveTextContent("Set parallelism (10)");
    await user.click(screen.getByTestId("lines-phase-menu-0-parallelism"));

    const input = screen.getByTestId("lines-parallelism-input");
    await user.clear(input);
    await user.type(input, "20");
    await user.click(screen.getByTestId("lines-parallelism-save"));

    await waitFor(() => {
      expect(updateFactoryLineMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          lineId: REFUND_LINE_PLAN_ID,
          steps: expect.arrayContaining([expect.objectContaining({ maxParallelism: 20 })]),
        }),
      );
    });
  });

  it("hides Edit on the Done column", async () => {
    const user = userEvent.setup();
    const factory: FactoriesFactory = {
      ...REFUND_FACTORY,
      lines: (REFUND_FACTORY.lines ?? []).map(withPlanLinePhases),
    };
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`, vi.fn(), factory);

    await user.click(screen.getByTestId("lines-phase-menu-3"));
    expect(screen.queryByTestId("lines-phase-menu-3-edit")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent(
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, "app-refund-done", {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("opens Edit from the line overflow menu", async () => {
    const user = userEvent.setup();
    renderBoard();

    expect(screen.queryByRole("link", { name: "Edit" })).not.toBeInTheDocument();
    await user.click(screen.getByTestId("lines-edit-menu"));
    await user.click(screen.getByTestId("lines-edit-menu-edit"));

    expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
      editFactoryLinePath("org-1", PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
  });
});
