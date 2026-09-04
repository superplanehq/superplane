import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EMPTY_SPENDING_FILTERS } from "@/pages/factories/pages/organizationSettings/spending-redesign/spendingRedesignLib";

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

import { useOrganizationSpendingReport, type OrganizationSpendingReportQuery } from "./useOrganizationSpendingReport";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function reportResponse(explorerTotals: { costCents: string }) {
  return { data: { explorerTotals } };
}

function baseQuery(overrides: Partial<OrganizationSpendingReportQuery> = {}): OrganizationSpendingReportQuery {
  return {
    organizationId: "org-1",
    range: { start: new Date("2026-01-01T00:00:00Z"), end: new Date("2026-02-01T00:00:00Z") },
    usageKind: "model",
    filters: EMPTY_SPENDING_FILTERS,
    groupBy: "workspace",
    ...overrides,
  };
}

describe("useOrganizationSpendingReport", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  /**
   * The filters and group-by are part of the query key, so switching either
   * one starts a new query. Without the previous report to hold, the page has
   * no data to render for a moment and drops every panel into its loading
   * state, which reads as the whole settings page reloading.
   */
  it("keeps the previous report visible while a new filter is loading", async () => {
    let resolveNextReport: ((page: unknown) => void) | undefined;
    organizationsDescribeOrganizationSpendingReport
      .mockResolvedValueOnce(reportResponse({ costCents: "100" }))
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveNextReport = resolve;
          }),
      );

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { result, rerender } = renderHook(
      ({ query }: { query: OrganizationSpendingReportQuery }) => useOrganizationSpendingReport(query),
      { initialProps: { query: baseQuery() }, wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.data?.explorerTotals?.costCents).toBe("100"));

    rerender({ query: baseQuery({ filters: { ...EMPTY_SPENDING_FILTERS, workspaceId: "workspace-1" } }) });

    await waitFor(() => expect(result.current.isFetching).toBe(true));
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data?.explorerTotals?.costCents).toBe("100");
    expect(result.current.isPlaceholderData).toBe(true);

    resolveNextReport?.(reportResponse({ costCents: "250" }));

    await waitFor(() => expect(result.current.data?.explorerTotals?.costCents).toBe("250"));
    expect(result.current.isPlaceholderData).toBe(false);
  });
});
