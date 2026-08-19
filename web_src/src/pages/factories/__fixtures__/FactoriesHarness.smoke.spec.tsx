import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import {
  LINE_RUN_IMPLEMENT_FAILED_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
  defaultFactoriesFixture,
} from "./factoryPageResponses";
import { FactoriesHarness } from "./FactoriesHarness";
import { REFUND_IMPLEMENTER_APP, refundLineCanvasFixture } from "./factoryOwnedCanvasFixture";

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

  it("opens the seeded agent sidebar in factory edit mode", async () => {
    const implementerAppId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    const lineId = REFUND_FACTORY_LINES[0]?.id ?? "line-plan-and-implement";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&agent=1&run=${LINE_RUN_IMPLEMENT_FAILED_ID}&from=lines&lineId=${lineId}`}
        factoriesFixture={defaultFactoriesFixture}
        appFixture={refundLineCanvasFixture()}
      />,
    );

    expect(await screen.findByTestId("factory-app-canvas-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(await screen.findByTestId("canvas-tool-sidebar", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(await screen.findByText(/Semaphore CI retries/i, {}, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);
});
