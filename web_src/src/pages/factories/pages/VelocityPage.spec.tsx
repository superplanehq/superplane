import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type {
  FactoriesDescribeFactoryVelocityResponse,
  FactoriesWorkOrder,
  OrganizationsIntegration,
} from "@/api-client";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, REFUND_FACTORY } from "../__fixtures__/factoryPageResponses";
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
const integrationsHookState: { data: OrganizationsIntegration[] } = { data: [] };
const repositoryResourcesHookState: { data: Array<{ name?: string }>; isLoading: boolean } = {
  data: [],
  isLoading: false,
};

vi.mock("@/hooks/useFactoryVelocity", () => ({
  useFactoryVelocity: () => ({
    data: velocityHookState.data,
    isLoading: velocityHookState.isLoading ?? false,
    isFetching: velocityHookState.isFetching ?? false,
    error: velocityHookState.error ?? null,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({
    data: workOrdersHookState.data ?? [],
    isLoading: workOrdersHookState.isLoading ?? false,
    isFetching: workOrdersHookState.isFetching ?? false,
    error: workOrdersHookState.error ?? null,
  }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({ data: integrationsHookState.data }),
  useIntegrationResources: () => ({
    data: repositoryResourcesHookState.data,
    isLoading: repositoryResourcesHookState.isLoading,
  }),
}));

function renderShell() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/velocity"]}>
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
  integrationsHookState.data = [];
  repositoryResourcesHookState.data = [];
  repositoryResourcesHookState.isLoading = false;
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

    renderShell();

    const yesterday = screen.getByTestId("velocity-yesterday");
    expect(yesterday).toHaveTextContent("Merged PRs");
    expect(yesterday).toHaveTextContent("3");
    expect(yesterday).not.toHaveTextContent("Cost");

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("Connect GitHub to compare People and SuperPlane.");
    expect(split).not.toHaveTextContent("SuperPlane authored");
  });

  it("explains when People merges could not be loaded", () => {
    resetState();
    integrationsHookState.data = [
      {
        metadata: { id: "int-1", name: "GitHub", integrationName: "github" },
        status: { state: "ready" },
      },
    ];
    repositoryResourcesHookState.data = [{ name: "acme/api" }];
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

    renderShell();

    const split = screen.getByTestId("velocity-source-split");
    expect(split).toHaveTextContent("We could not load People merges. SuperPlane counts still show.");
    expect(split).not.toHaveTextContent("No merged pull requests");
    expect(split).not.toHaveTextContent("SuperPlane authored");
  });

  it("explains when task time could not be loaded", () => {
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
    expect(flow).toHaveTextContent("We could not load task time.");
    expect(flow).not.toHaveTextContent("No tasks closed in this period.");
  });
});
