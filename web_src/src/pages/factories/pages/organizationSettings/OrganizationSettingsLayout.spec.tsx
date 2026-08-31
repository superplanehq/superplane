import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { ORGANIZATION_SETTINGS_NAV_ITEMS } from "./organizationSettingsNavItems";

describe("OrganizationSettingsLayout", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows back to workspaces, the organization name, and the settings menu", async () => {
    render(<FactoriesHarness pathSuffix="organization" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    expect(await screen.findByTestId("organization-settings-overview-name")).toHaveTextContent("SuperPlane");

    const backLink = within(sidebar).getByTestId("organization-settings-back");
    const name = within(sidebar).getByTestId("organization-settings-name");
    const general = within(sidebar).getByTestId("organization-settings-nav-general");
    const labels = ORGANIZATION_SETTINGS_NAV_ITEMS.map((item) =>
      within(sidebar).getByTestId(`organization-settings-nav-${item.id}`),
    );

    expect(backLink).toHaveTextContent("Back to workspaces");
    expect(name).toHaveTextContent("SuperPlane");
    expect(labels.map((item) => item.textContent)).toEqual(ORGANIZATION_SETTINGS_NAV_ITEMS.map((item) => item.label));
    expect(within(sidebar).getByTestId("organization-settings-nav-workspaces")).not.toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(general).toHaveAttribute("aria-current", "page");
    expect(backLink.compareDocumentPosition(name) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(name.compareDocumentPosition(general) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  }, 10000);

  it("redirects the old workspace-nested URL and returns to that workspace", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/organization/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    const backLink = within(sidebar).getByTestId("organization-settings-back");
    expect(backLink).toHaveTextContent("Back to workspace");
    expect(backLink).toHaveAttribute("href", `/${FACTORIES_ORGANIZATION_ID}/workspaces/${PRIMARY_FACTORY_KEY}`);
  }, 10000);

  it("opens a workspaces table from the Workspaces menu item", async () => {
    const user = userEvent.setup();
    render(<FactoriesHarness pathSuffix="organization/general" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    await user.click(within(sidebar).getByTestId("organization-settings-nav-workspaces"));

    const table = await screen.findByTestId("organization-settings-workspaces-table");
    expect(table).toHaveTextContent("Semaphore");
    expect(table).toHaveTextContent("SuperPlane");
    expect(table).not.toHaveTextContent("RF");
    expect(within(sidebar).getByTestId("organization-settings-nav-workspaces")).toHaveAttribute("aria-current", "page");
  }, 10000);

  it("opens Workspace usage from the menu", async () => {
    const user = userEvent.setup();
    render(<FactoriesHarness pathSuffix="organization/general" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    await user.click(within(sidebar).getByTestId("organization-settings-nav-workspace-usage"));

    expect(
      await screen.findByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
    expect(within(sidebar).getByTestId("organization-settings-nav-workspace-usage")).toHaveAttribute(
      "aria-current",
      "page",
    );
  }, 10000);

  it("opens Integrations from the menu", async () => {
    const user = userEvent.setup();
    render(<FactoriesHarness pathSuffix="organization/general" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    await user.click(within(sidebar).getByTestId("organization-settings-nav-integrations"));

    expect(await screen.findByText("Connect external tools and services to extend SuperPlane.")).toBeInTheDocument();
    expect(within(sidebar).getByTestId("organization-settings-nav-integrations")).toHaveAttribute(
      "aria-current",
      "page",
    );
  }, 10000);

  it("opens a coming-soon page for Members", async () => {
    const user = userEvent.setup();
    render(<FactoriesHarness pathSuffix="organization/general" factoriesFixture={defaultFactoriesFixture} />);

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    await user.click(within(sidebar).getByTestId("organization-settings-nav-members"));

    expect(await screen.findByTestId("factory-settings-soon")).toBeInTheDocument();
    expect(within(sidebar).getByTestId("organization-settings-nav-members")).toHaveAttribute("aria-current", "page");
  }, 10000);

  it("redirects the old organization LLM spend URL into Workspace usage", async () => {
    render(
      <FactoriesHarness pathSuffix="organization/llm-spend?credit=added" factoriesFixture={defaultFactoriesFixture} />,
    );

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("organization-settings-nav-workspace-usage")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(
      await screen.findByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
  }, 10000);

  it("redirects the old LLM spend URL into organization settings", async () => {
    render(
      <FactoriesHarness pathSuffix="settings/llm-spend?credit=added" factoriesFixture={defaultFactoriesFixture} />,
    );

    const sidebar = await screen.findByTestId("organization-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("organization-settings-nav-workspace-usage")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(
      await screen.findByText("Review factory token usage, VM time, and estimated spend for this organization."),
    ).toBeInTheDocument();
    expect(await screen.findByText("Refreshing hosted credit totals.")).toBeInTheDocument();
  }, 10000);
});
