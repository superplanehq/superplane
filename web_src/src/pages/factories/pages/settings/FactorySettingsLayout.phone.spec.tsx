import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
} from "../../__fixtures__/factoryPageResponses";

describe("FactorySettingsLayout at phone width", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  const desktopWidth = window.innerWidth;

  beforeEach(() => {
    Object.defineProperty(window, "innerWidth", { configurable: true, writable: true, value: 390 });
  });

  afterEach(() => {
    Object.defineProperty(window, "innerWidth", { configurable: true, writable: true, value: desktopWidth });
  });

  it("lists every settings group on the index instead of the sidebars", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const index = await screen.findByTestId("factory-settings-mobile-index", {}, { timeout: 8000 });
    expect(screen.queryByTestId("factories-sidebar")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-settings-sidebar")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-settings-mobile-bar")).not.toBeInTheDocument();
    expect(within(index).getByTestId("factory-settings-account-heading")).toBeInTheDocument();
    expect(within(index).getByTestId("factory-settings-workspace-heading")).toHaveTextContent("Workspace · RF");
    expect(within(index).getByTestId("factory-settings-organization-heading")).toHaveTextContent("Organization");
    expect(within(index).getByTestId("factory-settings-nav-organization-members")).toHaveTextContent("Members");
    expect(within(index).getByTestId("factory-settings-mobile-index-back")).toHaveAttribute(
      "href",
      expect.stringContaining(`/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/`),
    );
  }, 10000);

  it("opens a page from the index and returns to it from the back bar", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const index = await screen.findByTestId("factory-settings-mobile-index", {}, { timeout: 8000 });
    await user.click(within(index).getByTestId("factory-settings-nav-organization-members"));

    expect(await screen.findByTestId("workspace-page-header-title", {}, { timeout: 8000 })).toHaveTextContent(
      "Members",
    );
    expect(screen.queryByTestId("factory-settings-mobile-index")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-settings-sidebar")).not.toBeInTheDocument();
    const back = within(screen.getByTestId("factory-settings-mobile-bar")).getByTestId("factory-settings-mobile-back");
    expect(back).toHaveTextContent("Settings");

    await user.click(back);
    expect(await screen.findByTestId("factory-settings-mobile-index")).toBeInTheDocument();
  }, 15000);

  it("filters the index from Find settings", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const index = await screen.findByTestId("factory-settings-mobile-index", {}, { timeout: 8000 });
    await user.type(within(index).getByTestId("factory-settings-find"), "secret");

    const results = within(index).getByTestId("factory-settings-search-results");
    expect(within(results).getByText("Secrets")).toBeInTheDocument();
    expect(within(index).queryByTestId("factory-settings-nav-workspace-general")).not.toBeInTheDocument();
  }, 10000);

  it("keeps the bottom bar on settings pages at phone width", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/members`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const bottomBar = await screen.findByTestId("mobile-bottom-bar", {}, { timeout: 8000 });
    const shell = screen.getByTestId("factory-settings-layout");
    expect(shell.className).not.toContain("h-dvh");
    expect(shell).toHaveStyle({ position: "fixed", left: "0px", right: "0px" });
    expect(shell.style.bottom).toBe("");
    expect(within(bottomBar).getByTestId("mobile-tab-more")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("factory-settings-mobile-bar")).toBeInTheDocument();
  }, 10000);
});
