import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_AGENT_RESOURCES } from "@/lib/experimentalFeatures";
import { HEADER_MCP_RESOURCE, INLINE_SKILL, UI_UX_PRO_MAX_SKILL } from "../../__fixtures__/agentResourceFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";

const connectionsPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent-resources`;

describe("FactorySettingsAgentResourcesPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  it("hides the nav item when the feature is off", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-agent-resources")).not.toBeInTheDocument();
  }, 10000);

  it("shows the nav item when the feature is on", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-agent-resources")).toHaveTextContent(
      "Agent resources",
    );
  }, 10000);

  it("redirects away from the route when the feature is off", async () => {
    render(<FactoriesHarness pathSuffix={connectionsPath} factoriesFixture={defaultFactoriesFixture} />);

    await waitFor(() => {
      expect(screen.queryByTestId("factory-settings-sidebar")).not.toBeInTheDocument();
    });
    expect(screen.queryByTestId("factory-settings-agent-resources")).not.toBeInTheDocument();
  }, 10000);

  it("shows the MCP servers empty state", async () => {
    render(
      <FactoriesHarness
        pathSuffix={connectionsPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    expect(await screen.findByTestId("workspace-page-header-title", {}, { timeout: 8000 })).toHaveTextContent(
      "Agent resources",
    );
    expect(screen.getByTestId("agent-resources-tab-connections")).toHaveTextContent("MCP servers");
    expect(await screen.findByTestId("agent-resources-connections-empty")).toHaveTextContent("No MCP servers yet.");
    expect(screen.getByTestId("agent-resources-add-connection")).toHaveTextContent("Add MCP server");
  }, 10000);

  it("lists a header MCP server", async () => {
    render(
      <FactoriesHarness
        pathSuffix={connectionsPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
        }}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    expect(await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("docs")).toBeInTheDocument();
    expect(screen.getByText("https://mcp.example.com/mcp")).toBeInTheDocument();
    expect(screen.getByText("Header")).toBeInTheDocument();
  }, 10000);

  it("opens the add MCP server dialog from the query string", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${connectionsPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    expect(await screen.findByTestId("agent-resource-connection-dialog", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-name")).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-url")).toBeInTheDocument();
  }, 10000);

  it("shows the skills empty state", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={connectionsPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    await screen.findByTestId("factory-settings-agent-resources", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("agent-resources-tab-skills"));
    expect(await screen.findByTestId("agent-resources-skills-empty")).toHaveTextContent("No skills yet.");
    expect(screen.getByTestId("agent-resources-add-skill")).toBeEnabled();
    expect(screen.getByTestId("agent-resources-add-skill")).toHaveTextContent("Add skill");
  }, 10000);

  it("opens the add skill dialog from the query string", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${connectionsPath}?tab=skills&dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    expect(await screen.findByTestId("agent-resource-skill-dialog", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-skill-name")).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-skill-markdown")).toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
  }, 10000);

  it("lists an inline skill", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${connectionsPath}?tab=skills`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [INLINE_SKILL] },
        }}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    expect(await screen.findByTestId("agent-resources-skills-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("review-copy")).toBeInTheDocument();
    expect(screen.getByText("SKILL.md")).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
  }, 10000);

  it("lists a GitHub skill package", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${connectionsPath}?tab=skills`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [UI_UX_PRO_MAX_SKILL] },
        }}
        experimentalFeatures={[FEATURE_WORKSPACE_AGENT_RESOURCES]}
      />,
    );

    expect(await screen.findByTestId("agent-resources-skills-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("ui-ux-pro-max")).toBeInTheDocument();
    expect(screen.getByText("nextlevelbuilder/ui-ux-pro-max-skill@v1.2.0")).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
  }, 10000);
});
