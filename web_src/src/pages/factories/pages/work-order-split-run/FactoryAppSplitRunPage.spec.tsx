import { render, screen, waitFor, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { REFUND_IMPLEMENTER_APP, refundLineCanvasFixture } from "../../__fixtures__/factoryOwnedCanvasFixture";
import {
  defaultFactoriesFixture,
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

    await waitFor(() => {
      expect(screen.getByTestId("split-run-phase-implement-0")).toBeInTheDocument();
    });
    const page = screen.getByTestId("factory-app-split-run-page");
    expect(screen.getByTestId("factory-app-canvas-title")).toHaveTextContent("Implementation");
    expect(within(page).queryByTestId("split-run-phase-refund-planner-0")).not.toBeInTheDocument();
    expect(within(page).getByTestId("split-run-stream-implement-0")).toBeInTheDocument();
    expect(within(page).getByTestId("run-overlay-compact-canvas")).toBeInTheDocument();
    expect(within(page).getByTestId("split-run-resize-handle")).toBeInTheDocument();
    expect(within(page).queryByTestId("split-run-canvas-expand")).not.toBeInTheDocument();
    expect(within(page).queryByTestId("split-run-canvas-menu")).not.toBeInTheDocument();
    expect(within(page).getByTestId("factory-app-edit")).toHaveTextContent("Edit Automation");
  }, 10000);

  it("opens the planning canvas from ingest when the URL omits canvas", async () => {
    const line = REFUND_FACTORY_LINES[0];

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/app-refund-planner/split-run?from=lines&lineId=${line.id}&orderNumber=103&canvas=planning`}
        factoriesFixture={lineMetricsFactoriesFixture}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("split-run-phase-plan")).toBeInTheDocument();
    });
    expect(within(screen.getByTestId("factory-app-split-run-page")).getAllByText("Create plan").length).toBeGreaterThan(
      0,
    );
  }, 10000);

  it("does not show the demo run when no run is selected", async () => {
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}/split-run`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const page = await screen.findByTestId("factory-app-split-run-page", {}, { timeout: 8000 });
    await waitFor(() => {
      expect(page).toHaveAttribute("data-state", "not-found");
    });
    expect(screen.queryByText("Add refund reconciliation test")).not.toBeInTheDocument();
  }, 10000);

  it("does not show the demo run when the selected run is missing", async () => {
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}/split-run?from=lines&lineId=${line.id}&run=run-missing`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const page = await screen.findByTestId("factory-app-split-run-page", {}, { timeout: 8000 });
    await waitFor(() => {
      expect(page).toHaveAttribute("data-state", "not-found");
    });
    expect(screen.queryByText("Add refund reconciliation test")).not.toBeInTheDocument();
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

  it("points Edit at the configure canvas", async () => {
    const line = REFUND_FACTORY_LINES[0];
    const appId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${appId}/split-run?from=lines&lineId=${line.id}&run=${LINE_RUN_IMPLEMENT_ID}&orderNumber=103&canvas=implementation`}
        factoriesFixture={defaultFactoriesFixture}
        appFixture={refundLineCanvasFixture()}
      />,
    );

    const page = await screen.findByTestId("factory-app-split-run-page", {}, { timeout: 8000 });
    const edit = within(page).getByTestId("factory-app-edit");
    expect(edit).toHaveAttribute("href", expect.stringContaining(`/apps/${appId}?`));
    expect(edit).toHaveAttribute("href", expect.stringContaining("configure=1"));
    expect(edit.getAttribute("href")).not.toContain("split-run");
  }, 10000);
});
