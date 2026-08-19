import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import { VelocityPrototypePage } from "./VelocityPrototypePage";

// Regression test for the reported issue: the Factories/Pages/Velocity ->
// Default story (this exact FactoriesHarness + VelocityPrototypePage
// combination) must show the sub-hour-aware duration labels, not just an
// isolated component story.
describe("Velocity prototype page (Factories/Pages/Velocity -> Default)", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("reads the default 7-day work-order time section in minutes instead of a repeated 0h", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/velocity`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={{ velocity: VelocityPrototypePage }}
      />,
    );

    const flow = await screen.findByTestId("velocity-work-order-flow", {}, { timeout: 8000 });
    expect(flow).toHaveTextContent("27m");
    expect(flow).toHaveTextContent("12m");
    expect(flow).toHaveTextContent("15m");
    expect(flow).not.toHaveTextContent("0h");
  }, 10000);
});
