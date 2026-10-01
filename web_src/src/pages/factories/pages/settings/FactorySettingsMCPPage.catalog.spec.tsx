import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { OAUTH_CONNECTED_RESOURCE, OAUTH_NOT_CONNECTED_RESOURCE } from "../../__fixtures__/agentResourceFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import {
  AGENT_RESOURCES_COPY,
  CIRCLECI_PERSONAL_API_TOKEN_URL,
  GITHUB_PERSONAL_ACCESS_TOKEN_URL,
  SEMAPHORE_API_TOKEN_URL,
} from "./agentResourceCopy";
import type { FactoriesFixture } from "../../__fixtures__/factoryPageResponses";

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent`;
const mcpAndSkills = [FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS];

function emptyAgentResourcesFixture(): FactoriesFixture {
  return {
    ...defaultFactoriesFixture,
    agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
  };
}

function stubLocationAssign(assign: ReturnType<typeof vi.fn>) {
  const { pathname, href, origin, search, hash } = window.location;
  vi.stubGlobal("location", { pathname, href, origin, search, hash, assign });
}

describe("FactorySettingsMCPPage catalog", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("opens the MCP catalog from the query string", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-search")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-github")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-circleci")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-gitlab")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-category-code")).toHaveTextContent("Code");
    expect(screen.getByTestId("mcp-catalog-category-observability")).toHaveTextContent("Observability");
    expect(screen.getByTestId("mcp-catalog-category-docs")).toHaveTextContent("Docs");
    expect(screen.getByTestId("mcp-catalog-category-database")).toHaveTextContent("Database");
    expect(screen.getByTestId("mcp-catalog-custom")).toHaveTextContent("Add custom");
  }, 10000);

  it("filters the catalog and keeps Add custom pinned", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
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
        pathSuffix={`${agentPath}?dialog=add`}
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

  it("opens CircleCI from the catalog with a token field", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-circleci"));

    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-add-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-connection-dialog")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-name")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-url")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-resource-auth")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-sign-in")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-icon")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-token")).toBeInTheDocument();
    expect(screen.getByLabelText("Personal access token")).toBeInTheDocument();

    const instruction = screen.getByTestId("mcp-catalog-setup-instruction");
    expect(instruction).toHaveTextContent("Create a CircleCI personal API token and paste it here.");
    expect(within(instruction).getByRole("link", { name: "CircleCI personal API token" })).toHaveAttribute(
      "href",
      CIRCLECI_PERSONAL_API_TOKEN_URL,
    );
    expect(
      within(instruction).getByRole("link", { name: "CircleCI personal API token" }).querySelector("svg"),
    ).not.toBeNull();
  }, 10000);

  it("opens Semaphore from the catalog with a token field", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-semaphore"));

    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("mcp-catalog-setup-sign-in")).not.toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-token")).toBeInTheDocument();

    const instruction = screen.getByTestId("mcp-catalog-setup-instruction");
    expect(instruction).toHaveTextContent(
      "Ask Semaphore support to enable MCP. Reset your Semaphore API token and paste it here.",
    );
    expect(within(instruction).getByRole("link", { name: "Semaphore API token" })).toHaveAttribute(
      "href",
      SEMAPHORE_API_TOKEN_URL,
    );
  }, 10000);

  it("opens Linear from the catalog with Sign in", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
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

  it("creates a Datadog server for the selected site", async () => {
    const assign = vi.fn();
    stubLocationAssign(assign);
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={emptyAgentResourcesFixture()}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-datadog"));

    expect(await screen.findByTestId("mcp-catalog-setup-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("mcp-catalog-setup-instruction")).toHaveTextContent("Sign in with Datadog.");
    expect(screen.getByTestId("mcp-catalog-setup-site")).toHaveTextContent("US1 (datadoghq.com)");
    expect(screen.getByText("Choose the site from your Datadog URL.")).toBeInTheDocument();

    await user.click(screen.getByTestId("mcp-catalog-setup-site"));
    await user.click(screen.getByRole("option", { name: "EU (datadoghq.eu)" }));
    await user.click(screen.getByTestId("mcp-catalog-setup-sign-in"));

    await waitFor(() => {
      expect(assign).toHaveBeenCalledWith("https://auth.example.com/authorize?client_id=storybook");
    });
    expect(screen.getByTestId("agent-resources-connections-list")).toHaveTextContent("Datadog");
    expect(screen.getByTestId("agent-resources-connections-list")).toHaveTextContent("Not connected");
  }, 10000);

  it("opens a saved Datadog server on its site", async () => {
    const user = userEvent.setup();
    const datadogResource = {
      ...OAUTH_NOT_CONNECTED_RESOURCE,
      id: "resource-datadog",
      name: "datadog",
      url: "https://mcp.us5.datadoghq.com/v1/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [datadogResource] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByTestId(`agent-resource-edit-${datadogResource.id}`, {}, { timeout: 8000 }));
    expect(await screen.findByTestId("mcp-connection-settings", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("https://mcp.us5.datadoghq.com/v1/mcp")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: AGENT_RESOURCES_COPY.edit }));
    expect(await screen.findByTestId("mcp-catalog-setup-site")).toHaveTextContent("US5 (us5.datadoghq.com)");
  }, 10000);

  it("opens Sentry from the catalog with Sign in", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
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
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={emptyAgentResourcesFixture()}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-github"));
    await user.type(await screen.findByTestId("mcp-catalog-setup-token"), "ghp_example");
    await user.click(screen.getByTestId("mcp-catalog-setup-save"));

    expect(await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 })).toHaveTextContent(
      "GitHub",
    );
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
  }, 10000);

  it("creates a CircleCI server from a pasted token", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={emptyAgentResourcesFixture()}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-circleci"));
    await user.type(await screen.findByTestId("mcp-catalog-setup-token"), "cci_example");
    await user.click(screen.getByTestId("mcp-catalog-setup-save"));

    expect(await screen.findByTestId("agent-resources-connections-list", {}, { timeout: 8000 })).toHaveTextContent(
      "CircleCI",
    );
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
  }, 10000);

  it("starts Linear sign-in from the catalog", async () => {
    const assign = vi.fn();
    stubLocationAssign(assign);
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={emptyAgentResourcesFixture()}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-linear"));
    await user.click(await screen.findByTestId("mcp-catalog-setup-sign-in"));

    await waitFor(() => {
      expect(assign).toHaveBeenCalledWith("https://auth.example.com/authorize?client_id=storybook");
    });
  }, 10000);

  it("opens Add custom without catalog instructions", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
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

  it("resumes Sign in from Add when the catalog OAuth server already exists", async () => {
    const assign = vi.fn();
    stubLocationAssign(assign);
    const user = userEvent.setup();
    const sentryResource = {
      ...OAUTH_NOT_CONNECTED_RESOURCE,
      id: "resource-sentry",
      name: "sentry",
      url: "https://mcp.sentry.dev/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
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
  }, 10000);

  it("does not open Sign in from Add when the catalog OAuth server is already connected", async () => {
    const user = userEvent.setup();
    const sentryResource = {
      ...OAUTH_CONNECTED_RESOURCE,
      id: "resource-sentry-connected",
      name: "sentry",
      url: "https://mcp.sentry.dev/mcp",
    };
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}?dialog=add`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [sentryResource] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await screen.findByTestId("mcp-add-picker", {}, { timeout: 8000 });
    await user.click(screen.getByTestId("mcp-catalog-sentry"));
    await waitFor(() => {
      expect(screen.queryByTestId("mcp-add-picker")).not.toBeInTheDocument();
    });
    expect(screen.queryByTestId("mcp-catalog-setup-dialog")).not.toBeInTheDocument();
    expect(screen.getByTestId(`agent-resource-edit-${sentryResource.id}`)).toBeInTheDocument();
  }, 10000);
});
