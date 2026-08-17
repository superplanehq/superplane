import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "./FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY, REFUND_FACTORY_LINES } from "./factoryPageResponses";

async function expectFactoryAppCanvasStayed() {
  await waitFor(
    () => {
      const page = screen.getByTestId("factory-app-canvas-page");
      expect(within(page).queryByText("Loading…")).not.toBeInTheDocument();
    },
    { timeout: 8000 },
  );
  expect(screen.getByTestId("factory-app-canvas-page")).toBeInTheDocument();
  expect(screen.queryByTestId("overview-work-orders-card")).not.toBeInTheDocument();
}

describe("FactoriesHarness work orders", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows missions when Work Orders opens from the factory sidebar", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("mission-views", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);
});

describe("FactoriesHarness factory canvas navigation", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens the factory canvas when editing an automation", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        enableOnboarding={false}
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/automations`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    await screen.findByTestId("automations-list-page", {}, { timeout: 8000 });
    await screen.findByTestId("automations-list", {}, { timeout: 8000 });
    await user.click(screen.getAllByTestId("automations-card-menu")[0]!);
    await user.click(await screen.findByTestId("automations-card-edit"));

    await expectFactoryAppCanvasStayed();
  }, 15000);

  it("opens the factory canvas when clicking a line run", async () => {
    const user = userEvent.setup();
    const line = REFUND_FACTORY_LINES[0];
    render(
      <FactoriesHarness
        enableOnboarding={false}
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${line.id}`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    await screen.findByTestId("lines-phase-board", {}, { timeout: 8000 });
    await user.click(screen.getAllByTestId(/^lines-phase-run-/)[0]!);

    await expectFactoryAppCanvasStayed();
  }, 15000);
});
