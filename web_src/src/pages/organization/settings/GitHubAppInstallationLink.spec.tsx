import type { OrganizationsIntegration } from "@/api-client";
import { TooltipProvider } from "@/ui/tooltip";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { Header } from "./components/CapabilityBasedIntegrationDetails/Header";
import { IntegrationCatalog } from "./IntegrationCatalog";

const installationURL = "https://github.com/organizations/acme/settings/installations/101";

const githubIntegration: OrganizationsIntegration = {
  metadata: { id: "gh-1", name: "github-acme", integrationName: "github" },
  status: {
    state: "ready",
    properties: [{ name: "appInstallationURL", label: "GitHub App Installation URL", value: installationURL }],
  },
};

const catalogInstances = vi.hoisted(() => ({
  current: [] as OrganizationsIntegration[],
}));

vi.mock("@/hooks/useIntegrationCatalog", () => ({
  useIntegrationCatalog: () => {
    const item = {
      providerName: "github",
      providerLabel: "GitHub",
      integrationDef: {
        name: "github",
        label: "GitHub",
        description: "Connect GitHub repositories.",
        hostedAppInstall: true,
        legacySetupOnly: false,
      },
      instances: catalogInstances.current,
    };
    return {
      isLoading: false,
      permissionsLoading: false,
      canCreateIntegrations: true,
      canUpdateIntegrations: true,
      filterQuery: "",
      setFilterQuery: vi.fn(),
      isIntegrationSurveyActive: false,
      isModalOpen: false,
      selectedIntegration: null,
      handleCloseModal: vi.fn(),
      handleConnectClick: vi.fn(),
      openInstance: vi.fn(),
      integrationCatalog: [item],
      filteredIntegrationCatalog: [item],
    };
  },
}));

function renderCatalog() {
  return render(
    <MemoryRouter>
      <TooltipProvider>
        <IntegrationCatalog organizationId="org-1" appearance="factories" />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("GitHub App installation link", () => {
  beforeEach(() => {
    catalogInstances.current = [githubIntegration];
  });

  it("links a GitHub integration to its installation", () => {
    renderCatalog();

    const link = screen.getByTestId("github-app-installation-link");
    expect(link).toHaveTextContent("GitHub App");
    expect(link).toHaveAttribute("href", installationURL);
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("omits the link when the integration has no installation URL", () => {
    catalogInstances.current = [
      {
        metadata: { id: "gh-2", name: "github-pat", integrationName: "github" },
        status: { state: "ready", properties: [] },
      },
    ];

    renderCatalog();

    expect(screen.queryByTestId("github-app-installation-link")).not.toBeInTheDocument();
  });

  it("shows the link on the integration page", () => {
    render(
      <MemoryRouter>
        <Header
          organizationId="org-1"
          integration={githubIntegration}
          canDeleteIntegrations
          permissionsLoading={false}
          onRequestDelete={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("github-app-installation-link")).toHaveAttribute("href", installationURL);
  });
});
