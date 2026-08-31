import { render, screen, waitFor, within } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

describe("FactorySettingsLayout sidebar", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  it("shows every approved group and selects the current item", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(screen.getByTestId("factory-settings-main").className).toMatch(/overflow-y-auto/);
    expect(within(sidebar).getByTestId("factory-settings-back")).toHaveTextContent("Back to workspace");
    expect(within(sidebar).getByTestId("factory-settings-account-nav")).toHaveTextContent("Account");
    expect(within(sidebar).getByTestId("factory-settings-workspace-nav")).toHaveTextContent("Workspace");
    expect(within(sidebar).getByTestId("factory-settings-organization-nav")).toHaveTextContent("Organization");
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-general")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(within(sidebar).queryByText("Environments")).not.toBeInTheDocument();
    expect(within(sidebar).queryByText("Groups")).not.toBeInTheDocument();
    expect(within(sidebar).queryByText("Roles")).not.toBeInTheDocument();
  }, 10000);

  it("shows the Account General page without replacing the unified navigation", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-account-general")).toHaveAttribute("aria-current", "page");
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-general")).toBeInTheDocument();
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-general")).toBeInTheDocument();

    const profile = await screen.findByTestId("factory-settings-profile-form");
    expect(within(profile).getByText("Name")).toBeInTheDocument();
  }, 10000);

  it("redirects unknown legacy paths to Workspace General", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/unknown/nested`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    await waitFor(() => {
      expect(within(sidebar).getByTestId("factory-settings-nav-workspace-general")).toHaveAttribute(
        "aria-current",
        "page",
      );
    });
  });

  it.each([
    ["API keys", "api-keys"],
    ["Secrets", "secrets"],
  ])("selects the reused Organization %s page in the factory settings shell", async (_title, path) => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/${path}`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId(`factory-settings-nav-organization-${path}`)).toHaveAttribute(
      "aria-current",
      "page",
    );
  });
});
