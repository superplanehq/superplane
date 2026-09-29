import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { IntegrationProperty, OrganizationsIntegration } from "@/api-client";
import { GITHUB_APP_INSTALLATION_URL_PROPERTY_NAME } from "@/lib/integrations";
import { catalogAppearance } from "./integrationCatalogAppearance";
import { CatalogInstanceRow } from "./CatalogInstanceRow";

const INSTALLATION_URL = "https://github.com/organizations/acme/settings/installations/123";

function buildIntegration(properties: IntegrationProperty[]): OrganizationsIntegration {
  return {
    metadata: { id: "integration-1", name: "github-prod", integrationName: "github" },
    status: { state: "ready", properties },
  };
}

function renderRow(integration: OrganizationsIntegration, onConfigure = vi.fn()) {
  return render(
    <CatalogInstanceRow
      appearance="factories"
      integration={integration}
      styles={catalogAppearance("factories")}
      canUpdateIntegrations
      permissionsLoading={false}
      onConfigure={onConfigure}
    />,
  );
}

describe("CatalogInstanceRow", () => {
  it("shows a link to manage the GitHub App installation when the property is present", () => {
    renderRow(buildIntegration([{ name: GITHUB_APP_INSTALLATION_URL_PROPERTY_NAME, value: INSTALLATION_URL }]));

    const link = screen.getByRole("link", { name: /manage installation/i });
    expect(link).toHaveAttribute("href", INSTALLATION_URL);
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("hides the installation link when the property is absent", () => {
    renderRow(buildIntegration([]));

    expect(screen.queryByRole("link", { name: /manage installation/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Configure" })).toBeInTheDocument();
  });

  it("keeps Configure working regardless of the installation link", async () => {
    const user = userEvent.setup();
    const onConfigure = vi.fn();

    renderRow(
      buildIntegration([{ name: GITHUB_APP_INSTALLATION_URL_PROPERTY_NAME, value: INSTALLATION_URL }]),
      onConfigure,
    );

    await user.click(screen.getByRole("button", { name: "Configure" }));
    expect(onConfigure).toHaveBeenCalledTimes(1);
  });
});
