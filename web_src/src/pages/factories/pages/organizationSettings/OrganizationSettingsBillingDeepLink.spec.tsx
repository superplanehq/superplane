import { within } from "@testing-library/react";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { TEAM_ORGANIZATION_BILLING } from "../../__fixtures__/usageReportFixtures";

let billing: OrganizationsDescribeOrganizationBillingResponse = { ...TEAM_ORGANIZATION_BILLING };
let permissionsLoading = false;
let canManageBilling = true;
let featuresLoading = false;
let organizationReady = true;
const previewPurchase = vi.fn();
const purchaseSeats = vi.fn();
const refetch = vi.fn();
const syncBilling = vi.fn();

vi.mock("@/api-client", () => ({
  organizationsPreviewSeatPurchase: previewPurchase,
  organizationsPurchaseSeats: purchaseSeats,
  organizationsDescribeOrganizationBilling: vi.fn(),
}));
vi.mock("@/contexts/useAccount", () => ({ useAccount: () => ({ account: {} }) }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canManageBilling, isLoading: permissionsLoading }),
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => false, isLoading: featuresLoading, organizationReady }),
}));
vi.mock("@/hooks/useBillingProductEnabled", () => ({ useBillingProductEnabled: () => true }));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Organization" } } }),
}));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({ data: billing, isLoading: false, refetch }),
  useOrganizationSubscriptionActions: () => ({
    cancelPending: false,
    keepPending: false,
    onCancelSubscription: vi.fn(),
    onKeepSubscription: vi.fn(),
  }),
  syncOrganizationBilling: syncBilling,
}));
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({ data: {}, isLoading: false, refetch }),
}));
vi.mock("@/hooks/useOrganizationCreditGrants", () => ({
  useOrganizationCreditGrants: () => ({ data: { grants: [] }, isLoading: false, refetch }),
}));
vi.mock("@/hooks/useHostedCreditActions", () => ({
  useHostedCreditActions: () => ({ products: [] }),
  useHostedCreditOwnerContactMessage: () => undefined,
}));
vi.mock("@/hooks/useBillingCardSetup", () => ({
  useBillingCardSetup: () => ({ isPending: false, error: "", open: vi.fn() }),
}));
vi.mock("@/hooks/useOrganizationBillingSync", () => ({ useOrganizationBillingSync: () => undefined }));
vi.mock("@/hooks/useGooglePurchaseTracking", () => ({ useGooglePurchaseTracking: () => false }));
vi.mock("@/hooks/useHostedCreditReturnRefresh", () => ({ useHostedCreditReturnRefresh: () => "idle" }));
vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: () => undefined }));
vi.mock("@/hooks/useReportPageReady", () => ({ useReportPageReady: () => undefined }));
vi.mock("./BillingPaymentSettings", () => ({ BillingPaymentSettings: () => null }));
vi.mock("./HostedCreditTopUpBanner", () => ({ HostedCreditTopUpBanner: () => null }));

import { OrganizationSettingsBillingPage } from "./OrganizationSettingsBillingPage";

const initialUrl = "/org-1/workspaces/WS/settings/organization/billing?buySeats=1&tab=seats&tag=one&tag=two";

function CurrentLocation() {
  const location = useLocation();
  return <output aria-label="Current billing URL">{location.pathname + location.search + location.hash}</output>;
}

function BillingRoute({ client }: { client: QueryClient }) {
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initialUrl]}>
        <Routes>
          <Route
            path="/:organizationId/workspaces/:factoryKey/settings/organization/billing"
            element={
              <>
                <OrganizationSettingsBillingPage />
                <CurrentLocation />
              </>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  billing = { ...TEAM_ORGANIZATION_BILLING };
  permissionsLoading = false;
  canManageBilling = true;
  featuresLoading = false;
  organizationReady = true;
  previewPurchase.mockReset().mockResolvedValue({
    data: {
      purchasedSeats: 10,
      targetSeats: 11,
      subscriptionId: "sub_team",
      billingPeriodStart: "2026-10-01T00:00:00Z",
      periodEnd: "2026-11-01T00:00:00Z",
    },
  });
  purchaseSeats.mockReset();
  syncBilling.mockReset().mockImplementation(async () => billing);
});

