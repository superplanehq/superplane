import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { organizationsDescribeOrganizationSpendingReport } = vi.hoisted(() => ({
  organizationsDescribeOrganizationSpendingReport: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    organizationsDescribeOrganizationSpendingReport,
  };
});

import { OrganizationSettingsWorkspaceUsagePage } from "./OrganizationSettingsWorkspaceUsagePage";

function reportResponse(costCents: string) {
  return {
    data: {
      kpiTotals: { costCents, totalTokens: "100", durationSeconds: "10" },
      explorerTotals: { costCents, totalTokens: "100", durationSeconds: "10" },
      series: [],
      seriesKeys: [],
      breakdown: [],
      credit: { remainingCreditCents: "0", grantTotalCents: "0" },
      catalogs: { workspaces: [], users: [], models: [], machines: [] },
    },
  };
}

function renderPage(queryClient: QueryClient) {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/org-1/settings/organization/spending"]}>
        <Routes>
          <Route
            path="/:organizationId/settings/organization/spending"
            element={<OrganizationSettingsWorkspaceUsagePage />}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function loadingMessage() {
  return screen.queryByText("Loading spending...");
}

function refetchIndicator() {
  return screen.queryByTestId("spending-refetch-indicator");
}

/**
 * The page fires one request per usage kind (model, compute). Queue up a
 * resolver for each call so the test can control exactly when each of them
 * settles instead of accidentally resolving the same promise twice.
 */
function mockPendingReports() {
  const resolvers: Array<(value: unknown) => void> = [];
  organizationsDescribeOrganizationSpendingReport.mockImplementation(
    () =>
      new Promise((resolve) => {
        resolvers.push(resolve);
      }),
  );
  return {
    resolveAll(costCents: string) {
      resolvers.splice(0).forEach((resolve) => resolve(reportResponse(costCents)));
    },
  };
}

describe("OrganizationSettingsWorkspaceUsagePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows the full-page loading state on the very first visit", async () => {
    const pending = mockPendingReports();

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderPage(queryClient);

    expect(loadingMessage()).toBeInTheDocument();

    await act(async () => {
      pending.resolveAll("100");
    });

    await waitFor(() => expect(loadingMessage()).not.toBeInTheDocument());
  });

  /**
   * Switching settings tabs remounts this page. Before this fix, the fresh
   * `new Date()` baked into the query range meant every remount produced a
   * new React Query key, so the cached report never hit and the page always
   * fell back to the full "Loading spending..." message. Quantizing the
   * range's "now" anchor keeps the key stable across quick remounts.
   */
  it("keeps showing the previous report on a return visit and shows a quiet indicator while it revalidates", async () => {
    organizationsDescribeOrganizationSpendingReport.mockResolvedValue(reportResponse("100"));

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { unmount } = renderPage(queryClient);

    await waitFor(() => expect(loadingMessage()).not.toBeInTheDocument());
    expect(screen.getByTestId("spending-kpi-spend")).toHaveTextContent("$1.00");

    unmount();

    // Simulate the cached report having gone stale while the tab was away
    // (staleTime is 30s on this query) so the return visit triggers a quiet
    // background refetch instead of resolving instantly from cache.
    act(() => {
      queryClient.invalidateQueries();
    });

    const pending = mockPendingReports();

    renderPage(queryClient);

    // The previously loaded report is visible immediately: no full-page
    // loading swap.
    expect(loadingMessage()).not.toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-spend")).toHaveTextContent("$1.00");
    await waitFor(() => expect(refetchIndicator()).toBeInTheDocument());

    await act(async () => {
      pending.resolveAll("250");
    });

    await waitFor(() => expect(refetchIndicator()).not.toBeInTheDocument());
    expect(loadingMessage()).not.toBeInTheDocument();
    expect(screen.getByTestId("spending-kpi-spend")).toHaveTextContent("$2.50");
  });
});
