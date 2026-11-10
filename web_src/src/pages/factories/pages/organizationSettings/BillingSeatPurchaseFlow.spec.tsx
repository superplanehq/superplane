import { describe, expect, it, vi } from "bun:test";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Fragment, StrictMode, Suspense } from "react";
import { Link, MemoryRouter, useLocation } from "react-router";

const billing = {
  billingEnabled: true,
  planPerSeat: true,
  planSource: "polar",
  polarSubscriptionStatus: "active",
  creditPurchaseAllowed: true,
  cancelAtPeriodEnd: true,
};

vi.mock("@/hooks/useOrganizationBilling", () => ({
  useOrganizationBilling: () => ({ data: billing, refetch: vi.fn() }),
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  syncOrganizationBilling: vi.fn(),
}));
vi.mock("@/hooks/useBillingCardSetup", () => ({
  useBillingCardSetup: () => ({ isPending: false, error: "", open: vi.fn() }),
}));

const { BillingSeatPurchaseFlow } = await import("./BillingSeatPurchaseFlow");

describe("billing purchase deep-link consumption", () => {
  it.each([false, true])(
    "keeps a closed dialog closed during pending navigation and accepts a fresh link (Strict Mode: %s)",
    async (strictMode) => {
      const Mode = strictMode ? StrictMode : Fragment;
      let resumeNavigation!: () => void;
      let navigationBlocked = true;
      const pendingNavigation = new Promise<void>((resolve) => {
        resumeNavigation = resolve;
      });

      function NavigationDestination() {
        const location = useLocation();
        // Keep URL consumption pending while the user closes the already open dialog.
        if (!location.search.includes("buySeats=1") && navigationBlocked) throw pendingNavigation;
        return <Link to="/billing?buySeats=1">Buy more seats</Link>;
      }

      render(
        <Mode>
          <QueryClientProvider client={new QueryClient()}>
            <MemoryRouter initialEntries={["/billing?buySeats=1"]}>
              <Suspense fallback={<p>Loading</p>}>
                <BillingSeatPurchaseFlow organizationId="org-1" canManageBilling cancelAtPeriodEnd loading={false}>
                  {(_card, dialog) => (
                    <>
                      {dialog}
                      <NavigationDestination />
                    </>
                  )}
                </BillingSeatPurchaseFlow>
              </Suspense>
            </MemoryRouter>
          </QueryClientProvider>
        </Mode>,
      );
      await screen.findByRole("dialog", { name: "Buy seats" });
      fireEvent.click(screen.getByRole("button", { name: "Close" }));
      try {
        await waitFor(() => expect(screen.queryAllByRole("dialog", { hidden: true })).toHaveLength(0));
      } finally {
        navigationBlocked = false;
        await act(async () => resumeNavigation());
      }
      expect(screen.queryAllByRole("dialog", { hidden: true })).toHaveLength(0);

      fireEvent.click(screen.getByRole("link", { name: "Buy more seats" }));
      expect(await screen.findByRole("dialog", { name: "Buy seats" })).toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: "Close" }));
      await waitFor(() => expect(screen.queryAllByRole("dialog", { hidden: true })).toHaveLength(0));
    },
  );
});
