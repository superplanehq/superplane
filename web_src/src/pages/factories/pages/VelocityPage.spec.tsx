import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesDescribeFactoryVelocityResponse, FactoriesFactory, FactoriesWorkOrder } from "@/api-client";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  REFUND_FACTORY_WITH_REPO,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { VelocityPage } from "./VelocityPage";

interface VelocityHookState {
  data?: FactoriesDescribeFactoryVelocityResponse;
  isLoading?: boolean;
  isFetching?: boolean;
  error?: Error | null;
}

interface WorkOrdersHookState {
  data?: FactoriesWorkOrder[];
  isLoading?: boolean;
  isFetching?: boolean;
  error?: Error | null;
}

const velocityHookState: VelocityHookState = {};
const workOrdersHookState: WorkOrdersHookState = {};
const lastVelocityRequest: { integrationId?: string; repository?: string } = {};

vi.mock("@/hooks/useFactoryVelocity", () => ({
  useFactoryVelocity: (
    _organizationId: string,
    _factoryId: string,
    options: { integrationId?: string; repository?: string },
  ) => {
    lastVelocityRequest.integrationId = options.integrationId;
    lastVelocityRequest.repository = options.repository;
    return {
      data: velocityHookState.data,
      isLoading: velocityHookState.isLoading ?? false,
      isFetching: velocityHookState.isFetching ?? false,
      error: velocityHookState.error ?? null,
      refetch: vi.fn(),
    };
  },
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({
    data: workOrdersHookState.data ?? [],
    isLoading: workOrdersHookState.isLoading ?? false,
    isFetching: workOrdersHookState.isFetching ?? false,
    error: workOrdersHookState.error ?? null,
  }),
}));

function renderShell(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
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
    </QueryClientProvider>,
  );
}

function resetState() {
  velocityHookState.data = undefined;
  velocityHookState.isLoading = false;
  velocityHookState.isFetching = false;
  velocityHookState.error = null;
  workOrdersHookState.data = [];
  workOrdersHookState.isLoading = false;
  workOrdersHookState.isFetching = false;
  workOrdersHookState.error = null;
  lastVelocityRequest.integrationId = undefined;
  lastVelocityRequest.repository = undefined;
}

