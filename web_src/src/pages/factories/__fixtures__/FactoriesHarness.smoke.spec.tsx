import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { PRIMARY_FACTORY_KEY, defaultFactoriesFixture } from "./factoryPageResponses";
import { FactoriesHarness } from "./FactoriesHarness";

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
