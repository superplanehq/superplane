import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { FEATURE_NEW_BILLING } from "@/lib/experimentalFeatures";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../../__fixtures__/factoryPageResponses";
import { TEAM_ORGANIZATION_BILLING, ACTIVE_BILLING_PERIOD_ENDS_AT } from "../../__fixtures__/usageReportFixtures";

let canManageBilling = true;
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => resource !== "org" || action !== "update" || canManageBilling,
    isLoading: false,
  }),
}));

function renderBilling(
  billing: OrganizationsDescribeOrganizationBillingResponse,
  options: { query?: string; newBilling?: boolean; fixture?: Partial<FactoriesFixture> } = {},
) {
  return render(
    <FactoriesHarness
      pathSuffix={"workspaces/" + PRIMARY_FACTORY_KEY + "/settings/organization/billing" + (options.query ?? "")}
      experimentalFeatures={options.newBilling ? [FEATURE_NEW_BILLING] : []}
      factoriesFixture={{ ...defaultFactoriesFixture, organizationBilling: billing, ...options.fixture }}
    />,
  );
}

const purchaseRequests: string[] = [];

describe("Organization billing seat purchases", () => {
  beforeAll(() =>
    client.setConfig({
      baseUrl: "http://localhost",
      fetch: (input, init) => {
        const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
        if (url.includes("/billing/seats/")) purchaseRequests.push(url);
        return window.fetch(input, init);
      },
    }),
  );
  beforeEach(() => {
    canManageBilling = true;
    purchaseRequests.length = 0;
  });

  it.each(["button", "deep link"])("opens a server estimate through the %s", async (entry) => {
    const user = userEvent.setup();
    renderBilling(
      { ...TEAM_ORGANIZATION_BILLING },
      {
        newBilling: true,
        query: entry === "deep link" ? "?buySeats=1&tab=seats" : undefined,
        fixture: {
          seatPurchasePreview: {
            purchasedSeats: 10,
            targetSeats: 11,
            proratedChargeCents: "1234",
            newMonthlyTotalCents: "35123",
            periodEnd: ACTIVE_BILLING_PERIOD_ENDS_AT,
          },
        },
      },
    );
    const card = await screen.findByTestId("billing-seats");
    if (entry === "button") await user.click(within(card).getByRole("button", { name: "Buy seats" }));
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(within(dialog).getByRole("spinbutton", { name: "Additional seats" })).toHaveValue(1);
    expect(await within(dialog).findByText("Estimated charge now: $12.34")).toBeInTheDocument();
    expect(within(dialog).getByText("New monthly total: $351.23")).toBeInTheDocument();
    expect(
      within(dialog).getByText("today to " + new Date(ACTIVE_BILLING_PERIOD_ENDS_AT).toLocaleDateString()),
    ).toBeInTheDocument();
    expect(within(dialog).getByText("Estimate. Tax is calculated at payment.")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Confirm purchase" }));
    expect(await within(dialog).findByText("Seats added.")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(card).toHaveTextContent("9 of 11 seats used"));
  });

  it("denies the deep link without making purchase requests", async () => {
    canManageBilling = false;
    renderBilling({ ...TEAM_ORGANIZATION_BILLING }, { newBilling: true, query: "?buySeats=1" });
    const card = await screen.findByTestId("billing-seats");
    expect(card).toHaveTextContent("Ask an organization admin to buy seats.");
    expect(screen.getByText("Ask an organization admin to set up payment or change plans.")).toBeInTheDocument();
    expect(within(card).queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
    await new Promise((resolve) => setTimeout(resolve, 350));
    expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument();
    expect(purchaseRequests).toHaveLength(0);
  });

  it.each([
    { name: "admin-sourced plan", overrides: { planSource: "admin" }, newBilling: true },
    { name: "inactive subscription", overrides: { polarSubscriptionStatus: "canceled" }, newBilling: true },
    { name: "New Billing disabled", overrides: {}, newBilling: false },
  ])("does not offer purchases for $name", async ({ overrides, newBilling }) => {
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, ...overrides }, { newBilling, query: "?buySeats=1" });
    const card = await screen.findByTestId("billing-seats");
    expect(within(card).queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
    await new Promise((resolve) => setTimeout(resolve, 350));
    expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument();
    expect(purchaseRequests).toHaveLength(0);
  });

  it("explains the plan maximum without purchase controls", async () => {
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, purchasableSeats: 0 }, { newBilling: true, query: "?buySeats=1" });
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(dialog).toHaveTextContent(
      "Your plan allows 50 seats. Remove a member or cancel an invitation to free a seat.",
    );
    expect(within(dialog).queryByRole("spinbutton")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Confirm purchase" })).not.toBeInTheDocument();
    expect(purchaseRequests).toHaveLength(0);
  });

  it("allows counts beyond the default plan maximum when capacity is unlimited", async () => {
    const user = userEvent.setup();
    renderBilling(
      {
        ...TEAM_ORGANIZATION_BILLING,
        purchasableSeats: undefined,
        planMaximumSeats: undefined,
        invitableSeats: undefined,
      },
      { newBilling: true, query: "?buySeats=1" },
    );
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    const input = within(dialog).getByRole("spinbutton", { name: "Additional seats" });
    expect(input).not.toHaveAttribute("max");
    await user.clear(input);
    await user.type(input, "100");
    await waitFor(() =>
      expect(within(dialog).getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" }).disabled).toBe(false),
    );
  });

  it("blocks purchases until a canceled subscription is resumed", async () => {
    const user = userEvent.setup();
    renderBilling(
      { ...TEAM_ORGANIZATION_BILLING, cancelAtPeriodEnd: true },
      { newBilling: true, query: "?buySeats=1" },
    );
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(dialog).toHaveTextContent("Resume the subscription before buying seats.");
    expect(within(dialog).queryByRole("spinbutton")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Confirm purchase" })).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    expect(within(screen.getByTestId("billing-plans")).getByRole("button", { name: "Keep Team" })).toBeInTheDocument();
    expect(purchaseRequests).toHaveLength(0);
  });

  it.each(["failed", "action_required"])("shows the %s server result with payment recovery", async (status) => {
    const user = userEvent.setup();
    const result = {
      id: "purchase-result",
      status,
      message: "Confirm this payment with your bank.",
      createdAt: new Date().toISOString(),
    };
    renderBilling(
      { ...TEAM_ORGANIZATION_BILLING },
      {
        newBilling: true,
        query: "?buySeats=1",
        fixture: {
          seatPurchaseResponse: {
            purchase: result,
            billing: { ...TEAM_ORGANIZATION_BILLING, lastSeatPurchase: result },
          },
        },
      },
    );
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    await waitFor(() =>
      expect(within(dialog).getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" }).disabled).toBe(false),
    );
    await user.click(within(dialog).getByRole("button", { name: "Confirm purchase" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(result.message);
    expect(within(dialog).getByRole("button", { name: "Update payment method" })).toBeInTheDocument();
    expect(purchaseRequests.filter((url) => url.endsWith("/purchase"))).toHaveLength(1);
  });

  it("shows the server renewal freeze without a purchase", async () => {
    const message = "Seat purchases pause while the subscription renews. Try again after 18:00.";
    renderBilling(
      { ...TEAM_ORGANIZATION_BILLING },
      { newBilling: true, query: "?buySeats=1", fixture: { seatPurchasePreviewError: { status: 409, message } } },
    );
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(message);
    expect(within(dialog).getByRole("button", { name: "Confirm purchase" })).toBeDisabled();
    expect(purchaseRequests.filter((url) => url.endsWith("/purchase"))).toHaveLength(0);
  });
});
