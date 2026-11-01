import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { client } from "@/api-client/client.gen";
import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";
import { defaultHomePageFixture } from "@/pages/home/__fixtures__/homePageResponses";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { TEAM_ORGANIZATION_BILLING, DEFAULT_FACTORY_USAGE } from "../../__fixtures__/usageReportFixtures";

describe("Suspended organization billing", () => {
  beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));

  it("preserves invoices and the billing portal while blocking configuration changes", async () => {
    render(
      <OrgWorkspaceHarness
        startAt="home"
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`}
        homeFixture={{
          ...defaultHomePageFixture,
          organizationId: defaultFactoriesFixture.organizationId!,
          suspended: true,
          enabledExperimentalFeatures: [FEATURE_FACTORIES],
        }}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationBilling: TEAM_ORGANIZATION_BILLING,
          organizationWorkspaceUsage: {
            ...DEFAULT_FACTORY_USAGE,
            billingEnabled: true,
            hasBillingCustomer: true,
            invoices: [{ id: "invoice-1", productName: "Team renewal", amountCents: "29000", status: "paid" }],
          },
        }}
      />,
    );
    expect(await screen.findByText("Team renewal")).toBeVisible();
    expect(screen.getByRole("button", { name: "Manage invoices" })).toBeEnabled();
    const portal = screen.getByRole("button", { name: "Manage payment method" });
    expect(portal).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Change plan" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Buy seats" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel Team" })).not.toBeInTheDocument();
    const assign = vi.fn();
    const previousAssign = window.location.assign;
    window.location.assign = assign;
    try {
      await userEvent.setup().click(portal);
      await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.sh/portal/storybook"));
      assign.mockClear();
      await userEvent.setup().click(screen.getByRole("button", { name: "Manage invoices" }));
      await waitFor(() => expect(assign).toHaveBeenCalledWith("https://polar.sh/portal/storybook"));
    } finally {
      window.location.assign = previousAssign;
    }
  });
});