describe("VelocityPage shell", () => {
  it("sets the document title from the page and workspace name", () => {
    resetState();
    renderShell();

    expect(document.title).toBe(`Velocity · ${REFUND_FACTORY.name} · SuperPlane`);
  });

  it("shows the loading state while velocity is loading", () => {
    resetState();
    velocityHookState.isLoading = true;

    renderShell();

    expect(screen.getByTestId("velocity-loading-state")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-yesterday")).not.toBeInTheDocument();
  });

  it("shows an error state with retry when velocity fails to load", () => {
    resetState();
    velocityHookState.error = new Error("network");

    renderShell();

    expect(screen.getByTestId("velocity-error-state")).toBeInTheDocument();
    expect(screen.getByTestId("velocity-error-retry")).toBeInTheDocument();
    expect(screen.queryByTestId("velocity-yesterday")).not.toBeInTheDocument();
  });

  it("keeps the loaded view when a refetch fails and cached data remains", () => {
    resetState();
    velocityHookState.data = {
      yesterday: { superplaneMerged: 3, waste: 1 },
      totals: {
        superplaneMerged: 12,
        peopleMerged: 0,
        waste: 4,
        superplaneSharePct: 0,
        wastePct: 25,
      },
      points: [{ day: "Mon", superplaneMerged: 2, peopleMerged: 0, waste: 1 }],
      hasPeopleCohort: false,
    };
    velocityHookState.error = new Error("network");

    renderShell();

    expect(screen.getByTestId("velocity-yesterday")).toHaveTextContent("3");
    expect(screen.queryByTestId("velocity-error-state")).not.toBeInTheDocument();
  });

  it("renders the loaded view and hides People cohort without a repo", () => {
    resetState();
    velocityHookState.data = {
      yesterday: { superplaneMerged: 3, waste: 1 },
      totals: {
        superplaneMerged: 12,
        peopleMerged: 0,
        waste: 4,
        superplaneSharePct: 0,
        wastePct: 25,
      },
      points: [
        { day: "Mon", superplaneMerged: 2, peopleMerged: 0, waste: 1 },
        { day: "Tue", superplaneMerged: 3, peopleMerged: 0, waste: 0 },
      ],
      hasPeopleCohort: false,
    };

    renderShell(REFUND_FACTORY);

    const yesterday = screen.getByTestId("velocity-yesterday");
    expect(yesterday).toHaveTextContent("Merged PRs");
    expect(yesterday).toHaveTextContent("3");
    expect(yesterday).not.toHaveTextContent("Cost");

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("Connect a GitHub repository in workspace setup to compare People and SuperPlane.");
    expect(split).not.toHaveTextContent("SuperPlane authored");

    // No onboarding repo configured, so the velocity request omits integration/repo entirely.
    expect(lastVelocityRequest.integrationId).toBeUndefined();
    expect(lastVelocityRequest.repository).toBeUndefined();
  });

  it("does not render any repo selector — the workspace's onboarding repo is used automatically", () => {
    resetState();
    velocityHookState.data = {
      yesterday: { superplaneMerged: 3, waste: 1 },
      totals: {
        superplaneMerged: 12,
        peopleMerged: 0,
        waste: 4,
        superplaneSharePct: 0,
        wastePct: 25,
      },
      points: [{ day: "Mon", superplaneMerged: 2, peopleMerged: 0, waste: 1 }],
      hasPeopleCohort: true,
      repository: REFUND_FACTORY_WITH_REPO.onboarding?.appRepository,
    };

    renderShell(REFUND_FACTORY_WITH_REPO);

    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(lastVelocityRequest.integrationId).toBe(REFUND_FACTORY_WITH_REPO.onboarding?.vcsIntegrationId);
    expect(lastVelocityRequest.repository).toBe(REFUND_FACTORY_WITH_REPO.onboarding?.appRepository);

    const split = screen.getByTestId("velocity-source-split");
    expect(split).not.toHaveTextContent("Connect a GitHub repository");
  });

  it("explains when People merges could not be loaded", () => {
    resetState();
    velocityHookState.data = {
      yesterday: { superplaneMerged: 3, waste: 0 },
      totals: {
        superplaneMerged: 12,
        peopleMerged: 0,
        waste: 0,
        superplaneSharePct: 0,
        wastePct: 0,
      },
      points: [{ day: "Mon", superplaneMerged: 3, peopleMerged: 0, waste: 0 }],
      hasPeopleCohort: false,
      peopleSearchFailed: true,
      repository: "acme/api",
    };

    renderShell(REFUND_FACTORY_WITH_REPO);

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("We could not load People merges. SuperPlane counts still show.");
    expect(split).not.toHaveTextContent("No merged pull requests");
    expect(split).not.toHaveTextContent("SuperPlane authored");
  });

  it("explains when work order time could not be loaded", () => {
    resetState();
    velocityHookState.data = {
      yesterday: { superplaneMerged: 3, waste: 1 },
      totals: {
        superplaneMerged: 12,
        peopleMerged: 0,
        waste: 4,
        superplaneSharePct: 0,
        wastePct: 25,
      },
      points: [{ day: "Mon", superplaneMerged: 2, peopleMerged: 0, waste: 1 }],
      hasPeopleCohort: false,
    };
    workOrdersHookState.error = new Error("network");

    renderShell();

    const flow = screen.getByTestId("velocity-work-order-flow");
    expect(flow).toHaveTextContent("We could not load work order time.");
    expect(flow).not.toHaveTextContent("No work orders closed in this period.");
  });
});
