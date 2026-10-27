import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

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
    hasPermission: (resource: string, action: string) => resource !== "org" || action !== "update" || canManageBilling,
    isLoading: false,
  }),
}));

function renderBilling(
  billing: OrganizationsDescribeOrganizationBillingResponse,
  options: { query?: string; fixture?: Partial<FactoriesFixture> } = {},
) {
  return render(
    <FactoriesHarness
      pathSuffix={"workspaces/" + PRIMARY_FACTORY_KEY + "/settings/organization/billing" + (options.query ?? "")}
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
        query: entry === "deep link" ? "?buySeats=1&tab=seats" : undefined,
        fixture: {
          seatPurchasePreview: {
            purchasedSeats: 10,
            targetSeats: 11,
            proratedChargeCents: "1234",
            newMonthlyTotalCents: "35123",
            periodEnd: ACTIVE_BILLING_PERIOD_ENDS_AT,
            subscriptionId: "sub_team",
            billingPeriodStart: "2026-10-01T00:00:00Z",
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
    renderBilling({ ...TEAM_ORGANIZATION_BILLING }, { query: "?buySeats=1" });
    const card = await screen.findByTestId("billing-seats");
    expect(card).toHaveTextContent("Ask an organization admin to buy seats.");
    expect(screen.getByText("Ask an organization admin to manage billing.")).toBeInTheDocument();
    expect(within(card).queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
    await new Promise((resolve) => setTimeout(resolve, 350));
    expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument();
    expect(purchaseRequests).toHaveLength(0);
  });

  it("clears the over-capacity notice after buying enough seats", async () => {
    const user = userEvent.setup();
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, purchasedSeats: 8, occupiedSeats: 9, availableSeats: 0 }, {});
    const card = await screen.findByTestId("billing-seats");
    expect(within(card).getByRole("status")).toHaveTextContent(
      "9 members, 8 paid seats. Buy seats or remove members to invite again.",
    );
    await user.click(within(card).getByRole("button", { name: "Buy seats" }));
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    const confirm = within(dialog).getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" });
    await waitFor(() => expect(confirm.disabled).toBe(false));
    await user.click(confirm);
    expect(await within(dialog).findByText("Seats added.")).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(card).toHaveTextContent("9 of 9 seats used"));
    expect(within(card).queryByRole("status")).not.toBeInTheDocument();
  });

  it.each([
    { name: "admin-sourced plan", overrides: { planSource: "admin" } },
    { name: "inactive subscription", overrides: { polarSubscriptionStatus: "canceled" } },
  ])("does not offer purchases for $name", async ({ overrides }) => {
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, ...overrides }, { query: "?buySeats=1" });
    const card = await screen.findByTestId("billing-seats");
    expect(within(card).queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
    await new Promise((resolve) => setTimeout(resolve, 350));
    expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument();
    expect(purchaseRequests).toHaveLength(0);
  });

  it("explains the plan maximum without purchase controls", async () => {
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, purchasableSeats: 0 }, { query: "?buySeats=1" });
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
      { query: "?buySeats=1" },
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
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, cancelAtPeriodEnd: true }, { query: "?buySeats=1" });
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(dialog).toHaveTextContent("Resume the subscription before buying seats.");
    expect(within(dialog).queryByRole("spinbutton")).not.toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Confirm purchase" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument());
    await waitFor(() =>
      expect(within(screen.getByTestId("billing-plans")).getByRole("button", { name: "Keep Team" })).toBeVisible(),
    );
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
      { query: "?buySeats=1", fixture: { seatPurchasePreviewError: { status: 409, message } } },
    );
    const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(message);
    expect(within(dialog).getByRole("button", { name: "Confirm purchase" })).toBeDisabled();
    expect(purchaseRequests.filter((url) => url.endsWith("/purchase"))).toHaveLength(0);
  });

  it.each([
    { status: "failed", reopen: false },
    { status: "action_required", reopen: false },
    { status: "failed", reopen: true },
  ])(
    "requires a fresh estimate for the current quantity after card replacement ($status, reopen=$reopen)",
    async ({ status, reopen }) => {
      const user = userEvent.setup();
      const configuredFetch = client.getConfig().fetch;
      const popup = { closed: false, close: vi.fn(), location: { replace: vi.fn() }, opener: null };
      const openPopup = vi.spyOn(window, "open").mockReturnValue(popup as unknown as Window);
      const failed = {
        id: "declined-purchase",
        status,
        message: "Replace your card.",
        createdAt: new Date().toISOString(),
      };
      let attempts = 0;
      client.setConfig({
        fetch: (input, init) => {
          const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
          const response = (body: unknown, status = 200) =>
            Promise.resolve(
              new Response(JSON.stringify(body), {
                status,
                headers: { "Content-Type": "application/json" },
              }),
            );
          if (url.endsWith("/billing/seats/purchase")) {
            attempts += 1;
            if (attempts === 1) return response({ message: failed.message }, 412);
          }
          if (attempts === 1 && (url.endsWith("/billing") || url.endsWith("/billing/sync"))) {
            return response({ ...TEAM_ORGANIZATION_BILLING, hasPaymentMethod: true, lastSeatPurchase: failed });
          }
          if (url.endsWith("/billing/payment-method-session")) {
            return response({ portalUrl: "https://polar.example/portal" });
          }
          return (configuredFetch ?? window.fetch)(input, init);
        },
      });
      try {
        renderBilling({ ...TEAM_ORGANIZATION_BILLING, hasPaymentMethod: true }, { query: "?buySeats=1" });
        let dialog = await screen.findByRole("dialog", { name: "Buy seats" });
        const confirm = () => screen.getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" });
        const quantity = within(dialog).getByRole("spinbutton", { name: "Additional seats" });
        await user.clear(quantity);
        await user.type(quantity, "3");
        await waitFor(() => expect(confirm().disabled).toBe(false));
        await user.click(confirm());
        await within(dialog).findByText(failed.message);
        expect(confirm().disabled).toBe(true);
        await user.click(within(dialog).getByRole("button", { name: "Update payment method" }));
        await waitFor(() => expect(popup.location.replace).toHaveBeenCalled());
        expect(attempts).toBe(1);
        if (reopen) {
          await user.click(within(dialog).getByRole("button", { name: "Close" }));
          await user.click(within(screen.getByTestId("billing-seats")).getByRole("button", { name: "Buy seats" }));
          dialog = await screen.findByRole("dialog", { name: "Buy seats" });
          const currentQuantity = within(dialog).getByRole("spinbutton", { name: "Additional seats" });
          expect(currentQuantity).toHaveValue(1);
          await waitFor(() => expect(dialog).toHaveTextContent("New monthly total: $319.00"));
        }
        const estimatesBeforeSetup = purchaseRequests.filter((url) => url.endsWith("/preview")).length;
        popup.closed = true;
        await waitFor(() => expect(confirm().disabled).toBe(false), { timeout: 4000 });
        expect(within(dialog).getByRole("spinbutton", { name: "Additional seats" })).toHaveValue(reopen ? 1 : 3);
        expect(dialog).toHaveTextContent(reopen ? "New monthly total: $319.00" : "New monthly total: $377.00");
        expect(purchaseRequests.filter((url) => url.endsWith("/preview"))).toHaveLength(estimatesBeforeSetup + 1);
        expect(attempts).toBe(1);
        await user.click(confirm());
        await within(dialog).findByText("Seats added.");
        expect(attempts).toBe(2);
        await user.click(within(dialog).getByRole("button", { name: "Close" }));
        await waitFor(() =>
          expect(screen.getByTestId("billing-seats")).toHaveTextContent(`9 of ${reopen ? 11 : 13} seats used`),
        );
      } finally {
        client.setConfig({ fetch: configuredFetch });
        openPopup.mockRestore();
      }
    },
  );

  it("keeps the purchase form when a new attempt is rejected after an earlier success", async () => {
    const user = userEvent.setup();
    const message = "Seat purchases pause while the subscription renews. Try again after renewal.";
    // The harness deep-clones its fixture, so reject the second attempt at the
    // transport layer instead of mutating the fixture mid-test.
    const configuredFetch = client.getConfig().fetch;
    let purchaseAttempts = 0;
    client.setConfig({
      fetch: (input, init) => {
        const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
        if (url.endsWith("/billing/seats/purchase")) {
          purchaseAttempts += 1;
          if (purchaseAttempts > 1) {
            return Promise.resolve(
              new Response(JSON.stringify({ message }), {
                status: 412,
                headers: { "Content-Type": "application/json" },
              }),
            );
          }
        }
        return (configuredFetch ?? window.fetch)(input, init);
      },
    });
    try {
      renderBilling({ ...TEAM_ORGANIZATION_BILLING }, { query: "?buySeats=1" });
      const dialog = await screen.findByRole("dialog", { name: "Buy seats" });
      await waitFor(() =>
        expect(within(dialog).getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" }).disabled).toBe(
          false,
        ),
      );
      await user.click(within(dialog).getByRole("button", { name: "Confirm purchase" }));
      expect(await within(dialog).findByText("Seats added.")).toBeInTheDocument();
      expect(within(dialog).queryByRole("spinbutton")).not.toBeInTheDocument();
      await user.click(within(dialog).getByRole("button", { name: "Close" }));
      const card = screen.getByTestId("billing-seats");
      await user.click(within(card).getByRole("button", { name: "Buy seats" }));
      const reopened = await screen.findByRole("dialog", { name: "Buy seats" });
      expect(within(reopened).getByRole("spinbutton", { name: "Additional seats" })).toBeInTheDocument();
      await waitFor(() =>
        expect(within(reopened).getByRole<HTMLButtonElement>("button", { name: "Confirm purchase" }).disabled).toBe(
          false,
        ),
      );
      await user.click(within(reopened).getByRole("button", { name: "Confirm purchase" }));
      expect(await within(reopened).findByText(message)).toBeInTheDocument();
      expect(within(reopened).queryByText("Seats added.")).not.toBeInTheDocument();
      expect(within(reopened).getByRole("spinbutton", { name: "Additional seats" })).toBeInTheDocument();
      expect(within(reopened).getByRole("button", { name: "Confirm purchase" })).toBeInTheDocument();
      expect(within(reopened).queryByRole("button", { name: "Refresh billing" })).not.toBeInTheDocument();
      expect(purchaseRequests.filter((url) => url.endsWith("/purchase"))).toHaveLength(1);
    } finally {
      client.setConfig({ fetch: configuredFetch });
    }
  });
});
