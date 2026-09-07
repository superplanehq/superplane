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

describe("WorkOrdersPage broken integrations banner", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("hides the banner when every integration is ready", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
        orgIntegrations={[
          { metadata: { id: "gh-1", name: "github-main", integrationName: "github" }, status: { state: "ready" } },
        ]}
      />,
    );

    expect(await screen.findByTestId("work-orders-header", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("broken-integrations-banner")).not.toBeInTheDocument();
  }, 10000);

  it("names the integration and the repair step when a connection errors", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
        orgIntegrations={[
          {
            metadata: { id: "gh-1", name: "github-main", integrationName: "github" },
            status: { state: "error", stateDescription: "App was uninstalled" },
          },
        ]}
      />,
    );

    const banner = await screen.findByTestId("broken-integrations-banner", {}, { timeout: 8000 });
    expect(banner).toHaveTextContent("1 integration needs attention");
    expect(banner).toHaveTextContent("App was uninstalled");
    expect(screen.getByRole("link", { name: "Reinstall app" })).toHaveAttribute(
      "href",
      `${factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "integrations")}/gh-1`,
    );
  }, 10000);

  it("shows the banner on the production Tasks page when a connection errors", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
        orgIntegrations={[
          {
            metadata: { id: "gh-1", name: "github-main", integrationName: "github" },
            status: { state: "error", stateDescription: "App was uninstalled" },
          },
        ]}
        pageOverrides={{ workOrders: WorkOrdersPage }}
      />,
    );

    expect(await screen.findByTestId("broken-integrations-banner", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);
});
