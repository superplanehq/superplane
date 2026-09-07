import { render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageIds";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { SPENT_CREDIT_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";

describe("OrganizationSettingsBillingPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows remaining credit, grant history, and a disabled add-credit control", async () => {
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

    const balance = screen.getByTestId("billing-credit-balance");
    expect(balance).toHaveTextContent("Remaining hosted credit");
    expect(balance).toHaveTextContent("$41.24");
    expect(balance).toHaveTextContent("SuperPlane grant");
    expect(balance).toHaveTextContent("$50.00");
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

  it("shows an empty-credit warning and keeps add-credit disabled", async () => {
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
    expect(screen.getByRole("button", { name: "Add hosted credit" })).toBeDisabled();
  }, 10000);
});
