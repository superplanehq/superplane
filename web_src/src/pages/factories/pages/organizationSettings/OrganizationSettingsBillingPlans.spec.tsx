import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  ADMIN_ORGANIZATION_BILLING,
  BUSINESS_ORGANIZATION_BILLING,
  DEFAULT_FACTORY_USAGE,
  ENDING_ORGANIZATION_BILLING,
  PURCHASED_CREDIT_USAGE_REPORT,
  RESTORED_TRIAL_ORGANIZATION_BILLING,
} from "../../__fixtures__/usageReportFixtures";
import { BILLING_SPEND_ORDER_COPY, BILLING_SPEND_ORDER_WITH_GRANT_COPY } from "../../lib/billingCreditBuckets";
import { billingSubscriptionEndsCopy } from "../../lib/billingPlans";

const WELCOME_EXPIRY_LABEL = new Date("2026-09-22T12:00:00.000Z").toLocaleDateString();
const BUSINESS_PERIOD_END = "2026-10-09T12:00:00.000Z";
let canUpdateOrg = true;

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => {
      if (resource === "org" && action === "update") {
        return canUpdateOrg;
      }
      return true;
    },
    isLoading: false,
  }),
}));

describe("OrganizationSettingsBillingPage plans", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  beforeEach(() => {
    canUpdateOrg = true;
  });

  it("shows SuperPlane grant after top-up when remaining grant is greater than zero", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: {
            ...BUSINESS_ORGANIZATION_BILLING,
            remainingCreditCents: "115278",
            includedRemainingCents: "0",
            purchasedRemainingCents: "104278",
            welcomeRemainingCents: "0",
            adminRemainingCents: "11000",
          },
          organizationWorkspaceUsage: {
            ...PURCHASED_CREDIT_USAGE_REPORT,
            remainingCreditCents: "115278",
            grantTotalCents: "115278",
            purchasedCreditCents: "104278",
            hostedBilledCents: "0",
            billingEnabled: true,
            hasBillingCustomer: true,
          },
          organizationCreditGrants: [
            {
              id: "grant-admin",
              kind: "admin",
              amountCents: "11000",
              note: "Support grant",
              actorName: "Ada",
              createdAt: "2026-08-10T09:00:00.000Z",
            },
            {
              id: "grant-topup",
              kind: "topup",
              amountCents: "104278",
              polarOrderId: "ord_topup",
              createdAt: "2026-08-18T10:00:00.000Z",
            },
          ],
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).getByTestId("billing-credit-remaining-total")).toHaveTextContent("$1152.78");
    expect(within(balance).getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$0.00 remaining");
    expect(within(balance).getByTestId("billing-credit-included-remaining")).toHaveTextContent("$0.00 remaining");
    expect(within(balance).getByTestId("billing-credit-topup")).toHaveTextContent("Spend third");
    expect(within(balance).getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$1042.78 remaining");
    expect(within(balance).getByTestId("billing-credit-grant")).toHaveTextContent("SuperPlane grant");
    expect(within(balance).getByTestId("billing-credit-grant")).toHaveTextContent("Spend last");
    expect(within(balance).getByTestId("billing-credit-grant-remaining")).toHaveTextContent("$110.00 remaining");
    expect(balance).toHaveTextContent(BILLING_SPEND_ORDER_WITH_GRANT_COPY);
  }, 10000);

  it("does not show SuperPlane grant when remaining grant is zero", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: {
            ...BUSINESS_ORGANIZATION_BILLING,
            remainingCreditCents: "14124",
            adminRemainingCents: "0",
          },
          organizationWorkspaceUsage: {
            ...PURCHASED_CREDIT_USAGE_REPORT,
            billingEnabled: true,
            hasBillingCustomer: true,
          },
          organizationCreditGrants: MIXED_CREDIT_GRANTS,
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).queryByTestId("billing-credit-grant")).not.toBeInTheDocument();
    expect(within(balance).getByTestId("billing-credit-topup")).toHaveTextContent("Spend last");
    expect(balance).toHaveTextContent(BILLING_SPEND_ORDER_COPY);
    expect(balance).not.toHaveTextContent("then SuperPlane grant");
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("SuperPlane grant");
  }, 10000);

  it("shows trial usage after a refund while the trial window is still open", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: RESTORED_TRIAL_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...DEFAULT_FACTORY_USAGE,
            remainingCreditCents: "5000",
            hostedBilledCents: "0",
            billingEnabled: true,
            hasBillingCustomer: true,
          },
        }}
      />,
    );

    expect(await screen.findByTestId("billing-credit-trial")).toHaveTextContent("Trial credit");
    expect(screen.getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$50.00 remaining");
    expect(screen.getByTestId("billing-credit-trial")).toHaveTextContent(`Expires on ${WELCOME_EXPIRY_LABEL}`);
    expect(screen.getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$0.00 remaining");
    expect(screen.getByTestId("billing-credit-included-remaining")).toHaveTextContent("$0.00 remaining");
    expect(await screen.findByRole("button", { name: "Upgrade to Business" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Cancel Business" })).not.toBeInTheDocument();

    const balance = screen.getByTestId("billing-credit-balance");
    expect(balance).toHaveTextContent("Trial");
    expect(balance).toHaveTextContent("This is trial usage for machines and managed models.");
    expect(balance).not.toHaveTextContent("Hosted runs cannot start.");
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
  }, 10000);

  it("cancels Polar Business at period end from the Plans card", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: { ...BUSINESS_ORGANIZATION_BILLING },
        }}
      />,
    );

    await user.click(await screen.findByRole("button", { name: "Cancel Business" }));
    expect(screen.getByTestId("billing-cancel-subscription-dialog")).toHaveTextContent(
      `Business stays active until ${new Date(BUSINESS_PERIOD_END).toLocaleDateString()}. SuperPlane will not renew after that date.`,
    );
    await user.click(screen.getByTestId("billing-cancel-subscription-confirm"));
    expect(await screen.findByTestId("billing-subscription-ends")).toHaveTextContent(
      billingSubscriptionEndsCopy(BUSINESS_PERIOD_END) ?? "",
    );
    expect(screen.getByTestId("billing-current-plan")).toHaveTextContent("Current plan");
    expect(screen.queryByRole("button", { name: "Cancel Business" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Keep Business" })).toBeEnabled();
  }, 10000);

  it("keeps Polar Business when the owner reverses cancel", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: { ...ENDING_ORGANIZATION_BILLING },
        }}
      />,
    );

    expect(await screen.findByTestId("billing-subscription-ends")).toHaveTextContent(
      billingSubscriptionEndsCopy(BUSINESS_PERIOD_END) ?? "",
    );
    await user.click(screen.getByRole("button", { name: "Keep Business" }));
    expect(await screen.findByRole("button", { name: "Cancel Business" })).toBeEnabled();
    expect(screen.queryByTestId("billing-subscription-ends")).not.toBeInTheDocument();
  }, 10000);

  it("hides cancel for an admin Business plan", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: ADMIN_ORGANIZATION_BILLING,
        }}
      />,
    );

    expect(await screen.findByTestId("billing-current-plan")).toHaveTextContent("Current plan");
    expect(screen.queryByRole("button", { name: "Cancel Business" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("billing-subscription-ends")).not.toBeInTheDocument();
  }, 10000);

  it("shows the end date to members and hides cancel", async () => {
    canUpdateOrg = false;
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: ENDING_ORGANIZATION_BILLING,
        }}
      />,
    );

    expect(await screen.findByTestId("billing-subscription-ends")).toHaveTextContent(
      billingSubscriptionEndsCopy(BUSINESS_PERIOD_END) ?? "",
    );
    expect(screen.getByTestId("billing-current-plan")).toHaveTextContent("Current plan");
    expect(screen.queryByRole("button", { name: "Cancel Business" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Keep Business" })).not.toBeInTheDocument();
  }, 10000);
});
