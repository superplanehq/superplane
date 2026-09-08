import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_KEY,
} from "../__fixtures__/factoryPageResponses";
import {
  EXPIRED_WELCOME_USAGE_REPORT,
  PURCHASED_CREDIT_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
} from "../__fixtures__/usageReportFixtures";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { WorkOrdersPage } from "./WorkOrdersPage";

describe("WorkOrdersPage hosted credit banner", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows the trial banner when welcome credit remains", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const banner = await screen.findByTestId("hosted-credit-empty-banner", {}, { timeout: 8000 });
    expect(banner).toHaveTextContent("Trial");
    expect(banner).toHaveTextContent("$41.24");
    expect(screen.getByRole("link", { name: "Open billing" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "billing"),
    );
  }, 10000);

  it("hides the trial banner after the organization buys credit", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: PURCHASED_CREDIT_USAGE_REPORT,
        }}
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
    expect(banner).toHaveTextContent("Trial credit is empty");
    expect(banner).toHaveTextContent("SuperPlane-hosted runs cannot start. Open Billing to buy hosted credit.");
    expect(screen.getByRole("link", { name: "Open billing" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "billing"),
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

  it("shows the trial-ended banner when welcome credit expires", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: EXPIRED_WELCOME_USAGE_REPORT,
        }}
      />,
    );

    const banner = await screen.findByTestId("hosted-credit-empty-banner", {}, { timeout: 8000 });
    expect(banner).toHaveTextContent("Trial ended");
    expect(screen.getByRole("link", { name: "Open billing" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, "organization", "billing"),
    );
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
