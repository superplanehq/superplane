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
  PURCHASED_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";

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
    const usage = within(plans).getByTestId("billing-plan-usage");
    expect(usage).toHaveTextContent("Your trial usage");
    expect(usage).toHaveTextContent("$41.24 remaining");
    expect(usage).toHaveTextContent(`Ends ${WELCOME_EXPIRY_LABEL}`);
    expect(within(plans).getByTestId("billing-plan-usage-bar")).toHaveAttribute("aria-valuenow", "18");
    expect(within(plans).getByTestId("billing-plan-business")).toHaveTextContent("$199");
    expect(within(plans).getByRole("button", { name: "Upgrade to Business" })).toBeEnabled();
    expect(screen.getByRole("link", { name: "Talk to us" })).toHaveAttribute("href", "https://superplane.com/pricing/");
    expect(screen.queryByRole("button", { name: "Buy more" })).not.toBeInTheDocument();

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).queryByRole("button", { name: "Subscribe" })).not.toBeInTheDocument();
    expect(balance).toHaveTextContent(
      "Hosted credit pays SuperPlane-hosted machines and managed models for this organization.",
    );
    expect(balance).toHaveTextContent("Remaining hosted credit");
    expect(balance).toHaveTextContent("Trial");
    expect(within(balance).getByTestId("billing-remaining-credit")).toHaveTextContent("$41.24");
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
        }}
      />,
    );

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(within(balance).getByTestId("billing-remaining-credit")).toHaveTextContent("$0.00");
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
    expect(within(balance).getByTestId("billing-remaining-credit")).toHaveTextContent("$0.00");
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

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Buy more" })).toBeEnabled();
    });
    await user.click(screen.getByRole("button", { name: "Buy more" }));
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

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Buy more" })).toBeEnabled();
    });
    expect(screen.getByTestId("billing-current-plan")).toHaveTextContent("Current plan");
    expect(screen.getByTestId("billing-plan-usage")).toHaveTextContent("Your included usage");
    expect(screen.getByTestId("billing-plan-usage")).toHaveTextContent("$50.00 remaining");
    expect(screen.getByTestId("billing-plan-usage")).toHaveTextContent(
      `Resets ${new Date("2026-10-09T12:00:00.000Z").toLocaleDateString()}`,
    );
    expect(screen.queryByRole("button", { name: "Upgrade to Business" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Talk to us" })).toHaveAttribute("href", "https://superplane.com/pricing/");
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("billing-credit-balance")).getByTestId("billing-remaining-credit"),
    ).toHaveTextContent("$141.24");
    expect(screen.queryByRole("link", { name: "View spending" })).not.toBeInTheDocument();
    expect(screen.getByTestId("billing-credit-balance")).not.toHaveTextContent("welcome credit");

    await user.click(screen.getByRole("button", { name: "Buy more" }));
    expect(await screen.findByRole("menuitem", { name: "$50" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "$100" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "$500" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "Custom" })).toHaveAttribute("aria-disabled", "true");

    await user.click(screen.getByRole("menuitem", { name: "$50" }));
    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook");
    vi.unstubAllGlobals();

    const invoices = await screen.findByTestId("billing-polar-invoices");
    expect(within(invoices).getByText("Hosted credit 25")).toBeInTheDocument();
    expect(within(invoices).getByText("$25.00")).toBeInTheDocument();
    expect(within(invoices).getByText("Paid")).toBeInTheDocument();
    expect(within(invoices).getByText("Manage invoices")).toBeInTheDocument();
  }, 10000);
});
