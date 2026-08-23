import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import { editFactoryLinePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { BOARD_DONE_REJECTED_ORDER, BOARD_IMPLEMENT_FAILED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LINE_LIST_METRICS_BY_ID } from "./lineListMetricsMockData";
import { LinesPage } from "./LinesPage";

const createFactoryLineMutateAsync = vi.fn();
let mockWorkOrders: FactoriesWorkOrder[] = [];

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({ data: mockWorkOrders }),
  useFactoryApps: () => ({ data: [] }),
  useCreateFactoryLine: () => ({ mutateAsync: createFactoryLineMutateAsync, isPending: false }),
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
    mockWorkOrders = [];
    window.localStorage.clear();
  });

  function renderBoard() {
    return render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`]}>
          <FactoriesLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: PRIMARY_FACTORY_ID,
              factoryKey: PRIMARY_FACTORY_KEY,
              factory: REFUND_FACTORY,
              factories: [REFUND_FACTORY],
              openCreateWorkOrder: vi.fn(),
            }}
          >
            <Routes>
              <Route path="/org-1/workspaces/:factoryKey/lines/:lineId" element={<LinesPage />} />
              <Route path="/org-1/workspaces/:factoryKey/lines/:lineId/edit" element={<div>Edit line</div>} />
            </Routes>
            <LocationProbe />
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>,
    );
  }

  it("does not show a back link to the lines list", () => {
    renderBoard();

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-back-to-list")).not.toBeInTheDocument();
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
    mockWorkOrders = [BOARD_IMPLEMENT_FAILED_ORDER, BOARD_DONE_REJECTED_ORDER];
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
