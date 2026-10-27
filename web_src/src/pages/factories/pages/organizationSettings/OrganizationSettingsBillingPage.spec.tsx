import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { DEFAULT_CREDIT_GRANTS, MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  BUSINESS_ORGANIZATION_BILLING,
  DEFAULT_FACTORY_USAGE,
  EXPIRED_TRIAL_ORGANIZATION_BILLING,
  EXPIRED_WELCOME_USAGE_REPORT,
  LAPSED_ORGANIZATION_BILLING,
  RESTORED_TRIAL_ORGANIZATION_BILLING,
  LAPSED_TOPUP_USAGE_REPORT,
  PURCHASED_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
  ACTIVE_TRIAL_ENDS_AT,
  ACTIVE_BILLING_PERIOD_ENDS_AT,
} from "../../__fixtures__/usageReportFixtures";

const WELCOME_EXPIRY_LABEL = new Date(ACTIVE_TRIAL_ENDS_AT).toLocaleDateString();
let canUpdateOrg = true;

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => {
    const canAct = (resource: string, action: string) => resource !== "org" || action !== "update" || canUpdateOrg;
    return { canAct, hasPermission: canAct, isLoading: false };
  },
}));

describe("OrganizationSettingsBillingPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  beforeEach(() => {
    canUpdateOrg = true;
  });

  it("shows signup credit and history without a billing rollout flag", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationCreditGrants: MIXED_CREDIT_GRANTS,
        }}
      />,
    );

    expect(await screen.findByRole("heading", { name: "Billing" })).toBeInTheDocument();
    expect(screen.queryByTestId("workspace-page-header-subtitle")).not.toBeInTheDocument();

    const plans = await screen.findByTestId("billing-plans");
    expect(screen.getAllByTestId("billing-plans")).toHaveLength(1);
    expect(within(plans).queryByTestId("billing-plan-usage")).not.toBeInTheDocument();
    expect(within(plans).queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel Business" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Top up" })).not.toBeInTheDocument();

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).queryByRole("button", { name: "Subscribe" })).not.toBeInTheDocument();
    expect(balance).toHaveTextContent(
      "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.",
    );
    expect(within(balance).getByTestId("billing-credit-trial")).toHaveTextContent("Signup credit");
    expect(within(balance).getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$41.24");
    expect(within(balance).getByTestId("billing-credit-trial")).toHaveTextContent(`Expires on ${WELCOME_EXPIRY_LABEL}`);
    expect(within(balance).queryByTestId("billing-credit-included")).not.toBeInTheDocument();
    expect(within(balance).queryByTestId("billing-credit-topup")).not.toBeInTheDocument();
    expect(balance).not.toHaveTextContent("Subscribe to Business");
    expect(balance).not.toHaveTextContent("SuperPlane grant");
    expect(balance).not.toHaveTextContent("Purchased hosted credit");
    expect(balance).not.toHaveTextContent("Hosted billed spend");
    expect(screen.queryByRole("link", { name: "View spending" })).not.toBeInTheDocument();

    const history = screen.getByTestId("billing-credit-history");
    expect(within(history).getByText("Signup credit")).toBeInTheDocument();
    expect(within(history).getByText("SuperPlane grant")).toBeInTheDocument();
    expect(within(history).getByText("Top-up")).toBeInTheDocument();
    expect(within(history).getByText("Refund")).toBeInTheDocument();
    expect(within(history).getByText("+$25.00")).toBeInTheDocument();
    expect(within(history).getByText("-$5.00")).toBeInTheDocument();
    expect(within(history).getByText("Support grant · Granted by Ada")).toBeInTheDocument();
    expect(within(history).getByText(/Expires on|Expired on/)).toBeInTheDocument();
  }, 10000);

  it("shows an empty history when the organization has no grants", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationCreditGrants: [],
        }}
      />,
    );

    expect(await screen.findByTestId("billing-credit-history")).toHaveTextContent("No credit grants yet.");
  }, 10000);

  it("shows an empty balance and hides exhausted credit buckets", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: {
            ...DEFAULT_FACTORY_USAGE,
            remainingCreditCents: "0",
            hostedBilledCents: "5000",
            remainingCreditWarning: true,
            billingEnabled: true,
            hasBillingCustomer: false,
          },
          organizationBilling: {
            ...defaultFactoriesFixture.organizationBilling,
            remainingCreditCents: "0",
            welcomeRemainingCents: "0",
            includedRemainingCents: "0",
            purchasedRemainingCents: "0",
          },
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).queryByTestId("billing-credit-trial")).not.toBeInTheDocument();
    expect(balance).toHaveTextContent("Your hosted credit balance is empty.");
    expect(within(balance).getByTestId("billing-credit-remaining-total")).toHaveTextContent("$0.00");
    expect(within(balance).queryByTestId("billing-credit-included")).not.toBeInTheDocument();
    expect(within(balance).queryByTestId("billing-credit-topup")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Top up" })).not.toBeInTheDocument();
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("Signup credit");
  }, 10000);

  it("explains unused welcome credit after it expires", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: [STORYBOOK_HOSTED_CREDIT_PRODUCTS[0]],
          organizationBilling: EXPIRED_TRIAL_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...EXPIRED_WELCOME_USAGE_REPORT,
            hasBillingCustomer: false,
          },
          organizationCreditGrants: [
            {
              ...DEFAULT_CREDIT_GRANTS[0],
              expiresAt: "2026-08-15T12:00:00.000Z",
            },
          ],
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(balance).toHaveTextContent("Your hosted credit balance is empty.");
    expect(within(balance).queryByTestId("billing-credit-trial")).not.toBeInTheDocument();
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("Signup credit");
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("Expired on");
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
  }, 10000);

  it("updates the plan after Polar sync finds a Business subscription", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing?subscribed=1`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          billingAfterSync: BUSINESS_ORGANIZATION_BILLING,
        }}
      />,
    );

    expect(await screen.findByRole("heading", { name: "Billing" })).toBeInTheDocument();
    expect(await screen.findByTestId("billing-current-plan", {}, { timeout: 5000 })).toHaveTextContent("Current plan");
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
  }, 10000);

  it("opens Polar checkout from Top up when billing already has a customer", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...PURCHASED_CREDIT_USAGE_REPORT,
            billingEnabled: true,
            hasBillingCustomer: true,
            invoices: [
              {
                id: "ord_1",
                createdAt: "2026-09-01T12:00:00Z",
                amountCents: "2500",
                status: "paid",
                productName: "Hosted credit 25",
              },
            ],
          },
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    await waitFor(() => {
      expect(within(balance).getByRole("button", { name: "Top up" })).toBeEnabled();
    });
    expect(screen.getByTestId("billing-current-plan")).toHaveTextContent("Current plan");
    expect(screen.getByTestId("billing-credit-included")).toHaveTextContent("Included usage");
    expect(screen.getByTestId("billing-credit-included-remaining")).toHaveTextContent("$50.00");
    expect(screen.getByTestId("billing-credit-included")).toHaveTextContent(
      `Resets on ${new Date(ACTIVE_BILLING_PERIOD_ENDS_AT).toLocaleDateString()}`,
    );
    expect(screen.getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$41.24");
    expect(screen.getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$50.00");
    expect(within(balance).getByTestId("billing-credit-remaining-total")).toHaveTextContent("$141.24");
    expect(balance).toHaveTextContent("Credits are used in the order shown below.");
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel Business" })).toBeEnabled();
    expect(screen.queryByTestId("billing-subscription-ends")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "View spending" })).not.toBeInTheDocument();
    expect(balance).not.toHaveTextContent("welcome credit");

    await user.click(within(balance).getByRole("button", { name: "Top up" }));
    const dialog = await screen.findByRole("dialog", { name: "Add hosted credit" });
    expect(within(dialog).getByRole("button", { name: "$50" })).toBeEnabled();
    expect(within(dialog).getByRole("button", { name: "$100" })).toBeEnabled();
    expect(within(dialog).getByRole("button", { name: "$500" })).toBeEnabled();
    expect(within(dialog).getByRole("button", { name: "Custom" })).toBeDisabled();

    await user.click(within(dialog).getByRole("button", { name: "$50" }));
    const confirm = within(dialog).getByRole("button", { name: "Continue to checkout" });
    await waitFor(() => expect(confirm).toBeEnabled());
    await user.click(confirm);
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook"));
    vi.unstubAllGlobals();
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    const invoices = await screen.findByTestId("billing-polar-invoices");
    expect(within(invoices).getByText("Recent paid invoices for this organization.")).toBeInTheDocument();
    expect(within(invoices).getByText("Hosted credit 25")).toBeInTheDocument();
    expect(within(invoices).getByText("$25.00")).toBeInTheDocument();
    expect(within(invoices).getByText("Paid")).toBeInTheDocument();
    expect(within(invoices).getByText("Manage invoices")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Credit history" }));
    expect(await screen.findByTestId("billing-credit-history")).toBeInTheDocument();
    expect(screen.queryByTestId("billing-polar-invoices")).not.toBeInTheDocument();
  }, 10000);

  it("shows an empty invoices card when Polar has a customer and no invoices", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...PURCHASED_CREDIT_USAGE_REPORT,
            billingEnabled: true,
            hasBillingCustomer: true,
            invoices: [],
          },
        }}
      />,
    );

    const invoices = await screen.findByTestId("billing-polar-invoices");
    expect(within(invoices).getByText("No invoices yet")).toBeInTheDocument();
    expect(within(invoices).getByText("Paid invoices appear here after checkout.")).toBeInTheDocument();
    expect(within(invoices).getByText("Manage invoices")).toBeInTheDocument();
  }, 10000);

  it("preserves top-up credit after Business lapses without showing exhausted included usage", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: LAPSED_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: LAPSED_TOPUP_USAGE_REPORT,
          organizationCreditGrants: [
            {
              id: "grant-included",
              kind: "included",
              amountCents: "5000",
              createdAt: "2026-09-10T12:00:00.000Z",
              expiresAt: "2026-08-15T12:00:00.000Z",
            },
            {
              id: "grant-topup",
              kind: "topup",
              amountCents: "5000",
              polarOrderId: "ord_topup_50",
              createdAt: "2026-09-10T12:00:00.000Z",
              expiresAt: "2027-09-10T12:00:00.000Z",
            },
          ],
        }}
      />,
    );

    expect(await screen.findByTestId("billing-credit-topup-remaining")).toHaveTextContent("$50.00");
    expect(screen.queryByTestId("billing-credit-included")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel Business" })).not.toBeInTheDocument();
    expect(screen.getByTestId("billing-current-plan")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Top up" })).not.toBeInTheDocument();

    await userEvent.setup().click(screen.getByRole("tab", { name: "Credit history" }));
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("Expired on");
  }, 10000);

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
    expect(within(balance).queryByTestId("billing-credit-trial")).not.toBeInTheDocument();
    expect(within(balance).queryByTestId("billing-credit-included")).not.toBeInTheDocument();
    expect(within(balance).getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$1042.78");
    expect(within(balance).getByTestId("billing-credit-grant")).toHaveTextContent("SuperPlane grant");
    expect(within(balance).getByTestId("billing-credit-grant-remaining")).toHaveTextContent("$110.00");
    const rows = within(balance).getAllByRole("listitem");
    expect(rows[0]).toHaveTextContent("Top-up credit");
    expect(rows[1]).toHaveTextContent("SuperPlane grant");
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
    await userEvent.setup().click(screen.getByRole("tab", { name: "Credit history" }));
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("SuperPlane grant");
  }, 10000);

  it("preserves restored signup credit and its expiry after a refund", async () => {
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

    expect(await screen.findByTestId("billing-credit-trial")).toHaveTextContent("Signup credit");
    expect(screen.getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$50.00");
    expect(screen.getByTestId("billing-credit-trial")).toHaveTextContent(`Expires on ${WELCOME_EXPIRY_LABEL}`);
    expect(screen.queryByTestId("billing-credit-topup")).not.toBeInTheDocument();
    expect(screen.queryByTestId("billing-credit-included")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();

    const balance = screen.getByTestId("billing-credit-balance");
    expect(balance).not.toHaveTextContent("Subscribe to Business");
    expect(balance).not.toHaveTextContent("Hosted runs cannot start.");
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
  }, 10000);
});
