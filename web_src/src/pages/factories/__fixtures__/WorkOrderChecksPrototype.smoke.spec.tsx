import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { defaultFactoriesFixture, OPEN_WORK_ORDER, PRIMARY_FACTORY_KEY } from "./factoryPageResponses";
import { FactoriesHarness } from "./FactoriesHarness";

/**
 * Locks the prototype's main proof: the harnessed Work Order Detail page
 * shows boolean (pass/fail) checks — fed through
 * `WorkOrderChecksPrototypeSlotContext` — mixed into the real Checks
 * section next to the existing scored checks.
 */
describe("Work order boolean checks prototype", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows boolean checks next to scored checks on the Work Order Detail page", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-order/${OPEN_WORK_ORDER.number}`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const ciCard = await screen.findByTestId("work-order-check-check-ci", {}, { timeout: 8000 });
    expect(ciCard).toHaveTextContent("Pass");

    const securityScanCard = screen.getByTestId("work-order-check-check-security-scan");
    expect(securityScanCard).toHaveTextContent("Fail");

    // A scored check still renders alongside the boolean ones.
    expect(screen.getByTestId("work-order-check-check-risk-review")).toHaveTextContent("65");
  }, 10000);
});
