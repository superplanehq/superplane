import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";
import { SPENT_CREDIT_USAGE_REPORT } from "../__fixtures__/usageReportFixtures";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { WorkOrdersPage } from "./WorkOrdersPage";

describe("WorkOrdersPage hosted credit banner", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("hides the banner when remaining hosted credit is greater than zero", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("work-orders-header", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-empty-banner")).not.toBeInTheDocument();
  }, 10000);

  it("shows the banner on Tasks when remaining hosted credit is empty", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        }}
      />,
    );

    const banner = await screen.findByTestId("hosted-credit-empty-banner", {}, { timeout: 8000 });
    expect(banner).toHaveTextContent("Hosted credit is empty");
    expect(banner).toHaveTextContent("Add hosted credit to start SuperPlane-hosted runs.");
    expect(screen.getByRole("link", { name: "View spending" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "spending"),
    );
  }, 10000);

  it("shows the banner on the production Tasks page when remaining credit is empty", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        }}
        pageOverrides={{ workOrders: WorkOrdersPage }}
      />,
    );

    expect(await screen.findByTestId("hosted-credit-empty-banner", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);
});
