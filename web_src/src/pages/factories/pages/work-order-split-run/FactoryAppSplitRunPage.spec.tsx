import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { REFUND_IMPLEMENTER_APP } from "../../__fixtures__/factoryOwnedCanvasFixture";
import {
  LINE_RUN_IMPLEMENT_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
} from "../../__fixtures__/factoryPageResponses";
import { lineMetricsFactoriesFixture } from "../../__fixtures__/lineMetricsFactoriesFixture";

describe("FactoryAppSplitRunPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows only the implement log beside the implementation canvas", async () => {
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}/split-run?from=lines&lineId=${line.id}&run=${LINE_RUN_IMPLEMENT_ID}&orderNumber=103&canvas=implementation`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    const page = await screen.findByTestId("factory-app-split-run-page", {}, { timeout: 8000 });
    expect(within(page).getByTestId("factory-app-canvas-title")).toHaveTextContent("Implementation");
    expect(within(page).getByTestId("split-run-phase-implement")).toBeInTheDocument();
    expect(within(page).queryByTestId("split-run-phase-plan")).not.toBeInTheDocument();
    expect(within(page).getByTestId("split-run-stream-implement")).toBeInTheDocument();
    expect(within(page).getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
    expect(within(page).getByTestId("split-run-resize-handle")).toBeInTheDocument();
    expect(within(page).queryByTestId("split-run-canvas-expand")).not.toBeInTheDocument();
    expect(within(page).queryByTestId("split-run-canvas-menu")).not.toBeInTheDocument();
    expect(within(page).getByTestId("factory-app-edit")).toHaveTextContent("Edit");
  }, 10000);

  it("redirects the deprecated canvas run view to the split run page", async () => {
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}?run=${LINE_RUN_IMPLEMENT_ID}&from=lines&lineId=${line.id}&orderNumber=103`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("factory-app-split-run-page", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);

  it("opens configure from the header Edit button", async () => {
    const user = userEvent.setup();
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}/split-run?from=lines&lineId=${line.id}&run=${LINE_RUN_IMPLEMENT_ID}&orderNumber=103&canvas=implementation`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    const page = await screen.findByTestId("factory-app-split-run-page", {}, { timeout: 8000 });
    await user.click(within(page).getByTestId("factory-app-edit"));

    const canvasPage = await screen.findByTestId("factory-app-canvas-page", {}, { timeout: 8000 });
    expect(canvasPage).toHaveAttribute("data-configure", "true");
  }, 10000);
});
