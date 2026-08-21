import { render, screen } from "@testing-library/react";
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

  // Regression test for the reported bug: visiting a work order detail page
  // sets a specific title, and navigating back to the Work Orders list left
  // that title in place because `WorkOrdersPage` never called `usePageTitle`.
  it("resets the tab title after navigating from a work order detail page back to the list", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-order/101`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("work-order-detail-back", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toContain("Reconcile duplicate refunds in ledger");
    expect(document.title).toContain("Semaphore");

    await user.click(screen.getByTestId("work-order-detail-back"));

    expect(await screen.findByTestId("work-orders-header", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).not.toContain("Reconcile duplicate refunds in ledger");
    expect(document.title).toBe("Work Orders · Semaphore · SuperPlane");
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

  it("sets the tab title on the lines list", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("lines-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Lines · Semaphore · SuperPlane");
  }, 15000);

  it("updates the tab title when selecting a line from the list", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/lines`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("lines-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Lines · Semaphore · SuperPlane");

    await user.click(screen.getByTestId("lines-card-line-plan-and-implement"));
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
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/general`}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={pageOverrides}
      />,
    );

    expect(await screen.findByTestId("factory-settings-general-form", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("General · Settings · Semaphore · SuperPlane");

    await user.click(screen.getByTestId("factory-settings-nav-members"));
    expect(await screen.findByTestId("factory-settings-soon", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(document.title).toBe("Members · Settings · SuperPlane");
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
