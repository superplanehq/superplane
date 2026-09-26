import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { HEADER_MCP_RESOURCE, OAUTH_NOT_CONNECTED_RESOURCE } from "../../__fixtures__/agentResourceFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { GITHUB_PERSONAL_ACCESS_TOKEN_URL } from "./agentResourceCopy";

const mcpPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/mcp`;
const mcpAndSkills = [FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS];

describe("FactorySettingsMCPPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  it("hides the nav items when the feature is off", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-mcp")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-skills")).not.toBeInTheDocument();
  }, 10000);

  it("shows the nav items when both features are on", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-mcp")).toHaveTextContent("MCP servers");
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-skills")).toHaveTextContent("Skills");
  }, 10000);

  it("shows only MCP nav when only workspace_mcp is on", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_MCP]}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-mcp")).toHaveTextContent("MCP servers");
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-skills")).not.toBeInTheDocument();
  }, 10000);

  it("shows only Skills nav when only workspace_skills is on", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_SKILLS]}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-skills")).toHaveTextContent("Skills");
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-mcp")).not.toBeInTheDocument();
  }, 10000);

  it("redirects away from the route when the feature is off", async () => {
    render(<FactoriesHarness pathSuffix={mcpPath} factoriesFixture={defaultFactoriesFixture} />);

    await waitFor(() => {
      expect(screen.queryByTestId("factory-settings-sidebar")).not.toBeInTheDocument();
    });
    expect(screen.queryByTestId("factory-settings-mcp")).not.toBeInTheDocument();
  }, 10000);

  it("redirects the legacy agent-resources route to MCP", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent-resources`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("factory-settings-mcp", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);

  it("redirects the legacy skills tab to Skills", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent-resources?tab=skills`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("factory-settings-skills", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);

  it("shows the MCP servers empty state", async () => {
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("workspace-page-header-title", {}, { timeout: 8000 })).toHaveTextContent(
      "MCP servers",
    );
    expect(await screen.findByTestId("agent-resources-connections-empty")).toHaveTextContent("No MCP servers yet");
    expect(screen.getByTestId("agent-resources-add-connection")).toHaveTextContent("Add MCP server");
  }, 10000);

  it("lists a header MCP server with a connected status", async () => {
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("docs")).toBeInTheDocument();
    expect(screen.getByText("https://mcp.example.com/mcp")).toBeInTheDocument();
    expect(screen.queryByText("Header")).not.toBeInTheDocument();
    expect(screen.queryByText("Connected")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connected" })).toHaveAttribute("type", "button");
    expect(screen.getByTestId("mcp-status-connected")).toHaveAttribute("aria-label", "Connected");
    expect(screen.getByTestId(`agent-resource-view-tools-${HEADER_MCP_RESOURCE.id}`)).toHaveTextContent("Tools");

    const user = userEvent.setup();
    await user.hover(screen.getByTestId("mcp-status-connected"));
    expect(await screen.findByRole("tooltip")).toHaveTextContent("Connected");
  }, 10000);

  it("expands tools without showing descriptions", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(
      await screen.findByTestId(`agent-resource-view-tools-${HEADER_MCP_RESOURCE.id}`, {}, { timeout: 8000 }),
    );
    expect(await screen.findByTestId("mcp-tools-list")).toHaveTextContent("search");
    expect(screen.queryByText("Search the catalog.")).not.toBeInTheDocument();
  }, 10000);

  it("hides tools when sign-in is not connected", async () => {
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [OAUTH_NOT_CONNECTED_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 });
    expect(screen.getByRole("button", { name: "Not connected" })).toHaveAttribute("type", "button");
    expect(screen.getByTestId("mcp-status-disconnected")).toHaveAttribute("aria-label", "Not connected");
    expect(screen.queryByText("Sign-in")).not.toBeInTheDocument();
    expect(screen.queryByText("Not connected")).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`agent-resource-view-tools-${OAUTH_NOT_CONNECTED_RESOURCE.id}`),
    ).not.toBeInTheDocument();
  }, 10000);

  it("opens the MCP catalog from the query string", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-search")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-github")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-category-code")).toHaveTextContent("Code");
    expect(screen.getByTestId("mcp-catalog-category-observability")).toHaveTextContent("Observability");
    expect(screen.getByTestId("mcp-catalog-custom")).toHaveTextContent("Add custom");
  }, 10000);

  it("filters the catalog and keeps Add custom pinned", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.type(screen.getByTestId("mcp-catalog-search"), "github");
    expect(screen.getByTestId("mcp-catalog-github")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-category-code")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-category-issues")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-linear")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-custom")).toHaveTextContent("Add custom");

    await user.clear(screen.getByTestId("mcp-catalog-search"));
    await user.type(screen.getByTestId("mcp-catalog-search"), "zzzz");
    expect(screen.getByText("No servers match this search.")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-custom")).toBeInTheDocument();
  }, 10000);

  it("opens GitHub from the catalog with a token field", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-github"));

    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-add-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-url")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-auth")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-icon")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-token")).toBeInTheDocument();

    const instruction = screen.getByTestId("mcp-catalog-setup-instruction");
    expect(instruction).toHaveTextContent("Create a GitHub personal access token and paste it here.");
    expect(within(instruction).getByRole("link", { name: "GitHub personal access token" })).toHaveAttribute(
      "href",
      GITHUB_PERSONAL_ACCESS_TOKEN_URL,
    );
    expect(
      within(instruction).getByRole("link", { name: "GitHub personal access token" }).querySelector("svg"),
    ).not.toBeNull();
  }, 10000);

  it("opens Linear from the catalog with Sign in", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-linear"));

    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-url")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-auth")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-token")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-icon")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-instruction")).toHaveTextContent("Sign in with Linear.");
    expect(screen.getByTestId("mcp-catalog-setup-sign-in")).toHaveTextContent("Sign in");
    expect(screen.getByTestId("mcp-catalog-setup-sign-in").querySelector("svg")).not.toBeNull();
  }, 10000);

  it("opens Sentry from the catalog with Sign in", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-sentry"));

    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-token")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-icon")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-instruction")).toHaveTextContent("Sign in with Sentry.");
    expect(screen.getByTestId("mcp-catalog-setup-sign-in")).toHaveTextContent("Sign in");
  }, 10000);

  it("creates a GitHub server from a pasted token", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-github"));
    await user.type(await screen.findByTestId("mcp-catalog-setup-token"), "ghp_example");
    await user.click(screen.getByTestId("mcp-catalog-setup-save"));

    expect(await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 })).toHaveTextContent(
      "github",
    );
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
  }, 10000);

  it("starts Linear sign-in from the catalog", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-linear"));
    await user.click(await screen.findByTestId("mcp-catalog-setup-sign-in"));

    await waitFor(() => {
      expect(assign).toHaveBeenCalledWith("https://auth.example.com/authorize?client_id=storybook");
    });
    vi.unstubAllGlobals();
  }, 10000);

  it("opens Add custom without catalog instructions", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-custom"));

    expect(await screen.findByTestId("agent-resource-connection-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-name")).toHaveValue("");
    expect(screen.getByTestId("agent-resource-url")).toHaveValue("");
    expect(screen.getByTestId("agent-resource-auth")).toHaveTextContent("Sign-in");
  }, 10000);

  it("opens edit when the server name is clicked", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByTestId(`agent-resource-edit-${HEADER_MCP_RESOURCE.id}`, {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-resource-connection-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-auth")).toHaveTextContent("Header");
  }, 10000);

  it("opens Sign in when a catalog OAuth server is edited", async () => {
    const user = userEvent.setup();
    const sentryResource = {
      ...OAUTH_NOT_CONNECTED_RESOURCE,
      id: "resource-sentry",
      name: "sentry",
      url: "https://mcp.sentry.dev/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [sentryResource] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByTestId(`agent-resource-edit-${sentryResource.id}`, {}, { timeout: 8000 }));
    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-name")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-sign-in")).toHaveTextContent("Sign in");
  }, 10000);

  it("resumes Sign in from Add when the catalog OAuth server already exists", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();
    const sentryResource = {
      ...OAUTH_NOT_CONNECTED_RESOURCE,
      id: "resource-sentry",
      name: "sentry",
      url: "https://mcp.sentry.dev/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={`${mcpPath}?dialog=add`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [sentryResource] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-sentry"));
    await user.click(await screen.findByTestId("mcp-catalog-setup-sign-in"));

    await waitFor(() => {
      expect(assign).toHaveBeenCalledWith("https://auth.example.com/authorize?client_id=storybook");
    });
    expect(screen.getByTestId(`agent-resource-edit-${sentryResource.id}`)).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-edit-resource-2")).not.toBeInTheDocument();
    vi.unstubAllGlobals();
  }, 10000);

  it("opens the form when a header server uses a catalog OAuth URL", async () => {
    const user = userEvent.setup();
    const customSentry = {
      ...HEADER_MCP_RESOURCE,
      id: "resource-custom-sentry",
      name: "custom-sentry",
      url: "https://mcp.sentry.dev/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [customSentry] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByTestId(`agent-resource-edit-${customSentry.id}`, {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-resource-connection-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-name")).toHaveValue("custom-sentry");
    expect(screen.getByTestId("agent-resource-auth")).toHaveTextContent("Header");
  }, 10000);

  it("keeps header auth when editing a header connection", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={mcpPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByTestId(`agent-resource-menu-${HEADER_MCP_RESOURCE.id}`, {}, { timeout: 8000 }));
    await user.click(screen.getByText("Edit"));
    expect(await screen.findByTestId("agent-resource-connection-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-auth")).toHaveTextContent("Header");
  }, 10000);
});
