import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PropertiesTab } from "./PropertiesTab";

const installationURL = "https://github.com/organizations/superplanehq/settings/installations/123";

function renderPropertiesTab(integrationProperties: Parameters<typeof PropertiesTab>[0]["integrationProperties"]) {
  return render(
    <PropertiesTab
      integrationProperties={integrationProperties}
      propertyDrafts={Object.fromEntries(
        integrationProperties.map((property) => [property.name ?? "", property.value ?? ""]),
      )}
      setPropertyDrafts={vi.fn()}
      canUpdateIntegrations
      permissionsLoading={false}
      settingsMutationBusy={false}
      saveProperty={vi.fn().mockResolvedValue(undefined)}
      isSavingProperty={() => false}
    />,
  );
}

describe("PropertiesTab", () => {
  it("links GitHub App integrations to their repository access settings", () => {
    renderPropertiesTab([
      {
        name: "appInstallationURL",
        label: "GitHub App Installation URL",
        value: installationURL,
        editable: false,
      },
    ]);

    expect(screen.getByText("Can't find a repository?")).toBeInTheDocument();
    const link = screen.getByRole("link", { name: "Manage repository access" });
    expect(link).toHaveAttribute("href", installationURL);
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("does not show repository access guidance for integrations without an installation URL", () => {
    renderPropertiesTab([
      {
        name: "authMethod",
        label: "Authentication Method",
        value: "pat",
        editable: false,
      },
    ]);

    expect(screen.queryByText("Can't find a repository?")).not.toBeInTheDocument();
  });
});
