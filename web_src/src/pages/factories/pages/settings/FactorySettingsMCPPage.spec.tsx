import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import {
  FEATURE_SUPERPLANE_MCP_SERVER,
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";
import {
  HEADER_MCP_RESOURCE,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
} from "../../__fixtures__/agentResourceFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent`;
const superplaneMcpServerPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/superplane-mcp-server`;
const mcpConfigurePath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/mcp`;
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
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-agent")).not.toBeInTheDocument();
    expect(
      within(sidebar).queryByTestId("factory-settings-nav-workspace-superplane-mcp-server"),
    ).not.toBeInTheDocument();
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
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-agent")).toHaveTextContent("Agent");
    expect(
      within(sidebar).queryByTestId("factory-settings-nav-workspace-superplane-mcp-server"),
    ).not.toBeInTheDocument();
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
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-agent")).toHaveTextContent("Agent");
    expect(
      within(sidebar).queryByTestId("factory-settings-nav-workspace-superplane-mcp-server"),
    ).not.toBeInTheDocument();
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
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-agent")).toHaveTextContent("Agent");
    expect(
      within(sidebar).queryByTestId("factory-settings-nav-workspace-superplane-mcp-server"),
    ).not.toBeInTheDocument();
  }, 10000);

  it("redirects away from the route when the feature is off", async () => {
    render(<FactoriesHarness pathSuffix={agentPath} factoriesFixture={defaultFactoriesFixture} />);

    await waitFor(() => {
      expect(screen.queryByTestId("factory-settings-sidebar")).not.toBeInTheDocument();
    });
    expect(screen.queryByTestId("factory-settings-agent")).not.toBeInTheDocument();
  }, 10000);

  it("redirects the legacy agent-resources route to MCP", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent-resources`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("factory-settings-agent", {}, { timeout: 8000 })).toBeInTheDocument();
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
        pathSuffix={agentPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("workspace-page-header-title", {}, { timeout: 8000 })).toHaveTextContent("Agent");
    expect(await screen.findByTestId("agent-resources-connections-empty")).toHaveTextContent("No MCP servers yet");
    expect(screen.getByTestId("agent-resources-add-connection")).toHaveTextContent("Connect");
  }, 10000);

  it("lists a header MCP server with a connected status", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("Docs")).toBeInTheDocument();
    expect(screen.getByTestId(`agent-resource-status-${HEADER_MCP_RESOURCE.id}`)).toHaveTextContent("Connected");
    expect(screen.getByTestId(`agent-resource-tools-${HEADER_MCP_RESOURCE.id}`)).toHaveTextContent("2/2 tools");
    expect(screen.queryByText("Header")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Configure")).toBeInTheDocument();
  }, 10000);

  it("renders each server as its own compact card", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: {
            [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE, OAUTH_NOT_CONNECTED_RESOURCE],
          },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    const list = await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 });
    const docsCard = screen.getByTestId(`agent-resource-row-${HEADER_MCP_RESOURCE.id}`);
    const mobbinCard = screen.getByTestId(`agent-resource-row-${OAUTH_NOT_CONNECTED_RESOURCE.id}`);
    expect(list.tagName).toBe("UL");
    expect(docsCard).not.toBe(mobbinCard);
    expect(within(docsCard).getByText("Docs")).toBeInTheDocument();
    expect(within(docsCard).getByTestId(`agent-resource-status-${HEADER_MCP_RESOURCE.id}`)).toHaveTextContent(
      "Connected",
    );
    expect(within(docsCard).getByTestId(`agent-resource-tools-${HEADER_MCP_RESOURCE.id}`)).toHaveTextContent(
      "2/2 tools",
    );
    expect(within(docsCard).queryByText(/mcp\.example\.com/)).not.toBeInTheDocument();
    expect(within(mobbinCard).getByText("Not connected")).toBeInTheDocument();
    expect(within(mobbinCard).queryByText(/api\.mobbin\.com/)).not.toBeInTheDocument();
    expect(within(docsCard).getByTestId(`agent-resource-configure-${HEADER_MCP_RESOURCE.id}`)).toBeInTheDocument();
  }, 10000);

  it("shows tools on the configure page without descriptions", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${mcpConfigurePath}/${HEADER_MCP_RESOURCE.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("mcp-connection-tools", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(await screen.findByTestId("mcp-tools-list")).toHaveTextContent("search");
    expect(screen.queryByText("Search the catalog.")).not.toBeInTheDocument();
  }, 10000);

  it("hides tools when sign-in is not connected", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${mcpConfigurePath}/${OAUTH_NOT_CONNECTED_RESOURCE.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [OAUTH_NOT_CONNECTED_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("mcp-connection-settings", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-connection-tools")).not.toBeInTheDocument();
  }, 10000);

  it("opens edit from the configure page", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpConfigurePath}/${HEADER_MCP_RESOURCE.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByText("Edit", {}, { timeout: 8000 }));
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
        pathSuffix={`${mcpConfigurePath}/${sentryResource.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [sentryResource] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByText("Edit", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-name")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-sign-in")).toHaveTextContent("Sign in");
  }, 10000);

  it("opens Sign in when a connected catalog OAuth server is edited", async () => {
    const user = userEvent.setup();
    const sentryResource = {
      ...OAUTH_CONNECTED_RESOURCE,
      id: "resource-sentry-connected",
      name: "sentry",
      url: "https://mcp.sentry.dev/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={`${mcpConfigurePath}/${sentryResource.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [sentryResource] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByText("Edit", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-sign-in")).toHaveTextContent("Sign in");
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
        pathSuffix={`${mcpConfigurePath}/${customSentry.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [customSentry] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByText("Edit", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-resource-connection-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-name")).toHaveValue("custom-sentry");
    expect(screen.getByTestId("agent-resource-auth")).toHaveTextContent("Header");
  }, 10000);

  it("keeps header auth when editing a header connection", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${mcpConfigurePath}/${HEADER_MCP_RESOURCE.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByText("Edit", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-resource-connection-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-auth")).toHaveTextContent("Header");
  }, 10000);

  it("shows Connect nav when only SuperPlane MCP Server is on", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-superplane-mcp-server")).toHaveTextContent(
      "MCP Server",
    );
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-agent")).not.toBeInTheDocument();
  }, 10000);

  it("hides inbound MCP on the Agent page", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("factory-settings-agent", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("superplane-mcp-server")).not.toBeInTheDocument();
  }, 10000);

  it("shows how to connect when no SuperPlane MCP clients exist", async () => {
    render(
      <FactoriesHarness
        pathSuffix={superplaneMcpServerPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    expect(
      await screen.findByText(
        "Connect from Cursor, Claude, Codex, OpenCode, or any other external service.",
        {},
        { timeout: 8000 },
      ),
    ).toBeInTheDocument();
    expect(await screen.findByTestId("superplane-mcp-clients-empty", {}, { timeout: 8000 })).toHaveTextContent(
      "No clients connected yet",
    );
    await userEvent.setup().click(screen.getByTestId("superplane-mcp-connect-client"));
    expect(await screen.findByTestId("superplane-mcp-connect-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("superplane-mcp-server-url")).toHaveTextContent("/mcp");
    expect(screen.getByTestId("superplane-mcp-client-tools")).toHaveTextContent("Cursor");
    expect(screen.getByTestId("superplane-mcp-client-tools")).toHaveTextContent("Claude Code");
    expect(screen.getByTestId("superplane-mcp-client-tools")).toHaveTextContent("VS Code");
    expect(screen.getByTestId("superplane-mcp-client-tools")).toHaveTextContent("Codex");
    expect(screen.getByTestId("superplane-mcp-client-tools")).toHaveTextContent("OpenCode");
    expect(screen.getByTestId("superplane-mcp-config-copy-cursor")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resources-add-connection")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resources-connections-empty")).not.toBeInTheDocument();
  }, 10000);

  it("shows Claude Code and VS Code setup after the matching tab", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={superplaneMcpServerPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    expect(await screen.findByTestId("superplane-mcp-clients-empty", {}, { timeout: 8000 })).toBeInTheDocument();
    await user.click(screen.getByTestId("superplane-mcp-connect-client"));
    expect(await screen.findByTestId("superplane-mcp-connect-dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("superplane-mcp-client-tool-claudeCode"));
    expect(screen.getByText("Run this command in a terminal.")).toBeInTheDocument();
    expect(screen.getByTestId("superplane-mcp-config-copy-claudeCode")).toBeInTheDocument();

    await user.click(screen.getByTestId("superplane-mcp-client-tool-vscode"));
    expect(screen.getByText("Open MCP settings in VS Code.")).toBeInTheDocument();
    expect(screen.getByTestId("superplane-mcp-config-copy-vscode")).toBeInTheDocument();

    await user.click(screen.getByTestId("superplane-mcp-client-tool-codex"));
    expect(screen.getByText("Open Codex CLI or the IDE extension.")).toBeInTheDocument();
    const copyButton = screen.getByTestId("superplane-mcp-config-copy-codex");
    expect(copyButton).toHaveAttribute("aria-label", "Copy command");

    const writeText = vi.fn(() => Promise.resolve());
    const previousClipboard = navigator.clipboard;
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    try {
      await user.click(copyButton);
      expect(writeText).toHaveBeenCalledWith(`codex mcp add superplane --url ${window.location.origin}/mcp`);
    } finally {
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: previousClipboard,
      });
    }
  }, 10000);

  it("shows OpenCode setup and copies the configuration", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={superplaneMcpServerPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    expect(await screen.findByTestId("superplane-mcp-clients-empty", {}, { timeout: 8000 })).toBeInTheDocument();
    await user.click(screen.getByTestId("superplane-mcp-connect-client"));
    expect(await screen.findByTestId("superplane-mcp-connect-dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("superplane-mcp-client-tool-opencode"));
    expect(screen.getByText("Open opencode.json.")).toBeInTheDocument();
    expect(screen.getByText("Add the superplane server under mcp. Keep other servers.")).toBeInTheDocument();
    expect(screen.getByText("Run opencode mcp auth superplane and sign in.")).toBeInTheDocument();
    const copyButton = screen.getByTestId("superplane-mcp-config-copy-opencode");
    expect(copyButton).toHaveAttribute("aria-label", "Copy MCP configuration");

    const writeText = vi.fn(() => Promise.resolve());
    const previousClipboard = navigator.clipboard;
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    try {
      await user.click(copyButton);
      expect(writeText).toHaveBeenCalledWith(
        JSON.stringify(
          {
            mcp: {
              superplane: {
                type: "remote",
                url: `${window.location.origin}/mcp`,
                enabled: true,
              },
            },
          },
          null,
          2,
        ),
      );
    } finally {
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: previousClipboard,
      });
    }
  }, 10000);

  it("lists a connected SuperPlane MCP client and revokes it", async () => {
    const user = userEvent.setup();
    const clientId = "mcp-client-cursor";
    render(
      <FactoriesHarness
        pathSuffix={superplaneMcpServerPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          mcpClientsByFactoryId: {
            [PRIMARY_FACTORY_ID]: [
              {
                id: clientId,
                clientName: "Cursor",
                userId: STORYBOOK_ME_USER_ID,
                userName: STORYBOOK_ME_USER_NAME,
                userAvatarUrl: "/storybook/leonardo-dicaprio.jpg",
                createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
              },
            ],
          },
        }}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    expect(await screen.findByTestId("superplane-mcp-clients-list", {}, { timeout: 8000 })).toHaveTextContent("Cursor");
    expect(screen.getByTestId(`superplane-mcp-client-${clientId}`)).toHaveTextContent(STORYBOOK_ME_USER_NAME);

    await user.click(screen.getByTestId(`superplane-mcp-client-revoke-${clientId}`));
    expect(screen.getByText(`Revoke "Cursor"?`)).toBeInTheDocument();
    await user.click(screen.getByTestId("factory-delete-confirm-button"));
    await waitFor(() => {
      expect(screen.queryByTestId("superplane-mcp-clients-list")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("superplane-mcp-clients-empty")).toBeInTheDocument();
  }, 10000);
});
