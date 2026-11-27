import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";

import { BrokenIntegrationsBanner } from "./BrokenIntegrationsBanner";
import type { BrokenIntegration } from "./lib/brokenIntegrations";

const githubBroken: BrokenIntegration = {
  id: "gh-1",
  name: "github-main",
  integrationName: "github",
  reason: "error",
  description: "App was uninstalled",
  actionLabel: "Reinstall app",
};

const openaiIncomplete: BrokenIntegration = {
  id: "oa-1",
  name: "openai-main",
  integrationName: "openai",
  reason: "incomplete",
  actionLabel: "Finish setup",
};

describe("BrokenIntegrationsBanner", () => {
  it("renders nothing when there are no broken integrations", () => {
    const { container } = render(
      <MemoryRouter>
        <BrokenIntegrationsBanner
          integrations={[]}
          integrationsBasePath="/org/workspaces/RF/settings/organization/integrations"
        />
      </MemoryRouter>,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("names each broken integration, its reason, and links to the repair step", () => {
    const jiraBroken: BrokenIntegration = {
      id: "jira-1",
      name: "Engineering Jira",
      integrationName: "jira",
      reason: "error",
      description: "Reconnect to Jira",
      actionLabel: "Reconnect",
    };
    const oldGitHub = { ...githubBroken, id: "gh-old", name: "github-old" };
    render(
      <MemoryRouter>
        <BrokenIntegrationsBanner
          integrations={[githubBroken, oldGitHub, openaiIncomplete, jiraBroken]}
          integrationsBasePath="/org/workspaces/RF/settings/organization/integrations"
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("broken-integrations-banner")).toHaveTextContent("App was uninstalled");
    for (const integration of [githubBroken, oldGitHub, jiraBroken]) {
      const row = screen.getAllByRole("listitem").find((item) => item.textContent?.includes(integration.name));
      expect(row).toBeDefined();
      const connection = within(row!);
      const provider = integration.integrationName === "jira" ? "Jira" : "GitHub";
      expect(connection.getByText(`${integration.name} (${provider})`)).toBeVisible();
      expect(connection.getByRole("link", { name: integration.actionLabel })).toHaveAttribute(
        "href",
        `/org/workspaces/RF/settings/organization/integrations/${integration.id}`,
      );
    }
    expect(screen.getByRole("link", { name: "Finish setup" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/integrations/oa-1",
    );
  });

  it("hides the repair action when the user cannot manage integrations", () => {
    render(
      <MemoryRouter>
        <BrokenIntegrationsBanner
          integrations={[githubBroken]}
          integrationsBasePath="/org/workspaces/RF/settings/organization/integrations"
          canManageIntegrations={false}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByRole("link", { name: "Reinstall app" })).not.toBeInTheDocument();
  });
});
