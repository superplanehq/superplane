import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageIds";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  DEFAULT_FACTORY_USAGE,
  SPENT_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";

describe("OrganizationSettingsBillingPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows remaining credit, grant history, and a disabled add-credit control when Polar is off", async () => {
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
    expect(balance).toHaveTextContent("$41.24");
    expect(balance).toHaveTextContent("Organization wallet");
    expect(balance).toHaveTextContent("SuperPlane grant");
    expect(balance).toHaveTextContent("$50.00");
    expect(balance).toHaveTextContent("Welcome and support grants");
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

  it("shows an empty-credit warning and keeps add-credit disabled when Polar is off", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        }}
      />,
    );

    expect(await screen.findByText("Hosted credit is empty. SuperPlane-hosted runs cannot start.")).toBeInTheDocument();
    // Polar is marked on in the spent fixture, but no packs are configured yet.
    expect(screen.getByRole("button", { name: "Add hosted credit" })).toBeDisabled();
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

  it("lists Polar packs in a menu when more than one pack is available", async () => {
    const user = userEvent.setup();

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
          organizationWorkspaceUsage: {
            ...DEFAULT_FACTORY_USAGE,
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
      expect(screen.getByRole("button", { name: "Add hosted credit" })).toBeEnabled();
    });
    await user.click(screen.getByRole("button", { name: "Add hosted credit" }));
    expect(await screen.findByRole("menuitem", { name: "$25.00" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "$100.00" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "$500.00" })).toBeInTheDocument();
    await user.keyboard("{Escape}");

    const invoices = await screen.findByTestId("billing-polar-invoices");
    expect(within(invoices).getByText("Hosted credit 25")).toBeInTheDocument();
    expect(within(invoices).getByText("$25.00")).toBeInTheDocument();
    expect(within(invoices).getByText("Paid")).toBeInTheDocument();
    expect(within(invoices).getByText("Manage invoices")).toBeInTheDocument();
  }, 10000);
});
