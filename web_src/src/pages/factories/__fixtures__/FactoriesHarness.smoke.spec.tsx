import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY, defaultFactoriesFixture } from "./factoryPageResponses";
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
});
