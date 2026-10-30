import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  BUSINESS_ORGANIZATION_BILLING,
  TEAM_ORGANIZATION_BILLING,
  ACTIVE_BILLING_PERIOD_ENDS_AT,
} from "../../__fixtures__/usageReportFixtures";

let canManageBilling = true;
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => resource !== "org" || action !== "update" || canManageBilling,
    hasPermission: (resource: string, action: string) => resource !== "org" || action !== "update" || canManageBilling,
    isLoading: false,
  }),
}));

function renderBilling(billing: OrganizationsDescribeOrganizationBillingResponse, buySeats = false) {
  return render(
    <FactoriesHarness
      pathSuffix={
        "workspaces/" + PRIMARY_FACTORY_KEY + "/settings/organization/billing" + (buySeats ? "?buySeats=1" : "")
      }
      factoriesFixture={{ ...defaultFactoriesFixture, organizationBilling: billing }}
    />,
  );
}

function detail(card: HTMLElement, label: string) {
  const term = within(card).getByText(label, { selector: "dt" });
  return term.nextElementSibling;
}

describe("Organization billing seats", () => {
  beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));
  beforeEach(() => {
    canManageBilling = true;
  });

  it("shows purchased capacity and API totals independently of the catalog price to members", async () => {
    canManageBilling = false;
    // Different catalog price catches accidental recalculation in the browser.
    renderBilling({ ...TEAM_ORGANIZATION_BILLING, planPriceCents: "3100", planMaximumSeats: 80 });
    const card = await screen.findByTestId("billing-seats");
    const date = new Date(ACTIVE_BILLING_PERIOD_ENDS_AT).toLocaleDateString();
    expect(card).toHaveTextContent("9 of 10 seats used");
    expect(detail(card, "Available seats")).toHaveTextContent("1");
    expect(detail(card, "Pending invitations")).toHaveTextContent("2");
    expect(detail(card, "Seats available for new invitations")).toHaveTextContent("0");
    expect(detail(card, "Plan maximum")).toHaveTextContent("80");
    expect(detail(card, "Current monthly total")).toHaveTextContent("$290.00");
    expect(detail(card, "Next renewal")).toHaveTextContent(date);
    expect(detail(card, "Expected renewal")).toHaveTextContent(
      "9 seats, $261.00 on " + date + ", based on current members.",
    );
    expect(card).toHaveTextContent("Estimates exclude taxes and discounts.");
    expect(card).toHaveTextContent(
      "Pending invitations do not count at renewal. After renewal, an invitation needs a free seat.",
    );
    expect(within(card).queryByRole("button")).not.toBeInTheDocument();
    expect(card).toHaveTextContent("Ask an organization admin to buy seats.");
    expect(
      screen.getByTestId("billing-plans").compareDocumentPosition(card) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      card.compareDocumentPosition(screen.getByTestId("billing-credit-balance")) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it("shows the members, minimum one, as the renewal estimate", async () => {
    renderBilling({
      ...TEAM_ORGANIZATION_BILLING,
      pendingInvitationSeats: 0,
      occupiedSeats: 0,
      availableSeats: 10,
      invitableSeats: 10,
      expectedRenewalSeats: 1,
      expectedRenewalCents: "2900",
    });
    const card = await screen.findByTestId("billing-seats");
    expect(card).toHaveTextContent("0 of 10 seats used");
    expect(detail(card, "Pending invitations")).toHaveTextContent("0");
    expect(detail(card, "Expected renewal")).toHaveTextContent("1 seat, $29.00 on ");
    expect(within(card).queryByRole("status")).not.toBeInTheDocument();
    expect(within(card).queryByTestId("billing-seats-renewal-cap")).not.toBeInTheDocument();
  });

  it.each([
    { name: "zero paid seats", purchased: 0 },
    { name: "missing paid seats", purchased: undefined },
  ])("shows an unavailable renewal estimate with $name", async ({ purchased }) => {
    renderBilling({
      ...TEAM_ORGANIZATION_BILLING,
      purchasedSeats: purchased,
      pendingInvitationSeats: 0,
      invitableSeats: 0,
      availableSeats: 0,
      expectedRenewalSeats: undefined,
      currentMonthlyTotalCents: "0",
      expectedRenewalCents: undefined,
    });
    const card = await screen.findByTestId("billing-seats");
    expect(detail(card, "Current monthly total")).toHaveTextContent("$0.00");
    expect(detail(card, "Expected renewal")).toHaveTextContent(
      "Not available. The paid seat count needs repair. Contact support.",
    );
    expect(detail(card, "Expected renewal")).not.toHaveTextContent("seats,");
    expect(within(card).queryByTestId("billing-seats-renewal-cap")).not.toBeInTheDocument();
  });

  describe("when members exceed paid seats", () => {
    const overCapacity = { ...TEAM_ORGANIZATION_BILLING, purchasedSeats: 8, occupiedSeats: 9, availableSeats: 0 };

    it("tells admins to buy seats or remove members", async () => {
      renderBilling(overCapacity);
      const card = await screen.findByTestId("billing-seats");
      expect(card).toHaveTextContent("9 members, 8 paid seats. Buy seats or remove members to invite again.");
      expect(card).not.toHaveTextContent("Ask an organization admin");
    });

    it("uses singular forms for one member and one paid seat", async () => {
      renderBilling({ ...overCapacity, purchasedSeats: 1, occupiedSeats: 2 });
      expect(await screen.findByTestId("billing-seats")).toHaveTextContent("2 members, 1 paid seat. Buy seats");
    });

    it.each([true, false])(
      "bills only the paid seats at renewal and explains the cap (can manage billing: %s)",
      async (canManage) => {
        canManageBilling = canManage;
        renderBilling({
          ...overCapacity,
          purchasedSeats: 1,
          occupiedSeats: 3,
          expectedRenewalSeats: 1,
          currentMonthlyTotalCents: "2900",
          expectedRenewalCents: "2900",
        });
        const card = await screen.findByTestId("billing-seats");
        expect(detail(card, "Expected renewal")).toHaveTextContent("1 seat, $29.00 on ");
        expect(within(card).getByRole("status")).toHaveTextContent(
          canManage ? "3 members, 1 paid seat." : "Ask an organization admin to buy seats or remove members.",
        );
        expect(within(card).getByTestId("billing-seats-renewal-cap")).toHaveTextContent(
          "Renewal bills 1 paid seat. Renewal does not add seats.",
        );
      },
    );

    it("tells other users to ask an admin", async () => {
      canManageBilling = false;
      renderBilling(overCapacity);
      const card = await screen.findByTestId("billing-seats");
      expect(card).toHaveTextContent("Ask an organization admin to buy seats or remove members.");
      expect(card).not.toHaveTextContent("Buy seats or remove members to invite again.");
    });

    it.each([8, 7])("shows no notice when occupied seats do not exceed %s purchased seats", async (occupied) => {
      renderBilling({ ...overCapacity, occupiedSeats: occupied });
      const card = await screen.findByTestId("billing-seats");
      expect(within(card).queryByRole("status")).not.toBeInTheDocument();
    });
  });

  it.each([undefined, "", "invalid-date"])(
    "handles an unavailable renewal date (%s) and amounts",
    async (currentPeriodEnd) => {
      renderBilling({
        ...TEAM_ORGANIZATION_BILLING,
        currentPeriodEnd,
        currentMonthlyTotalCents: undefined,
        expectedRenewalCents: undefined,
        planMaximumSeats: undefined,
        purchasableSeats: undefined,
      });
      const card = await screen.findByTestId("billing-seats");
      expect(detail(card, "Plan maximum")).toHaveTextContent("Unlimited");
      expect(detail(card, "Current monthly total")).toHaveTextContent("Not available");
      expect(detail(card, "Next renewal")).toHaveTextContent("Not available");
      expect(detail(card, "Expected renewal")).toHaveTextContent("9 seats, Not available, based on current members.");
      expect(card).not.toHaveTextContent("Invalid Date");
    },
  );

  it.each([
    { name: "Business", billing: BUSINESS_ORGANIZATION_BILLING },
    { name: "trial", billing: { ...BUSINESS_ORGANIZATION_BILLING, plan: "trial", planPerSeat: false } },
    { name: "None", billing: { ...BUSINESS_ORGANIZATION_BILLING, plan: "none", planPerSeat: false } },
    { name: "billing off", billing: { ...TEAM_ORGANIZATION_BILLING, billingEnabled: false } },
  ])("hides seats for $name", async ({ billing }) => {
    renderBilling(billing);
    await screen.findByTestId("billing-credit-balance");
    expect(screen.queryByTestId("billing-seats")).not.toBeInTheDocument();
  });

  describe("while payment is blocked", () => {
    // After the fallback, the plan row is on the access plan. The renewal
    // estimate still uses the paid seat price from the server.
    const pastDue = {
      ...TEAM_ORGANIZATION_BILLING,
      polarSubscriptionStatus: "past_due",
      planPerSeat: false,
      planPriceCents: "0",
      availableSeats: undefined,
      planMaximumSeats: undefined,
      purchasableSeats: undefined,
      currentMonthlyTotalCents: undefined,
    };
    const date = new Date(ACTIVE_BILLING_PERIOD_ENDS_AT).toLocaleDateString();

    function expectNoSeatPurchase() {
      expect(screen.queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
      expect(screen.queryByRole("dialog", { name: "Buy seats" })).not.toBeInTheDocument();
    }

    it.each([
      { name: "during the grace period", planPerSeat: true },
      { name: "after the grace period", planPerSeat: false },
    ])("shows the renewal and blocks new members $name", async ({ planPerSeat }) => {
      renderBilling({ ...pastDue, planPerSeat }, true);
      const card = await screen.findByTestId("billing-seats");
      expect(card).toHaveTextContent("9 of 10 seats used");
      expect(card).toHaveTextContent("No one can join until the payment is updated.");
      expect(detail(card, "Next renewal")).toHaveTextContent(date);
      expect(detail(card, "Expected renewal")).toHaveTextContent(
        "9 seats, $261.00 on " + date + ", based on current members.",
      );
      expect(within(card).queryByRole("status")).not.toBeInTheDocument();
      expectNoSeatPurchase();
    });

    it("shows an unavailable amount and date instead of zero", async () => {
      renderBilling({ ...pastDue, expectedRenewalCents: undefined, currentPeriodEnd: undefined }, true);
      const card = await screen.findByTestId("billing-seats");
      expect(detail(card, "Next renewal")).toHaveTextContent("Not available");
      expect(detail(card, "Expected renewal")).toHaveTextContent("9 seats, Not available, based on current members.");
      expect(card).not.toHaveTextContent("$0.00");
    });

    it("shows the repair text without paid seats", async () => {
      renderBilling({
        ...pastDue,
        purchasedSeats: 0,
        expectedRenewalSeats: undefined,
        expectedRenewalCents: undefined,
      });
      const card = await screen.findByTestId("billing-seats");
      expect(detail(card, "Expected renewal")).toHaveTextContent(
        "Not available. The paid seat count needs repair. Contact support.",
      );
      expect(detail(card, "Expected renewal")).not.toHaveTextContent("seats,");
    });

    it.each([true, false])(
      "shows members above the paid seats with payment recovery steps (can manage billing: %s)",
      async (canManage) => {
        canManageBilling = canManage;
        renderBilling(
          { ...pastDue, purchasedSeats: 1, occupiedSeats: 3, expectedRenewalSeats: 1, expectedRenewalCents: "2900" },
          true,
        );
        const card = await screen.findByTestId("billing-seats");
        expect(within(card).getByRole("status")).toHaveTextContent(
          canManage
            ? "3 members, 1 paid seat. Update the payment method before you add seats or members."
            : "Ask an organization admin to update the payment.",
        );
        expect(card).not.toHaveTextContent("Buy seats");
        expect(within(card).getByTestId("billing-seats-renewal-cap")).toHaveTextContent(
          "Renewal bills 1 paid seat. Renewal does not add seats.",
        );
        expect(detail(card, "Expected renewal")).toHaveTextContent("1 seat, $29.00 on " + date);
        expectNoSeatPurchase();
      },
    );

    it("shows only a conditional estimate without a date while paused", async () => {
      renderBilling({ ...pastDue, polarSubscriptionStatus: "paused" }, true);
      const card = await screen.findByTestId("billing-seats");
      expect(card).toHaveTextContent("No one can join until the payment is updated.");
      expect(detail(card, "Renewal estimate if the subscription resumes")).toHaveTextContent(
        "9 seats, $261.00, based on current members.",
      );
      expect(within(card).queryByText("Expected renewal")).not.toBeInTheDocument();
      expect(within(card).queryByText("Next renewal")).not.toBeInTheDocument();
      expect(card).not.toHaveTextContent(date);
      expectNoSeatPurchase();
    });

    it("shows only a conditional estimate when the subscription is set to end", async () => {
      renderBilling({ ...pastDue, cancelAtPeriodEnd: true }, true);
      const card = await screen.findByTestId("billing-seats");
      expect(detail(card, "Subscription ends")).toHaveTextContent(date);
      expect(detail(card, "Renewal estimate if you keep the subscription")).toHaveTextContent("9 seats, $261.00 on ");
      expect(within(card).queryByText("Expected renewal")).not.toBeInTheDocument();
      expectNoSeatPurchase();
    });
  });

  it("shows a conditional estimate after cancellation and normal copy after resuming", async () => {
    const user = userEvent.setup();
    renderBilling({ ...TEAM_ORGANIZATION_BILLING });
    const card = await screen.findByTestId("billing-seats");
    await user.click(screen.getByRole("button", { name: "Cancel Team" }));
    await user.click(screen.getByTestId("billing-cancel-subscription-confirm"));
    expect(await within(card).findByText("Subscription ends")).toBeInTheDocument();
    expect(detail(card, "Renewal estimate if you keep the subscription")).toHaveTextContent("9 seats, $261.00 on ");
    expect(within(card).queryByText("Next renewal")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Keep Team" }));
    expect(await within(card).findByText("Next renewal")).toBeInTheDocument();
    expect(detail(card, "Expected renewal")).toHaveTextContent("9 seats, $261.00 on ");
    expect(within(card).queryByText("Renewal estimate if you keep the subscription")).not.toBeInTheDocument();
  }, 10000);
});
