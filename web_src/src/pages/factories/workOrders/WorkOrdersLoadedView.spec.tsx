import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import { EMPTY_WORK_ORDER_FILTERS } from "../lib/workOrderListModel";
import type { WorkOrderListState } from "../lib/useWorkOrderListState";
import { WorkOrdersLoadedView } from "./WorkOrdersLoadedView";

const factory: FactoriesFactory = { id: "factory-1", name: "Refunds", key: "RF" };

const closedOrder: FactoriesWorkOrder = {
  id: "wo-closed",
  number: "1",
  title: "Archive stale draft",
  state: "STATE_CLOSED",
  result: "RESULT_COMPLETED",
  createdAt: "2024-06-01T00:00:00Z",
  updatedAt: "2024-06-02T00:00:00Z",
  lineDispatches: [],
  assignees: [],
};

const experimentalFeatureHas = { current: (_id: string) => false };

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => experimentalFeatureHas.current(id),
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [] }),
}));

function listState(overrides: Partial<WorkOrderListState> = {}): WorkOrderListState {
  const filters = overrides.filters ?? { ...EMPTY_WORK_ORDER_FILTERS, labels: ["mergeable"] };
  return {
    layout: "list",
    setLayout: vi.fn(),
    ordering: "updated",
    setOrdering: vi.fn(),
    scope: "active",
    setScope: vi.fn(),
    filters,
    toggleFilter: vi.fn(),
    removeFilter: vi.fn(),
    clearFilterDimension: vi.fn(),
    clearFilters: vi.fn(),
    filterCount: filters.labels.length + filters.statuses.length,
    search: "",
    setSearch: vi.fn(),
    clearSearch: vi.fn(),
    searchOpen: false,
    openSearch: vi.fn(),
    closeSearch: vi.fn(),
    filterMenuOpen: false,
    setFilterMenuOpen: vi.fn(),
    hasActiveFilters: true,
    resetView: vi.fn(),
    ...overrides,
  };
}

function renderView(state: WorkOrderListState) {
  render(
    <MemoryRouter>
      <WorkOrdersLoadedView
        organizationId="org-1"
        factoryKey="RF"
        factory={factory}
        factoryLines={[]}
        workOrders={[closedOrder]}
        state={state}
        canCreate
        onCreateWorkOrder={vi.fn()}
        canDispatch
        canAssign
        permissionsLoading={false}
        dispatchingOrderIds={new Set()}
        isAssigneesSaving={false}
        onDispatch={vi.fn()}
        onAssigneesSave={vi.fn()}
      />
    </MemoryRouter>,
  );
}

describe("WorkOrdersLoadedView", () => {
  it("uses the scoped empty state when a stored Mergeable filter is hidden", () => {
    experimentalFeatureHas.current = () => false;
    renderView(listState());

    expect(screen.getByTestId("work-orders-scope-empty-state")).toBeInTheDocument();
    expect(screen.queryByTestId("work-orders-filtered-empty-state")).not.toBeInTheDocument();
  });

  it("uses the filtered empty state when Mergeable is visible", () => {
    experimentalFeatureHas.current = () => true;
    renderView(listState());

    expect(screen.getByTestId("work-orders-filtered-empty-state")).toBeInTheDocument();
    expect(screen.queryByTestId("work-orders-scope-empty-state")).not.toBeInTheDocument();
  });
});
