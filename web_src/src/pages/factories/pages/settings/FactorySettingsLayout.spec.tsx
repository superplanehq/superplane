import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_MODELS } from "@/lib/experimentalFeatures";
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

  it("opens the Account Profile redesign without replacing the unified navigation", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    await waitFor(() => {
      expect(within(sidebar).getByTestId("factory-settings-nav-account-profile")).toHaveAttribute(
        "aria-current",
        "page",
      );
    });
    expect(within(sidebar).getByTestId("factory-settings-nav-account-security")).toBeInTheDocument();
    expect(within(sidebar).getByTestId("factory-settings-nav-account-notifications")).toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-account-linked-accounts")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-account-general")).not.toBeInTheDocument();
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-general")).toBeInTheDocument();
    expect(within(sidebar).getByTestId("factory-settings-nav-organization-general")).toBeInTheDocument();

    const profile = await screen.findByTestId("account-redesign-identity");
    expect(within(profile).getByText("Name")).toBeInTheDocument();
  }, 10000);

  it("redirects linked-accounts to Profile", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/linked-accounts`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    await waitFor(() => {
      expect(within(sidebar).getByTestId("factory-settings-nav-account-profile")).toHaveAttribute(
        "aria-current",
        "page",
      );
    });
    expect(await screen.findByTestId("account-redesign-velocity-github")).toBeInTheDocument();
  }, 10000);

  it("shows the shipped Notifications page from Account settings", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/profile`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    await user.click(within(sidebar).getByTestId("factory-settings-nav-account-notifications"));
    expect(await screen.findByTestId("account-redesign-notifications")).toBeInTheDocument();
    expect(screen.getByText("Send task emails")).toBeInTheDocument();
    expect(screen.getByText("All workspaces")).toBeInTheDocument();
    expect(screen.getByText("Selected workspaces")).toBeInTheDocument();
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

  describe("Find settings", () => {
    it("shows section results for Security and hides the grouped nav", async () => {
      const user = userEvent.setup();
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/profile`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      await user.type(within(sidebar).getByTestId("factory-settings-find"), "secur");
      expect(within(sidebar).getByTestId("factory-settings-search-results")).toBeInTheDocument();
      expect(within(sidebar).getByText("Security")).toBeInTheDocument();
      expect(within(sidebar).getByText("Sign in methods")).toBeInTheDocument();
      expect(within(sidebar).queryByTestId("factory-settings-account-nav")).not.toBeInTheDocument();
    }, 10000);

    it("shows Spending results when the query matches billing", async () => {
      const user = userEvent.setup();
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      await user.type(within(sidebar).getByTestId("factory-settings-find"), "billing");
      const results = within(sidebar).getByTestId("factory-settings-search-results");
      expect(within(results).getAllByText("Spending").length).toBeGreaterThanOrEqual(2);
      expect(within(sidebar).queryByTestId("factory-settings-workspace-nav")).not.toBeInTheDocument();
    }, 10000);

    it("shows Workspace key and scrolls to that field when opened", async () => {
      const user = userEvent.setup();
      const scrollIntoView = vi.fn();
      Element.prototype.scrollIntoView = scrollIntoView;

      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/profile`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      await user.type(within(sidebar).getByTestId("factory-settings-find"), "workspace key");
      await user.click(within(sidebar).getByTestId("factory-settings-search-section:workspace:general:factory-settings-key"));
      expect(await screen.findByTestId("factory-settings-key")).toBeInTheDocument();
      expect(screen.getByTestId("factory-settings-key").closest("#factory-settings-key")).toHaveAttribute(
        "id",
        "factory-settings-key",
      );
      await waitFor(() => {
        expect(scrollIntoView).toHaveBeenCalled();
      });
    }, 10000);

    it("shows Claude under Organization Integrations", async () => {
      const user = userEvent.setup();
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      await user.type(within(sidebar).getByTestId("factory-settings-find"), "claude");
      const claudeResult = await waitFor(
        () => within(sidebar).getByTestId("factory-settings-search-integration:claude"),
        { timeout: 8000 },
      );
      expect(claudeResult).toHaveTextContent("Claude");
      expect(claudeResult).toHaveTextContent("Organization › Integrations");
      expect(claudeResult).toHaveAttribute(
        "href",
        expect.stringContaining("/settings/organization/integrations?section=integration-claude"),
      );
      await user.click(claudeResult);
      expect(await screen.findByTestId("workspace-page-header-title", {}, { timeout: 8000 })).toHaveTextContent(
        "Integrations",
      );
    }, 15000);

    it("shows an empty state when nothing matches", async () => {
      const user = userEvent.setup();
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      await user.type(within(sidebar).getByTestId("factory-settings-find"), "zzzz-no-match");
      expect(within(sidebar).getByTestId("factory-settings-find-empty")).toHaveTextContent("No matching settings.");
      expect(within(sidebar).queryByTestId("factory-settings-account-nav")).not.toBeInTheDocument();
    }, 10000);

    it("keeps the find query after navigating to a match", async () => {
      const user = userEvent.setup();
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/account/profile`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      const findInput = within(sidebar).getByTestId("factory-settings-find");
      await user.type(findInput, "sign in methods");
      await user.click(
        within(sidebar).getByTestId("factory-settings-search-section:account:security:account-redesign-signin"),
      );
      expect(await screen.findByTestId("account-redesign-signin")).toBeInTheDocument();
      expect(within(sidebar).getByTestId("factory-settings-find")).toHaveValue("sign in methods");
      expect(within(sidebar).getByTestId("factory-settings-search-results")).toBeInTheDocument();
    }, 10000);
  });

  describe("workspace-models experimental feature", () => {
    it("hides the Models nav item when the feature is off", async () => {
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-models")).not.toBeInTheDocument();
    }, 10000);

    it("shows the Models nav item when the feature is on", async () => {
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
          factoriesFixture={defaultFactoriesFixture}
          experimentalFeatures={[FEATURE_WORKSPACE_MODELS]}
        />,
      );

      const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      expect(within(sidebar).getByTestId("factory-settings-nav-workspace-models")).toHaveTextContent("Models");
    }, 10000);

    it("redirects away from the Models route when the feature is off", async () => {
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/models`}
          factoriesFixture={defaultFactoriesFixture}
        />,
      );

      await waitFor(() => {
        expect(screen.queryByTestId("factory-settings-sidebar")).not.toBeInTheDocument();
      });
      expect(screen.queryByTestId("workspace-page-header-title")).not.toBeInTheDocument();
    }, 10000);

    it("renders the Models page when the feature is on", async () => {
      render(
        <FactoriesHarness
          pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/models`}
          factoriesFixture={defaultFactoriesFixture}
          experimentalFeatures={[FEATURE_WORKSPACE_MODELS]}
        />,
      );

      await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
      expect(await screen.findByTestId("workspace-page-header-title")).toHaveTextContent("Models");
    }, 10000);
  });
});
