import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesFactoryIntake, FactoriesWorkOrder, FactoryApp } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { editFactoryLinePath, factoryAppConfigurePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  ACME_ONBOARDING_FACTORY,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_APP,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_DONE_REJECTED_ORDER, BOARD_IMPLEMENT_FAILED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { withPlanLinePhases } from "../__fixtures__/lineMetricsPlanLine";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LINE_LIST_METRICS_BY_ID } from "./lineListMetricsMockData";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "./onboarding/first-run/reviewCandidates";
import { LinesPage } from "./LinesPage";

const createFactoryLineMutateAsync = vi.fn();
const updateFactoryLineMutateAsync = vi.fn();
const useFactoryWorkOrders = vi.fn(() => ({ data: [] as FactoriesWorkOrder[] }));
const useFactoryApps = vi.fn(() => ({ data: [] as FactoryApp[] }));
const useFactoryIntakes = vi.fn(() => ({ data: [] as FactoriesFactoryIntake[] }));
const createFactoryIntakeMutateAsync = vi.fn();

const SENTRY_INTAKE_ID = "intake-sentry";
const PAGERDUTY_INTAKE_ID = "intake-pagerduty";

const CONFIGURED_INTAKES: FactoriesFactoryIntake[] = [
  GITHUB_ISSUES_INTAKE,
  {
    id: SENTRY_INTAKE_ID,
    canvasId: "app-sentry-intake",
    name: "Sentry exceptions",
    source: "SOURCE_SENTRY_EXCEPTIONS",
    healthy: true,
  },
  {
    id: PAGERDUTY_INTAKE_ID,
    canvasId: "app-pagerduty-intake",
    name: "PagerDuty incidents",
    source: "SOURCE_PAGERDUTY_INCIDENTS",
    healthy: true,
  },
];

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => useFactoryWorkOrders(),
  useFactoryApps: () => useFactoryApps(),
  useCreateFactoryLine: () => ({ mutateAsync: createFactoryLineMutateAsync, isPending: false }),
  useUpdateFactoryLine: () => ({ mutateAsync: updateFactoryLineMutateAsync, isPending: false }),
  useWorkOrderEvents: () => ({ data: { pages: [] } }),
  useWorkOrderArtifacts: () => ({ data: [] }),
  useCloseWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderAssignees: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateWorkOrderStatus: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => useFactoryIntakes(),
  useFactoryIntakeRuns: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useCreateFactoryIntake: () => ({ mutateAsync: createFactoryIntakeMutateAsync, isPending: false }),
  useUpdateFactoryIntake: () => ({ mutateAsync: vi.fn(), isPending: false, error: null }),
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

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "storybook-user" } }),
}));

