import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, defaultFactoriesFixture } from "./factoryPageResponses";
import { FactoriesHarness } from "./FactoriesHarness";
import { CONNECTED_SETUP_INTEGRATIONS, SETUP_ANSWERS, factoriesFixtureWithSetupAnswers } from "./setupStoryFixtures";

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
});

describe("FactoriesHarness workspace setup", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("mounts the same setup page as the app, without workspace chrome", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup`}
        factoriesFixture={defaultFactoriesFixture}
        onboardingSeed={{ pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Refunds Factory" } }}
      />,
    );

    expect(await screen.findByTestId("workspace-setup", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-cancel")).toBeInTheDocument();
    expect(screen.queryByTestId("factories-sidebar")).not.toBeInTheDocument();
  }, 10000);

  it("continues from a seeded GitHub connection to the repository list", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup`}
        factoriesFixture={defaultFactoriesFixture}
        onboardingSeed={{ pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Refunds Factory" } }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    await user.click(await screen.findByRole("button", { name: /acme-github/ }, { timeout: 8000 }));

    expect(await screen.findByRole("option", { name: /acme\/api/ }, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);

  it("opens the step from the URL with the saved answers restored", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=issues`}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository)}
        onboardingSeed={{ pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Refunds Factory" } }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    expect(
      await screen.findByRole("button", { name: /Change backlog repository/ }, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 15000);
});