describe("billing purchase deep-link readiness", () => {
  it("retains the link while permissions load, then consumes only buySeats", async () => {
    permissionsLoading = true;
    canManageBilling = false;
    const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
    const view = render(<BillingRoute client={client} />);
    expect(screen.getByLabelText("Current billing URL")).toHaveTextContent(initialUrl);
    expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument();
    expect(previewPurchase).not.toHaveBeenCalled();
    expect(purchaseSeats).not.toHaveBeenCalled();

    permissionsLoading = false;
    canManageBilling = true;
    view.rerender(<BillingRoute client={client} />);

    expect(await screen.findByRole("dialog", { name: "Buy seats" })).toBeInTheDocument();
    await waitFor(() => expect(previewPurchase).toHaveBeenCalledTimes(1));
    expect(screen.getByLabelText("Current billing URL")).toHaveTextContent(
      "/org-1/workspaces/WS/settings/organization/billing?tab=seats&tag=one&tag=two",
    );
    expect(purchaseSeats).not.toHaveBeenCalled();
  });
  it.each([
    { status: "succeeded", age: 23, visible: true, message: "Seats added." },
    { status: "succeeded", age: 25, visible: false, message: "Seats added." },
    { status: "failed", age: 25, visible: false, message: "Card declined." },
    { status: "failed", age: 1, visible: true, message: "Card declined." },
    { status: "action_required", age: 1, visible: true, message: "Confirm with your bank." },
    { status: "pending", age: 25, visible: true, message: "Waiting for payment confirmation." },
    { status: "awaiting_confirmation", age: 25, visible: true, message: "Waiting for payment confirmation." },
  ])("filters a $status purchase aged $age hours", async ({ status, age, visible, message }) => {
    canManageBilling = false;
    billing = {
      ...TEAM_ORGANIZATION_BILLING,
      lastSeatPurchase: {
        id: "purchase-1",
        status,
        message,
        createdAt: new Date(Date.now() - age * 60 * 60 * 1000).toISOString(),
      },
    };
    render(<BillingRoute client={new QueryClient()} />);
    await screen.findByTestId("billing-seats");
    if (visible) expect(screen.getByText(message)).toBeInTheDocument();
    else expect(screen.queryByText(message)).not.toBeInTheDocument();
  });

  it("does not revive an old payment failure when the dialog opens", async () => {
    const message = "This old card was declined.";
    billing = {
      ...TEAM_ORGANIZATION_BILLING,
      lastSeatPurchase: {
        id: "old-purchase",
        status: "failed",
        message,
        createdAt: new Date(Date.now() - 25 * 60 * 60 * 1000).toISOString(),
      },
    };
    render(<BillingRoute client={new QueryClient()} />);
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(within(dialog).queryByText(message)).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Update payment method" })).not.toBeInTheDocument();
    await waitFor(() =>
      expect(within(dialog).getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" }).disabled).toBe(false),
    );
  });
  it.each([true, false])(
    "keeps inactive-subscription payment recovery gated by billing permission (%s)",
    async (canManage) => {
      canManageBilling = canManage;
      billing = {
        ...TEAM_ORGANIZATION_BILLING,
        polarSubscriptionStatus: "past_due",
        creditPurchaseAllowed: false,
        lastSeatPurchase: {
          id: "failed-purchase",
          status: "action_required",
          message: "Confirm the payment with your bank.",
          createdAt: new Date().toISOString(),
        },
      };
      render(<BillingRoute client={new QueryClient()} />);
      const card = await screen.findByTestId("billing-seats");
      expect(screen.getByText("Confirm the payment with your bank.")).toBeInTheDocument();
      if (canManage) expect(screen.getByRole("button", { name: "Update payment method" })).toBeInTheDocument();
      else expect(screen.queryByRole("button", { name: "Update payment method" })).not.toBeInTheDocument();
      expect(within(card).queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
      expect(previewPurchase).not.toHaveBeenCalled();
      expect(purchaseSeats).not.toHaveBeenCalled();
    },
  );

  it.each(["past_due", "paused"])(
    "keeps timed-out purchase recovery visible after the subscription becomes %s",
    async (status) => {
      const client = new QueryClient();
      const view = render(<BillingRoute client={client} />);
      const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
      await waitFor(() =>
        expect(screen.getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" }).disabled).toBe(false),
      );
      billing = {
        ...billing,
        lastSeatPurchase: {
          id: "pending-purchase",
          status: "awaiting_confirmation",
          createdAt: new Date().toISOString(),
        },
      };
      purchaseSeats.mockResolvedValue({ data: { billing, purchase: billing.lastSeatPurchase } });
      fireEvent.click(within(dialog).getByRole("button", { name: "Confirm purchase" }));
      await within(dialog).findByText("Waiting for payment confirmation.");
      billing = { ...billing, polarSubscriptionStatus: status, creditPurchaseAllowed: false };
      view.rerender(<BillingRoute client={client} />);
      expect(screen.getByRole("dialog", { name: "Buy seats" })).toBeInTheDocument();
      expect(within(dialog).queryByRole("spinbutton")).not.toBeInTheDocument();
      expect(within(dialog).queryByRole("button", { name: "Confirm purchase" })).not.toBeInTheDocument();
      vi.useFakeTimers();
      try {
        // Re-enter the polling effect with an already expired window.
        vi.setSystemTime(Date.now() + 120_001);
        billing = { ...billing, lastSeatPurchase: { ...billing.lastSeatPurchase, status: "pending" } };
        view.rerender(<BillingRoute client={client} />);
        await act(async () => {});
      } finally {
        vi.useRealTimers();
      }
      fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
      expect(screen.getByText("Waiting for payment confirmation.")).toBeInTheDocument();
      const requestsBeforeRefresh = syncBilling.mock.calls.length;
      fireEvent.click(screen.getByRole("button", { name: "Refresh billing" }));
      await waitFor(() => expect(syncBilling).toHaveBeenCalledTimes(requestsBeforeRefresh + 1));
      expect(purchaseSeats).toHaveBeenCalledTimes(1);
      expect(previewPurchase).toHaveBeenCalledTimes(1);
    },
  );
});
