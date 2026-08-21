import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";
import { editFactoryLinePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { LINE_LIST_METRICS_BY_ID } from "./lineListMetricsMockData";
import { LinesPage } from "./LinesPage";

const createFactoryLineMutateAsync = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({ data: [] }),
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
  function renderBoard(
    path = `/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`,
    openCreateWorkOrder = vi.fn(),
  ) {
    return render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter initialEntries={[path]}>
          <FactoriesLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: PRIMARY_FACTORY_ID,
              factoryKey: PRIMARY_FACTORY_KEY,
              factory: REFUND_FACTORY,
              factories: [REFUND_FACTORY],
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
      </QueryClientProvider>,
    );
  }

  it("does not show a back link to the lines list", () => {
    renderBoard();

    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.queryByTestId("lines-back-to-list")).not.toBeInTheDocument();
    expect(screen.queryByTestId("line-intake-drawer")).not.toBeInTheDocument();
  });

  it("creates work orders from the backlog header plus and add control", async () => {
    const openCreateWorkOrder = vi.fn();
    const user = userEvent.setup();
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}`, openCreateWorkOrder);

    const backlog = screen.getByTestId("lines-backlog-column");
    expect(within(backlog).getByTestId("lines-backlog-create")).toBeInTheDocument();
    expect(within(backlog).getByTestId("lines-backlog-add")).toHaveTextContent("Add work order");

    await user.click(within(backlog).getByTestId("lines-backlog-create"));
    await user.click(within(backlog).getByTestId("lines-backlog-add"));
    expect(openCreateWorkOrder).toHaveBeenCalledTimes(2);
  });

  it("opens the Intake drawer beside the board when the intake query is set", () => {
    renderBoard(`/org-1/workspaces/${PRIMARY_FACTORY_KEY}/lines/${REFUND_LINE_PLAN_ID}?intake=1`);

    expect(screen.getByTestId("line-intake-drawer")).toBeInTheDocument();
    expect(screen.getByTestId("lines-detail-page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Plan and Implement" })).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-close")).toBeInTheDocument();
    expect(screen.getByTestId("line-intake-add")).toHaveTextContent("Add intake");
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
