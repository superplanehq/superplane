import { render, screen, within } from "@testing-library/react";
import { beforeAll, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { BUSINESS_ORGANIZATION_BILLING } from "../../__fixtures__/usageReportFixtures";

beforeAll(() => {
  client.setConfig({ baseUrl: "http://localhost" });
});

it("shows the failed renewal, hides plan checkout, and keeps the seats summary", async () => {
  const failedAt = "2026-10-20T10:00:00Z";
  render(
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationBilling: {
          ...BUSINESS_ORGANIZATION_BILLING,
          plan: "none",
          planName: "None",
          planPerSeat: false,
          polarSubscriptionStatus: "past_due",
          paymentFailedAt: failedAt,
          purchasedSeats: 10,
          occupiedSeats: 9,
          availablePlans: [
            { key: "team", name: "Team", perSeat: true, subscriptionRequired: true, priceCents: "2900" },
          ],
        },
      }}
    />,
  );

  const banner = await screen.findByTestId("billing-payment-failed");
  expect(banner).toHaveTextContent(
    `The renewal payment failed on ${new Date(failedAt).toLocaleDateString()}. Team features are off until you update the payment method.`,
  );
  expect(within(banner).getByRole("button", { name: "Update payment method" })).toBeEnabled();
  expect(screen.queryByRole("button", { name: "Subscribe to Team" })).not.toBeInTheDocument();
  expect(screen.queryByRole("spinbutton", { name: "Team seats" })).not.toBeInTheDocument();
  const seats = await screen.findByTestId("billing-seats");
  expect(seats).toHaveTextContent("9 of 10 seats used");
  expect(seats).toHaveTextContent("No one can join until the payment is updated.");
}, 10000);