vi.mock("@/hooks/useWorkOrderChecks", async () => {
  const { DEFAULT_CHECKS_BY_ORDER_ID } = await import("../__fixtures__/workOrderCheckFixtures");
  return {
    useWorkOrderChecks: (_organizationId: string, _factoryId: string, orderId: string) => ({
      data: DEFAULT_CHECKS_BY_ORDER_ID[orderId] ?? [],
    }),
  };
});

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
    window.localStorage.clear();
    updateFactoryLineMutateAsync.mockReset();
    useFactoryWorkOrders.mockReturnValue({ data: [] });
    useFactoryApps.mockReturnValue({ data: [] });
    useFactoryIntakes.mockReturnValue({ data: [] });
    createFactoryIntakeMutateAsync.mockReset();
  });

  function renderBoard(
    path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
    openCreateWorkOrder = vi.fn(),
    factory: FactoriesFactory = REFUND_FACTORY,
  ) {
    return render(
      <QueryClientProvider client={new QueryClient()}>
        <ThemeProvider>
          <TooltipProvider>
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
          </TooltipProvider>
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

  it("shows a score on a review-candidate backlog card and opens the split run", async () => {
    useFactoryWorkOrders.mockReturnValue({ data: REVIEW_CANDIDATE_WORK_ORDERS });
    const user = userEvent.setup();
    renderBoard();

    const card = screen.getByTestId("work-order-card-wo-review-pay-842");
    const cardScore = within(card).getByTestId("work-order-card-score-wo-review-pay-842");
    expect(cardScore).toHaveAttribute("role", "meter");
    expect(cardScore).toHaveAttribute("aria-valuenow", "5");
    expect(cardScore).toHaveAttribute("aria-valuemax", "5");
    expect(cardScore.querySelectorAll("[data-filled='true']")).toHaveLength(5);
    const start = within(card).getByRole("button", { name: "Start" });
    expect(cardScore.compareDocumentPosition(start) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Open Add retry handling to webhook delivery" }));

    const dialog = screen.getByTestId("work-order-split-run");
    expect(within(dialog).getByRole("heading", { name: "Add retry handling to webhook delivery" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Plan" })).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("tab", { name: "Ticket" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: "Description" })).toHaveAttribute("data-state", "active");
    expect(within(dialog).getByTestId("split-run-work-order-tab")).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-overview-checks")).toHaveTextContent("Confidence score");
    expect(within(dialog).getByTestId("split-run-review")).toBeInTheDocument();

    await user.click(within(dialog).getByRole("tab", { name: "Log" }));
    expect(within(dialog).getByRole("heading", { name: "Log" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("split-run-phase-backlog")).toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-ingest")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-analyze")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("split-run-phase-plan")).not.toBeInTheDocument();
    expect(screen.queryByTestId("review-candidate-modal")).not.toBeInTheDocument();
  });

  it("opens the Intake drawer beside the board when the intake query is set", () => {
    useFactoryIntakes.mockReturnValue({ data: CONFIGURED_INTAKES });
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    expect(screen.getByTestId("line-intake-drawer")).toBeInTheDocument();
    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Plan and Implement" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-close")).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-add")).toHaveTextContent("Add intake");
    expect(screen.getByTestId(`line-intake-source-${SENTRY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId(`line-intake-source-${PAGERDUTY_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-analyzing")).not.toBeInTheDocument();
  });

  it("lists two intakes on the same source", () => {
    useFactoryIntakes.mockReturnValue({
      data: [
        GITHUB_ISSUES_INTAKE,
        { id: "intake-triage", canvasId: "app-triage", name: "Triage issues", source: "SOURCE_GITHUB_ISSUES" },
      ],
    });
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toHaveTextContent("GitHub issues");
    expect(screen.getByTestId("line-intake-source-intake-triage")).toHaveTextContent("Triage issues");
  });

  it("creates an intake from the picker and opens its canvas", async () => {
    createFactoryIntakeMutateAsync.mockResolvedValueOnce({ id: "intake-new", canvasId: "canvas-new" });
    const user = userEvent.setup();
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    await user.click(screen.getByTestId("line-intake-add"));
    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    await waitFor(() => {
      expect(createFactoryIntakeMutateAsync).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
    });
    await waitFor(() => {
      expect(screen.getByTestId("lines-test-location")).toHaveTextContent(
        `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/apps/canvas-new`,
      );
    });
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
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderBoard(
      `/org-1/workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
      vi.fn(),
      ACME_ONBOARDING_FACTORY,
    );

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Collapse GitHub issues" })).toHaveAttribute("aria-expanded", "true");
    expect(screen.queryByTestId(`line-intake-source-${SENTRY_INTAKE_ID}`)).not.toBeInTheDocument();
    expect(screen.queryByTestId(`line-intake-source-${PAGERDUTY_INTAKE_ID}`)).not.toBeInTheDocument();
  });

  it("opens the factory canvas editor from Edit automation", async () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    const user = userEvent.setup();
    renderBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
    );

    await user.click(screen.getByRole("button", { name: `Open ${GITHUB_ISSUES_INTAKE.name} settings` }));
    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(screen.getByRole("link", { name: "Edit automation" })).toHaveAttribute(
      "href",
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, GITHUB_ISSUES_INTAKE_APP.id!, {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("loads analyzing tickets from the configured GitHub intake", () => {
    useFactoryIntakes.mockReturnValue({ data: [GITHUB_ISSUES_INTAKE] });
    renderBoard(
      `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`,
    );

    expect(screen.getByTestId(`line-intake-source-${GITHUB_ISSUES_INTAKE_ID}`)).toBeInTheDocument();
    expect(screen.queryByText("Handle duplicate refunds on retry")).not.toBeInTheDocument();
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

    await user.click(screen.getByTestId("lines-phase-menu-2"));
    expect(screen.queryByTestId("lines-phase-menu-2-edit")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-test-location")).not.toHaveTextContent(
      factoryAppConfigurePath("org-1", PRIMARY_FACTORY_KEY, "app-refund-done", {
        from: "lines",
        lineId: REFUND_LINE_PLAN_ID,
      }),
    );
  });

  it("hides the phase path and shows work-order filters", async () => {
    const user = userEvent.setup();
    renderBoard();

    const header = screen.getByTestId("lines-detail-header");
    expect(within(header).queryByText(/→/)).not.toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-scope-all")).toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-scope-active")).toHaveTextContent("Needs attention");
    expect(within(header).getByTestId("work-orders-scope-my")).toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-filter-trigger")).toBeInTheDocument();
    expect(within(header).getByTestId("work-orders-search-trigger")).toBeInTheDocument();
    expect(within(header).queryByTestId("work-order-list-create-button")).not.toBeInTheDocument();

    await user.click(within(header).getByTestId("work-orders-filter-trigger"));
    expect(screen.getByTestId("work-orders-filter-statuses")).toBeInTheDocument();
    expect(screen.queryByTestId("work-orders-filter-lineIds")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-orders-filter-assigneeIds")).toBeInTheDocument();
  });

  it("narrows the board when the search query changes", async () => {
    const user = userEvent.setup();
    useFactoryWorkOrders.mockReturnValue({
      data: [BOARD_IMPLEMENT_FAILED_ORDER, BOARD_DONE_REJECTED_ORDER],
    });
    renderBoard();

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.getByText("Replace the refund batch exporter")).toBeInTheDocument();

    await user.click(screen.getByTestId("work-orders-search-trigger"));
    await user.type(screen.getByTestId("work-orders-search-input"), "timeout");

    expect(screen.getByText("Fix refund dispatcher timeout loop")).toBeInTheDocument();
    expect(screen.queryByText("Replace the refund batch exporter")).not.toBeInTheDocument();
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
