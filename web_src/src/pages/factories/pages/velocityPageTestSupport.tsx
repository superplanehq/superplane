import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { vi } from "bun:test";

import type { FactoriesDescribeFactoryVelocityResponse, FactoriesFactory, FactoriesWorkOrder } from "@/api-client";
import type { FactoryVelocityParams } from "@/hooks/useFactoryVelocity";
import { TooltipProvider } from "@/ui/tooltip";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
import { PEOPLE_FIRST_PAGE_SIZE, peoplePageSizeForOffset } from "../lib/velocityPeopleSort";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { VelocityPage } from "./VelocityPage";

type VelocityPerson = NonNullable<FactoriesDescribeFactoryVelocityResponse["people"]>[number];

export interface VelocityHookState {
  data?: FactoriesDescribeFactoryVelocityResponse;
  isLoading?: boolean;
  isFetching?: boolean;
  error?: Error | null;
  /**
   * The whole People cohort behind the mock's paging, sorted the way the
   * backend would return it. Defaults to `data.people` when unset, which is
   * enough for tests with fewer than one page of people.
   */
  allPeople?: VelocityPerson[];
  /**
   * Holds the report of the previous offset, the way the query does while the
   * next page loads.
   */
  holdsPreviousReport?: boolean;
}

export interface WorkOrdersHookState {
  data?: FactoriesWorkOrder[];
  isLoading?: boolean;
  isFetching?: boolean;
  error?: Error | null;
}

export const velocityHookState: VelocityHookState = {};
export const workOrdersHookState: WorkOrdersHookState = {};
/** Every call the page made to `useFactoryVelocity`, newest last. */
export const velocityHookCalls: FactoryVelocityParams[] = [];

/** Workspace setup picks the GitHub integration and the app repository. */
export const FACTORY_WITH_SETUP_REPO: FactoriesFactory = {
  ...REFUND_FACTORY,
  onboarding: { ...REFUND_FACTORY.onboarding, vcsIntegrationId: "int-1", appRepository: "acme/api" },
};

export const startSync = vi.fn();
export const syncHookState: { isPending?: boolean } = {};

/** A cohort large enough to exercise paging, already in default sort order. */
export function manyPeople(count: number): VelocityPerson[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `person-${index + 1}`,
    name: `Contributor ${String(index + 1).padStart(2, "0")}`,
    email: `contributor${index + 1}@example.com`,
    authoredMerged: 1,
    factoryMerged: 0,
    factoryWaste: 0,
    medianCycleHours: 0,
    costCents: "0",
  }));
}

export function factoryVelocityHookResult(params: FactoryVelocityParams) {
  velocityHookCalls.push(params);

  const base = velocityHookState.data;
  const allPeople = velocityHookState.allPeople ?? base?.people ?? [];
  const holdsPreviousReport = velocityHookState.holdsPreviousReport ?? false;
  const offset = holdsPreviousReport ? 0 : (params.peopleOffset ?? 0);
  const pageSize = holdsPreviousReport
    ? PEOPLE_FIRST_PAGE_SIZE
    : (params.peoplePageSize ?? peoplePageSizeForOffset(offset));
  const page = allPeople.slice(offset, offset + pageSize);

  return {
    data: base
      ? {
          ...base,
          people: page,
          peopleTotal: allPeople.length,
          peopleHasMore: offset + page.length < allPeople.length,
        }
      : undefined,
    isLoading: velocityHookState.isLoading ?? false,
    isFetching: velocityHookState.isFetching ?? false,
    isPlaceholderData: holdsPreviousReport,
    error: velocityHookState.error ?? null,
    refetch: vi.fn(),
  };
}

export function syncFactoryVelocityHookResult() {
  return {
    mutate: startSync,
    isPending: syncHookState.isPending ?? false,
  };
}

export function factoryWorkOrdersHookResult() {
  return {
    data: workOrdersHookState.data ?? [],
    isLoading: workOrdersHookState.isLoading ?? false,
    isFetching: workOrdersHookState.isFetching ?? false,
    error: workOrdersHookState.error ?? null,
  };
}

/** A window with output, so the page renders the report instead of a state card. */
export function populatedResponse(
  overrides: Partial<FactoriesDescribeFactoryVelocityResponse> = {},
): FactoriesDescribeFactoryVelocityResponse {
  return {
    yesterday: { superplaneMerged: 3, waste: 1 },
    totals: {
      superplaneMerged: 12,
      peopleMerged: 8,
      waste: 4,
      superplaneSharePct: 60,
      wastePct: 25,
      costCents: "4200",
      tokens: "185000",
      wasteCostCents: "900",
      tasksClosed: 16,
      tasksWaste: 4,
    },
    points: [
      { day: "1", superplaneMerged: 2, peopleMerged: 1, waste: 1, costCents: "800", tokens: "20000" },
      { day: "2", superplaneMerged: 3, peopleMerged: 2, waste: 0, costCents: "1200", tokens: "30000" },
    ],
    hasPeopleCohort: true,
    ...overrides,
  };
}

export function renderShell(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <TooltipProvider delayDuration={0}>
        <MemoryRouter initialEntries={["/velocity"]}>
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
            <VelocityPage />
          </FactoriesLayoutContext.Provider>
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

export function metricCell(container: HTMLElement, label: string): HTMLElement {
  const cell = within(container).getByText(label).closest("div.min-w-0");
  if (!(cell instanceof HTMLElement)) {
    throw new Error(`No metric cell for ${label}`);
  }
  return cell;
}

export function metricColorDot(cell: HTMLElement): HTMLElement | null {
  return cell.querySelector("span[aria-hidden]");
}

export function resetState() {
  startSync.mockClear();
  syncHookState.isPending = false;
  velocityHookState.data = undefined;
  velocityHookState.isLoading = false;
  velocityHookState.isFetching = false;
  velocityHookState.error = null;
  velocityHookState.allPeople = undefined;
  velocityHookState.holdsPreviousReport = false;
  velocityHookCalls.length = 0;
  workOrdersHookState.data = [];
  workOrdersHookState.isLoading = false;
  workOrdersHookState.isFetching = false;
  workOrdersHookState.error = null;
}
