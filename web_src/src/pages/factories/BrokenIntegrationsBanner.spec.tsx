import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

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
    render(
      <MemoryRouter>
        <BrokenIntegrationsBanner
          integrations={[githubBroken, openaiIncomplete]}
          integrationsBasePath="/org/workspaces/RF/settings/organization/integrations"
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("broken-integrations-banner")).toHaveTextContent("2 integrations need attention");
    expect(screen.getByTestId("broken-integrations-banner")).toHaveTextContent("App was uninstalled");
    expect(screen.getByRole("link", { name: "Reinstall app" })).toHaveAttribute(
      "href",
      "/org/workspaces/RF/settings/organization/integrations/gh-1",
    );
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
