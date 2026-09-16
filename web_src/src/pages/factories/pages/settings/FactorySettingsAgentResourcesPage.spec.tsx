import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_AGENT_RESOURCES } from "@/lib/experimentalFeatures";
import { HEADER_MCP_RESOURCE, UI_UX_PRO_MAX_SKILL } from "../../__fixtures__/agentResourceFixtures";
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

  it("shows the connections empty state", async () => {
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
    expect(await screen.findByTestId("agent-resources-connections-empty")).toHaveTextContent("No MCP connections yet.");
    expect(screen.getByTestId("agent-resources-add-connection")).toHaveTextContent("Add connection");
  }, 10000);

  it("lists a header connection", async () => {
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

  it("opens the add connection dialog from the query string", async () => {
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
    expect(await screen.findByTestId("agent-resources-skills-empty")).toHaveTextContent(
      "Skills are not available yet.",
    );
    expect(screen.getByTestId("agent-resources-add-skill")).toBeDisabled();
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
