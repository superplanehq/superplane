import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { factorySettingsPath } from "../lib/factoryPagePaths";
import { FactoriesHarness } from "./FactoriesHarness";
import { REFUND_IMPLEMENTER_APP, refundLineCanvasFixture } from "./factoryOwnedCanvasFixture";
import {
  ACME_ONBOARDING_FACTORY_ID,
  ACME_ONBOARDING_FACTORY_KEY,
  ACME_ONBOARDING_LINE_ID,
  FACTORIES_ORGANIZATION_ID,
  GITHUB_ISSUES_INTAKE_ID,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY_LINES,
  defaultFactoriesFixture,
} from "./factoryPageResponses";
import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";
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

  it("keeps a canvas run view on the factory inspector page", async () => {
    const implementerAppId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    const lineId = REFUND_FACTORY_LINES[0]?.id ?? "line-plan-and-implement";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?run=${LINE_RUN_IMPLEMENT_FAILED_ID}&from=lines&lineId=${lineId}`}
        factoriesFixture={defaultFactoriesFixture}
        appFixture={refundLineCanvasFixture()}
      />,
    );

    expect(await screen.findByTestId("factory-app-canvas-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(await screen.findByTestId("factory-app-edit", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("factory-app-split-run-page")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-app-workspace-toggles")).not.toBeInTheDocument();
  }, 15000);

  it("shows the edit workspace chrome in factory Configure", async () => {
    const implementerAppId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    const lineId = REFUND_FACTORY_LINES[0]?.id ?? "line-plan-and-implement";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&run=${LINE_RUN_IMPLEMENT_FAILED_ID}&from=lines&lineId=${lineId}`}
        factoriesFixture={defaultFactoriesFixture}
        appFixture={refundLineCanvasFixture()}
      />,
    );

    expect(await screen.findByTestId("factory-app-canvas-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(await screen.findByTestId("factory-app-discard", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("factory-app-save")).toBeInTheDocument();
    expect(screen.getByTestId("factory-app-workspace-agent")).toBeInTheDocument();
    expect(screen.getByTestId("factory-app-workspace-components")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByTestId("factory-app-more-options")).toBeInTheDocument();
    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();
  }, 15000);

  it("opens the agent sidebar from the factory Configure header", async () => {
    const user = userEvent.setup();
    const implementerAppId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    const lineId = REFUND_FACTORY_LINES[0]?.id ?? "line-plan-and-implement";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&run=${LINE_RUN_IMPLEMENT_FAILED_ID}&from=lines&lineId=${lineId}`}
        factoriesFixture={defaultFactoriesFixture}
        appFixture={refundLineCanvasFixture()}
      />,
    );

    expect(await screen.findByTestId("factory-app-canvas-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("canvas-tool-sidebar")).not.toBeInTheDocument();

    await user.click(await screen.findByTestId("factory-app-workspace-agent", {}, { timeout: 8000 }));

    expect(await screen.findByTestId("canvas-tool-sidebar", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);

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

  it("toggles the components sidebar from the factory Configure header", async () => {
    const user = userEvent.setup();
    const implementerAppId = REFUND_IMPLEMENTER_APP.id ?? "app-refund-implementer";
    const lineId = REFUND_FACTORY_LINES[0]?.id ?? "line-plan-and-implement";

    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/${implementerAppId}?configure=1&run=${LINE_RUN_IMPLEMENT_FAILED_ID}&from=lines&lineId=${lineId}`}
        factoriesFixture={defaultFactoriesFixture}
        appFixture={refundLineCanvasFixture()}
      />,
    );

    expect(await screen.findByTestId("factory-app-canvas-page", {}, { timeout: 8000 })).toBeInTheDocument();
    const componentsToggle = await screen.findByTestId("factory-app-workspace-components", {}, { timeout: 8000 });
    expect(componentsToggle).toHaveAttribute("aria-pressed", "false");
    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();

    await user.click(componentsToggle);

    expect(await screen.findByTestId("building-blocks-sidebar", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(componentsToggle).toHaveAttribute("aria-pressed", "true");

    await user.click(componentsToggle);

    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();
    expect(componentsToggle).toHaveAttribute("aria-pressed", "false");
  }, 15000);

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
    expect(screen.queryByTestId("workspace-setup-cancel")).not.toBeInTheDocument();
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

    await user.click(await screen.findByTestId("first-run-get-started", {}, { timeout: 8000 }));
    await user.click(await screen.findByTestId("first-run-github-continue", {}, { timeout: 8000 }));

    expect(await screen.findByRole("option", { name: /acme\/api/ }, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);

  it("opens the repository list when GitHub sends the browser back to the VCS step", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=vcs&pick=newest`}
        factoriesFixture={defaultFactoriesFixture}
        onboardingSeed={{ pending: { workspaceId: PRIMARY_FACTORY_ID, workspaceName: "Refunds Factory" } }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    expect(await screen.findByRole("option", { name: /acme\/api/ }, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);

  it("opens setup after Create new workspace", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/overview`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    await user.click(await screen.findByTestId("factories-workspace-switch", {}, { timeout: 8000 }));
    await user.click(screen.getByTestId("factories-workspace-create"));

    expect(await screen.findByTestId("workspace-setup", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByText("The workspace was created without a key")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factories-sidebar")).not.toBeInTheDocument();
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

    expect(await screen.findByTestId("first-run-tickets", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /GitHub Issues/ }, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);
});

describe("FactoriesHarness Acme onboarding", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens Acme onboarding from the switcher with an empty line board", async () => {
    const user = userEvent.setup();
    const planLineId = REFUND_FACTORY_LINES[0]?.id;
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/${planLineId}`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    await user.click(await screen.findByTestId("factories-workspace-switch", {}, { timeout: 8000 }));
    await user.click(screen.getByTestId(`factories-workspace-option-${ACME_ONBOARDING_FACTORY_ID}`));

    expect(
      await screen.findByRole("button", { name: /Switch workspace, Acme onboarding/ }, { timeout: 8000 }),
    ).toBeInTheDocument();
    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("lines-column-title-backlog")).toHaveTextContent("Backlog");
    await waitFor(() => {
      expect(screen.getByTestId("lines-column-title-phase-0")).toHaveTextContent("Implement");
    });
    expect(screen.getByTestId("lines-column-title-verify")).toHaveTextContent("Verify");
    expect(screen.getByTestId("lines-column-title-done")).toHaveTextContent("Done");
    expect(screen.queryByTestId("lines-column-title-phase-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("backlog-onboarding-card")).toBeInTheDocument();
    expect(screen.getByTestId("lines-backlog-column")).not.toHaveTextContent("No work orders in the backlog.");
    expect(screen.getByTestId("lines-phase-column-0")).toHaveTextContent("Nothing here.");
    expect(screen.getByTestId("lines-verify-column")).toHaveTextContent("No work orders in Verify.");
    expect(screen.getByTestId("lines-done-column")).toHaveTextContent("No work orders in Done.");
    expect(screen.queryByTestId("lines-phase-column-1")).not.toBeInTheDocument();
    expect(screen.queryAllByTestId(/^work-order-card-/)).toHaveLength(0);
  }, 15000);

  it("opens the populated Semaphore line board from the first-run Acme board", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${ACME_ONBOARDING_FACTORY_KEY}/lines/${ACME_ONBOARDING_LINE_ID}?intake=1&intakeId=${GITHUB_ISSUES_INTAKE_ID}`}
        factoriesFixture={lineMetricsFactoriesFixture}
        enableOnboarding={false}
      />,
    );

    await user.click(await screen.findByTestId("factories-workspace-switch", {}, { timeout: 8000 }));
    await user.click(screen.getByTestId(`factories-workspace-option-${PRIMARY_FACTORY_ID}`));

    expect(
      await screen.findByRole("button", { name: /Switch workspace, Semaphore/ }, { timeout: 8000 }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("factories-sidebar")).toBeInTheDocument();
    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("workspace-setup")).not.toBeInTheDocument();
    expect((await screen.findAllByTestId(/^work-order-card-/, {}, { timeout: 8000 })).length).toBeGreaterThan(0);
  }, 15000);
});
