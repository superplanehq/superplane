import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { WorkOrdersPage } from "../pages/WorkOrdersPage";
import { PRIMARY_FACTORY_KEY, defaultFactoriesFixture } from "./factoryPageResponses";
import { FactoriesHarness } from "./FactoriesHarness";

// The harness swaps in a Storybook-only Missions rail variant of the Work
// Orders page by default; use the real production `WorkOrdersPage` (the one
// with the reported bug) so these tests exercise the actual fix.
const pageOverrides = { workOrders: WorkOrdersPage };

describe("client-side navigation updates document.title", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens a task permalink on the line board", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/task/101`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    const permalinkPopup = await screen.findByTestId("work-order-split-run", {}, { timeout: 8000 });
    expect(
      within(permalinkPopup).getByRole("heading", { name: "Reconcile duplicate refunds in ledger" }),
    ).toBeInTheDocument();
    expect(document.title).toBe("Plan and Implement · Semaphore · SuperPlane");
  }, 15000);

  it("canonicalizes a legacy /work-orders/:id URL onto the task permalink", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders/wo-open-refunds`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    const legacyPopup = await screen.findByTestId("work-order-split-run", {}, { timeout: 8000 });
    expect(
      within(legacyPopup).getByRole("heading", { name: "Reconcile duplicate refunds in ledger" }),
    ).toBeInTheDocument();
  }, 15000);

  it("redirects the legacy /work-orders list to /tasks", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("work-orders-header", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Tasks · Semaphore · SuperPlane");
  }, 15000);

  it("redirects the legacy singular /work-order/:number permalink to /task/:number", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-order/101`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    const permalinkPopup = await screen.findByTestId("work-order-split-run", {}, { timeout: 8000 });
    expect(
      within(permalinkPopup).getByRole("heading", { name: "Reconcile duplicate refunds in ledger" }),
    ).toBeInTheDocument();
    expect(document.title).toBe("Plan and Implement · Semaphore · SuperPlane");
  }, 15000);

  it("updates the tab title on the line board home", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines/line-plan-and-implement`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Plan and Implement · Semaphore · SuperPlane");
  }, 15000);

  it("sends /lines to the line board", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("lines-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Plan and Implement · Semaphore · SuperPlane");
  }, 15000);

  it("updates the tab title when selecting an automation inline, then back to the list", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/automations`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("automations-list-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Automations · Semaphore · SuperPlane");

    await user.click(await screen.findByTestId("automations-app-app-refund-planner", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("automations-detail-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Refund Planner · Semaphore · SuperPlane");

    await user.click(screen.getByTestId("automations-detail-back"));
    expect(await screen.findByTestId("automations-list-page", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Automations · Semaphore · SuperPlane");
  }, 15000);

  it("sets the tab title from the canvas name on a factory-owned app canvas", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/apps/app-refund-implementer`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByText("Refund Implementer", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Refund Implementer · Semaphore · SuperPlane");
  }, 15000);

  it("sets a distinct tab title for each factory settings section", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("factory-settings-general-form", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("General · Settings · Semaphore · SuperPlane");

    await user.click(screen.getByTestId("factory-settings-nav-organization-members"));
    await waitFor(() => expect(document.title).toBe("Members · SuperPlane"), { timeout: 8000 });
  }, 15000);

  it("sets the tab title on the Missions and Wiki coming-soon pages", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/missions`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={{ ...pageOverrides, wiki: undefined }}
      />,
    );

    expect(await screen.findByTestId("coming-soon-body", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Missions · SuperPlane");
  }, 15000);

  it("sets the tab title on the Wiki coming-soon page", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/wiki`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={{ ...pageOverrides, wiki: undefined }}
      />,
    );

    expect(await screen.findByTestId("coming-soon-body", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Wiki · SuperPlane");
  }, 15000);
});
