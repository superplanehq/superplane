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
    isLoading: false,
  }),
}));

function renderBilling(billing: OrganizationsDescribeOrganizationBillingResponse) {
  return render(
    <FactoriesHarness
      pathSuffix={"workspaces/" + PRIMARY_FACTORY_KEY + "/settings/organization/billing"}
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
    expect(detail(card, "Plan maximum")).toHaveTextContent("80");
    expect(detail(card, "Current monthly total")).toHaveTextContent("$290.00");
    expect(detail(card, "Next renewal")).toHaveTextContent(date);
    expect(detail(card, "Expected renewal")).toHaveTextContent(
      "9 seats, $261.00 on " + date + ", based on current members.",
    );
    expect(card).toHaveTextContent("Estimates exclude taxes and discounts.");
    expect(card).toHaveTextContent(
      "Unused seats are removed at renewal. Pending invitations are not renewed and expire at renewal.",
    );
    expect(within(card).queryByRole("button")).not.toBeInTheDocument();
    expect(
      screen.getByTestId("billing-plans").compareDocumentPosition(card) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      card.compareDocumentPosition(screen.getByTestId("billing-credit-balance")) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  it.each([
    { name: "legacy zero capacity", occupied: 9 },
    { name: "no current members", occupied: 0 },
  ])("preserves zero values with $name", async ({ occupied }) => {
    renderBilling({
      ...TEAM_ORGANIZATION_BILLING,
      purchasedSeats: 0,
      occupiedSeats: occupied,
      availableSeats: 0,
      expectedRenewalSeats: Math.max(occupied, 1),
      currentMonthlyTotalCents: "0",
      expectedRenewalCents: String(Math.max(occupied, 1) * 2900),
      planMaximumSeats: 0,
    });
    const card = await screen.findByTestId("billing-seats");
    expect(card).toHaveTextContent(occupied + " of 0 seats used");
    expect(detail(card, "Available seats")).toHaveTextContent("0");
    expect(detail(card, "Plan maximum")).toHaveTextContent("0");
    expect(detail(card, "Current monthly total")).toHaveTextContent("$0.00");
    expect(detail(card, "Expected renewal")).toHaveTextContent(
      Math.max(occupied, 1) + (occupied === 0 ? " seat, $29.00" : " seats, $261.00"),
    );
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
    await screen.findByTestId("billing-plans");
    expect(screen.queryByTestId("billing-seats")).not.toBeInTheDocument();
  });

  it.each([
    { name: "past due after the grace period", status: "past_due", planPerSeat: false },
    { name: "past due during the grace period", status: "past_due", planPerSeat: true },
    { name: "paused", status: "paused", planPerSeat: true },
  ])("keeps a seats summary and blocks new members while $name", async ({ status, planPerSeat }) => {
    renderBilling({
      ...TEAM_ORGANIZATION_BILLING,
      polarSubscriptionStatus: status,
      planPerSeat,
      availableSeats: undefined,
      planMaximumSeats: undefined,
      purchasableSeats: undefined,
      expectedRenewalSeats: undefined,
      currentMonthlyTotalCents: undefined,
      expectedRenewalCents: undefined,
    });
    const card = await screen.findByTestId("billing-seats");
    expect(card).toHaveTextContent("9 of 10 seats used");
    expect(card).toHaveTextContent("No one can join until the payment is updated.");
    expect(within(card).queryByText("Expected renewal")).not.toBeInTheDocument();
    expect(within(card).queryByText("Not available", { selector: "dd" })).not.toBeInTheDocument();
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
