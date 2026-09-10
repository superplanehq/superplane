import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

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
  LAPSED_TOPUP_USAGE_REPORT,
  PURCHASED_CREDIT_USAGE_REPORT,
  RESTORED_TRIAL_ORGANIZATION_BILLING,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";
import { BILLING_SPEND_ORDER_COPY, BILLING_TRIAL_TTL_COPY } from "../../lib/billingCreditBuckets";

const WELCOME_EXPIRY_LABEL = new Date("2026-09-22T12:00:00.000Z").toLocaleDateString();

describe("OrganizationSettingsBillingPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows remaining welcome credit and trial copy when Polar has no customer", async () => {
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
    expect(within(plans).queryByTestId("billing-plan-usage")).not.toBeInTheDocument();
    expect(within(plans).getByTestId("billing-plan-business")).toHaveTextContent("$199");
    expect(within(plans).getByRole("button", { name: "Upgrade to Business" })).toBeEnabled();
    expect(screen.getByRole("link", { name: "Talk to us" })).toHaveAttribute("href", "https://superplane.com/pricing/");
    expect(screen.queryByRole("button", { name: "Buy more" })).not.toBeInTheDocument();

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).queryByRole("button", { name: "Subscribe" })).not.toBeInTheDocument();
    expect(balance).toHaveTextContent(
      "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.",
    );
    expect(balance).toHaveTextContent(BILLING_SPEND_ORDER_COPY);
    expect(within(balance).getByTestId("billing-credit-trial")).toHaveTextContent("Trial credit");
    expect(within(balance).getByTestId("billing-credit-trial")).toHaveTextContent("Spend first");
    expect(within(balance).getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$41.24 remaining");
    expect(within(balance).getByTestId("billing-credit-trial")).toHaveTextContent(
      `Expires on ${WELCOME_EXPIRY_LABEL}. ${BILLING_TRIAL_TTL_COPY}`,
    );
    expect(within(balance).getByTestId("billing-credit-included")).toHaveTextContent("Included usage");
    expect(within(balance).getByTestId("billing-credit-included")).toHaveTextContent("Spend next");
    expect(within(balance).getByTestId("billing-credit-included-remaining")).toHaveTextContent("$0.00 remaining");
    expect(within(balance).getByTestId("billing-credit-included")).toHaveTextContent("Included with Business.");
    expect(within(balance).getByTestId("billing-credit-topup")).toHaveTextContent("Top-up credit");
    expect(within(balance).getByTestId("billing-credit-topup")).toHaveTextContent("Spend last");
    expect(within(balance).getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$0.00 remaining");
    expect(balance).toHaveTextContent("Trial");
    expect(balance).toHaveTextContent("This is trial usage for machines and managed models.");
    expect(balance).toHaveTextContent(`The trial ends on ${WELCOME_EXPIRY_LABEL}.`);
    expect(balance).toHaveTextContent("Subscribe to Business to keep hosted runs.");
    expect(balance).not.toHaveTextContent("SuperPlane grant");
    expect(balance).not.toHaveTextContent("Purchased hosted credit");
    expect(balance).not.toHaveTextContent("Hosted billed spend");
    expect(screen.queryByRole("link", { name: "View spending" })).not.toBeInTheDocument();

    const history = screen.getByTestId("billing-credit-history");
    expect(within(history).getByText("Trial")).toBeInTheDocument();
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

  it("shows $0 and a subscribe prompt when trial usage is used up", async () => {
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
    expect(within(balance).getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$0.00 remaining");
    expect(balance).toHaveTextContent(
      "Trial credit is used up. Hosted runs cannot start. Subscribe to Business to continue.",
    );
    expect(await screen.findByRole("button", { name: "Upgrade to Business" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Buy more" })).not.toBeInTheDocument();
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
    expect(balance).toHaveTextContent(
      "The trial has ended. Hosted runs cannot start. Subscribe to Business to continue.",
    );
    expect(within(balance).getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$0.00 remaining");
    expect(await screen.findByRole("button", { name: "Upgrade to Business" })).toBeEnabled();
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

  it("opens Polar checkout from Business Subscribe", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    await user.click(await screen.findByRole("button", { name: "Upgrade to Business" }));
    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_business");
    vi.unstubAllGlobals();
  }, 10000);

  it("opens Polar checkout from Subscribe when the checkout flag is off", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: {
            ...defaultFactoriesFixture.organizationBilling,
            subscriptionCheckoutEnabled: false,
          },
        }}
      />,
    );

    const business = await screen.findByTestId("billing-plan-business");
    await user.click(within(business).getByRole("button", { name: "Upgrade to Business" }));
    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_business");
    vi.unstubAllGlobals();
  }, 10000);

  it("opens Polar checkout for a single hosted credit pack", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: [STORYBOOK_HOSTED_CREDIT_PRODUCTS[0]],
          organizationBilling: BUSINESS_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...DEFAULT_FACTORY_USAGE,
            billingEnabled: true,
            hasBillingCustomer: false,
          },
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    await waitFor(() => {
      expect(within(balance).getByRole("button", { name: "Buy more" })).toBeEnabled();
    });
    await user.click(within(balance).getByRole("button", { name: "Buy more" }));
    await user.click(await screen.findByRole("menuitem", { name: "$50" }));

    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook");
    vi.unstubAllGlobals();
  }, 10000);

  it("opens Polar checkout from Buy more when billing already has a customer", async () => {
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
      expect(within(balance).getByRole("button", { name: "Buy more" })).toBeEnabled();
    });
    expect(screen.getByTestId("billing-current-plan")).toHaveTextContent("Current plan");
    expect(screen.getByTestId("billing-credit-included")).toHaveTextContent("Included usage");
    expect(screen.getByTestId("billing-credit-included-remaining")).toHaveTextContent("$50.00 remaining");
    expect(screen.getByTestId("billing-credit-included")).toHaveTextContent(
      `Resets ${new Date("2026-10-09T12:00:00.000Z").toLocaleDateString()}`,
    );
    expect(screen.getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$0.00 remaining");
    expect(screen.getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$91.24 remaining");
    expect(within(balance).getByTestId("billing-credit-remaining-total")).toHaveTextContent("$141.24");
    expect(balance).toHaveTextContent(BILLING_SPEND_ORDER_COPY);
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Talk to us" })).toHaveAttribute("href", "https://superplane.com/pricing/");
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "View spending" })).not.toBeInTheDocument();
    expect(balance).not.toHaveTextContent("welcome credit");

    await user.click(within(balance).getByRole("button", { name: "Buy more" }));
    expect(await screen.findByRole("menuitem", { name: "$50" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "$100" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "$500" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "Custom" })).toHaveAttribute("aria-disabled", "true");

    await user.click(screen.getByRole("menuitem", { name: "$50" }));
    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook");
    vi.unstubAllGlobals();

    const invoices = await screen.findByTestId("billing-polar-invoices");
    expect(within(invoices).getByText("Recent paid invoices for this organization.")).toBeInTheDocument();
    expect(within(invoices).getByText("Hosted credit 25")).toBeInTheDocument();
    expect(within(invoices).getByText("$25.00")).toBeInTheDocument();
    expect(within(invoices).getByText("Paid")).toBeInTheDocument();
    expect(within(invoices).getByText("Manage invoices")).toBeInTheDocument();
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

  it("asks the organization to subscribe after Business lapses and does not show included usage", async () => {
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
              polarOrderId: "trial-conversion:org:sub",
              createdAt: "2026-09-10T12:00:00.000Z",
              expiresAt: "2027-09-10T12:00:00.000Z",
            },
          ],
        }}
      />,
    );

    expect(await screen.findByTestId("billing-credit-included-remaining")).toHaveTextContent("$0.00 remaining");
    expect(screen.getByTestId("billing-credit-included")).toHaveTextContent("Included with Business.");
    expect(screen.getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$50.00 remaining");
    expect(screen.getByTestId("billing-credit-balance")).toHaveTextContent(BILLING_SPEND_ORDER_COPY);
    expect(await screen.findByRole("button", { name: "Upgrade to Business" })).toBeEnabled();
    expect(screen.queryByTestId("billing-current-plan")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Buy more" })).not.toBeInTheDocument();

    const balance = screen.getByTestId("billing-credit-balance");
    expect(balance).toHaveTextContent("Hosted runs cannot start. Subscribe to Business to continue.");
    expect(screen.getByTestId("billing-credit-history")).toHaveTextContent("Expired on");
  }, 10000);

  it("shows trial usage after a refund while the trial window is still open", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: RESTORED_TRIAL_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: LAPSED_TOPUP_USAGE_REPORT,
        }}
      />,
    );

    expect(await screen.findByTestId("billing-credit-trial")).toHaveTextContent("Trial credit");
    expect(screen.getByTestId("billing-credit-trial-remaining")).toHaveTextContent("$0.00 remaining");
    expect(screen.getByTestId("billing-credit-trial")).toHaveTextContent(`Expires on ${WELCOME_EXPIRY_LABEL}`);
    expect(screen.getByTestId("billing-credit-topup-remaining")).toHaveTextContent("$50.00 remaining");
    expect(screen.getByTestId("billing-credit-included-remaining")).toHaveTextContent("$0.00 remaining");
    expect(await screen.findByRole("button", { name: "Upgrade to Business" })).toBeEnabled();

    const balance = screen.getByTestId("billing-credit-balance");
    expect(balance).toHaveTextContent("Trial");
    expect(balance).toHaveTextContent("This is trial usage for machines and managed models.");
    expect(balance).not.toHaveTextContent("Hosted runs cannot start.");
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
  }, 10000);
});
