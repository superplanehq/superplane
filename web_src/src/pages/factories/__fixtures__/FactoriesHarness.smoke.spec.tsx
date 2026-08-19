import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  FACTORIES_ORGANIZATION_ID,
  defaultFactoriesFixture,
} from "./factoryPageResponses";
import { factorySettingsPath } from "../lib/factoryPagePaths";
import { FactoriesHarness } from "./FactoriesHarness";

describe("FactoriesHarness work orders", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows the Work Orders board when Work Orders opens from the factory sidebar", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("work-orders-header", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("mission-views")).not.toBeInTheDocument();
  }, 10000);

  it("serves a factory-owned canvas so app clicks do not bounce to Overview", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/overview`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("overview-work-orders-card", {}, { timeout: 8000 })).toBeInTheDocument();

    const response = await fetch("http://localhost/api/v1/canvases/app-refund-implementer");
    const body = (await response.json()) as { canvas?: { metadata?: { factoryId?: string } } };
    expect(body.canvas?.metadata?.factoryId).toBe(PRIMARY_FACTORY_ID);
  }, 10000);

  it("lets the signed-in user open workspace settings from the sidebar cog", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/overview`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const settingsLink = await screen.findByTestId("factories-workspace-settings-link", {}, { timeout: 8000 });
    await waitFor(() => {
      expect(settingsLink).toHaveAttribute("href", factorySettingsPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY));
    });
    expect(settingsLink).not.toHaveClass("pointer-events-none");
  }, 10000);

  it("lets the signed-in user open organization settings from the user menu", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/overview`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const trigger = await screen.findByTestId("factories-sidebar-user-menu-trigger", {}, { timeout: 8000 });
    expect(screen.queryByTestId("factories-sidebar-organization-settings-link")).not.toBeInTheDocument();
    await user.click(trigger);
    const orgCog = await screen.findByTestId("factories-sidebar-organization-settings-link");
    await user.click(orgCog);
    expect(await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);
});
