import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { posthog } from "@/posthog";

const menuAccess = vi.hoisted(() => ({ installationAdmin: false, deniedResource: "", featureLoading: false }));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: {
      id: "user-1",
      name: "Ada Lovelace",
      email: "ada@example.com",
      installation_admin: menuAccess.installationAdmin,
    },
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: null }),
}));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({
    data: [{ id: "org-456", slug: "other", name: "Other organization" }],
    refetch: vi.fn(),
  }),
}));

const experimentalFeatureMocks = vi.hoisted(() => ({
  has: vi.fn((_featureId: string) => false),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: experimentalFeatureMocks.has,
    enabledExperimentalFeatures: [],
    isLoading: menuAccess.featureLoading,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    hasPermission: (resource: string) => resource !== menuAccess.deniedResource,
    canAct: () => true,
    isLoading: false,
  }),
}));

vi.mock("@/posthog", () => ({
  posthog: { reset: vi.fn() },
}));

describe("OrganizationMenuButton", () => {
  beforeEach(() => {
    experimentalFeatureMocks.has.mockReset();
    experimentalFeatureMocks.has.mockReturnValue(false);
    menuAccess.installationAdmin = false;
    menuAccess.deniedResource = "";
    menuAccess.featureLoading = false;
    window.localStorage.clear();
    vi.mocked(posthog.reset).mockClear();
  });

  it("links the logo to organization selection when no organization is active", () => {
    render(
      <MemoryRouter>
        <OrganizationMenuButton />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/");
  });

  it("links the logo to the active organization when one is active", () => {
    render(
      <MemoryRouter>
        <OrganizationMenuButton organizationId="org-123" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/org-123");
  });

  it("links the logo to workspaces when factories are on", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === "factories");
    render(
      <MemoryRouter>
        <OrganizationMenuButton organizationId="org-123" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Go to canvases" })).toHaveAttribute("href", "/org-123/workspaces");
  });

  function openMenu() {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <OrganizationMenuButton organizationId="org-123" />
        </MemoryRouter>
      </ThemeProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Open organization menu" }));
  }

  for (const loading of [false, true]) {
    it(`hides all legacy settings links while factories access is ${loading ? "unresolved" : "off"}`, () => {
      menuAccess.featureLoading = loading;
      openMenu();

      const settingsLinks = screen
        .getAllByRole("link")
        .filter((link) => link.getAttribute("href")?.startsWith("/org-123/settings"));
      expect(settingsLinks).toHaveLength(0);
      for (const label of [
        "Settings",
        "Members",
        "API Keys",
        "Groups",
        "Roles",
        "Integrations",
        "Secrets",
        "Profile",
      ]) {
        expect(screen.queryByText(label)).not.toBeInTheDocument();
      }
    });
  }

  it("retains enabled settings destinations", () => {
    experimentalFeatureMocks.has.mockImplementation((featureId) => featureId === "factories");
    openMenu();

    for (const [label, section] of [
      ["Settings", "general"],
      ["Members", "members"],
      ["API Keys", "api-keys"],
      ["Groups", "groups"],
      ["Roles", "roles"],
      ["Integrations", "integrations"],
      ["Secrets", "secrets"],
      ["Profile", "profile"],
    ]) {
      expect(screen.getByRole("link", { name: label })).toHaveAttribute("href", `/org-123/settings/${section}`);
    }
    expect(screen.getByRole("link", { name: "Workspaces" })).toHaveAttribute("href", "/org-123/workspaces");
  });

  it("retains permission checks for enabled settings links", () => {
    experimentalFeatureMocks.has.mockReturnValue(true);
    menuAccess.deniedResource = "members";
    openMenu();

    expect(screen.queryByRole("link", { name: "Members" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Members" })).toBeDisabled();
    expect(screen.getByRole("link", { name: "Settings" })).toBeInTheDocument();
  });

  it("keeps organization switching, theme changes, and sign-out usable when factories are off", () => {
    openMenu();
    fireEvent.click(screen.getByRole("button", { name: "Switch organization" }));
    expect(screen.getByRole("link", { name: "Other organization" })).toHaveAttribute("href", "/other");
    expect(screen.getByRole("link", { name: "Create new organization" })).toHaveAttribute("href", "/onboarding");

    for (const label of ["Dark", "Light", "System"]) {
      fireEvent.click(screen.getByRole("button", { name: label }));
      expect(screen.getByRole("button", { name: label })).toHaveAttribute("aria-pressed", "true");
    }

    fireEvent.click(screen.getByRole("button", { name: "Sign Out" }));
    expect(posthog.reset).toHaveBeenCalledTimes(1);
    expect(window.location.pathname).toBe("/logout");
    expect(screen.queryByRole("button", { name: "Sign Out" })).not.toBeInTheDocument();
  });

  for (const installationAdmin of [false, true]) {
    it(`shows installation administration only for an administrator (${installationAdmin})`, () => {
      menuAccess.installationAdmin = installationAdmin;
      openMenu();

      if (installationAdmin) {
        expect(screen.getByRole("link", { name: "Installation Admin" })).toHaveAttribute("href", "/admin");
        return;
      }
      expect(screen.queryByRole("link", { name: "Installation Admin" })).not.toBeInTheDocument();
    });
  }
});
