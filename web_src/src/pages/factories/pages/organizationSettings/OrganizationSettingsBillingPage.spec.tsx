import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { DEFAULT_CREDIT_GRANTS, MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageIds";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  DEFAULT_FACTORY_USAGE,
  EXPIRED_WELCOME_USAGE_REPORT,
  PURCHASED_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";

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
    expect(screen.getByRole("button", { name: "Add hosted credit" })).toBeDisabled();

    const balance = await screen.findByTestId("billing-credit-balance");
    expect(balance).toHaveTextContent("Remaining hosted credit");
    expect(balance).toHaveTextContent("Trial");
    expect(within(balance).getByTestId("billing-remaining-credit")).toHaveTextContent("$41.24");
    expect(balance).toHaveTextContent("This remaining balance is welcome credit.");
    expect(balance).toHaveTextContent(`Unused credit expires on ${WELCOME_EXPIRY_LABEL}.`);
    expect(balance).toHaveTextContent("Purchase hosted credit to keep SuperPlane-hosted runs after the trial.");
    expect(balance).not.toHaveTextContent("SuperPlane grant");
    expect(balance).not.toHaveTextContent("Purchased hosted credit");
    expect(balance).not.toHaveTextContent("Hosted billed spend");
    expect(screen.queryByTestId("billing-credit-packs")).not.toBeInTheDocument();
    expect(await screen.findByTestId("factories-sidebar-plan-label")).toHaveTextContent("Trial");
    expect(screen.getByRole("link", { name: "View spending" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "spending"),
    );

    const history = screen.getByTestId("billing-credit-history");
    expect(within(history).getByText("Welcome credit")).toBeInTheDocument();
    expect(within(history).getByText("SuperPlane grant")).toBeInTheDocument();
    expect(within(history).getByText("Purchased")).toBeInTheDocument();
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

  it("shows $0 and a purchase prompt when Polar has no customer and credit is empty", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: [STORYBOOK_HOSTED_CREDIT_PRODUCTS[1]],
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
    expect(balance).toHaveTextContent("Hosted credit is empty. Click Add hosted credit to purchase more.");
    expect(screen.queryByTestId("billing-credit-packs")).not.toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Add hosted credit" })).toBeEnabled();
  }, 10000);

  it("explains unused welcome credit after it expires", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: [STORYBOOK_HOSTED_CREDIT_PRODUCTS[1]],
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
      "Welcome credit expired. SuperPlane-hosted runs cannot start. Click Add hosted credit to purchase more.",
    );
    expect(within(balance).getByTestId("billing-remaining-credit")).toHaveTextContent("$0.00");
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
          hostedCreditProducts: [STORYBOOK_HOSTED_CREDIT_PRODUCTS[1]],
          organizationWorkspaceUsage: {
            ...DEFAULT_FACTORY_USAGE,
            billingEnabled: true,
            hasBillingCustomer: false,
          },
        }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Add hosted credit" })).toBeEnabled();
    });
    await user.click(screen.getByRole("button", { name: "Add hosted credit" }));

    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook");
    vi.unstubAllGlobals();
  }, 10000);

  it("stacks Polar pack buttons when billing already has a customer", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
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

    expect(await screen.findByTestId("billing-credit-packs")).toBeInTheDocument();
    expect(screen.queryByTestId("factories-sidebar-plan-label")).not.toBeInTheDocument();
    expect(
      within(screen.getByTestId("billing-credit-balance")).getByTestId("billing-remaining-credit"),
    ).toHaveTextContent("$141.24");
    expect(screen.queryByRole("button", { name: "Add hosted credit" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add $25.00" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add $100.00" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add $500.00" })).toBeInTheDocument();
    expect(screen.getByTestId("billing-credit-balance")).not.toHaveTextContent("welcome credit");

    await user.click(screen.getByRole("button", { name: "Add $25.00" }));
    expect(assign).toHaveBeenCalledWith("https://buy.polar.sh/polar_c_storybook");
    vi.unstubAllGlobals();

    const invoices = await screen.findByTestId("billing-polar-invoices");
    expect(within(invoices).getByText("Hosted credit 25")).toBeInTheDocument();
    expect(within(invoices).getByText("$25.00")).toBeInTheDocument();
    expect(within(invoices).getByText("Paid")).toBeInTheDocument();
    expect(within(invoices).getByText("Manage invoices")).toBeInTheDocument();
  }, 10000);
});
