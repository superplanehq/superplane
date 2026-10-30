import type { ReactNode } from "react";
import { act, render, screen, waitFor } from "@testing-library/react";
import { Outlet } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import App from "./App";

const access = vi.hoisted(() => ({
  factoriesEnabled: false,
  registryLoading: false,
  authenticated: true,
  deniedResource: "",
  useFactories: vi.fn(() => ({ data: [{ id: "factory-1", key: "demo" }], isLoading: false })),
}));

// Keep App's route tree and guards real; isolate account and server state.
vi.mock("@/contexts/AccountProvider", () => ({ AccountProvider: ({ children }: { children: ReactNode }) => children }));
vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: access.authenticated ? { id: "account-1" } : null, loading: false }),
}));
vi.mock("@/contexts/PermissionsProvider", () => ({
  PermissionsProvider: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    hasPermission: (resource: string) => resource !== access.deniedResource,
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({
    data: {
      metadata: { id: "org-123", slug: "org-123", name: "Demo" },
      spec: { enabledExperimentalFeatures: access.factoriesEnabled ? ["factories"] : [] },
    },
    isLoading: false,
    isSuccess: true,
  }),
}));
vi.mock("@/hooks/useExperimentalFeatures", () => ({
  useExperimentalFeaturesRegistry: () => ({
    data: access.registryLoading ? undefined : { features: [{ id: "factories", released: false }] },
    isLoading: access.registryLoading,
    isError: false,
  }),
}));
vi.mock("@/hooks/useFactoryData", () => ({ useFactories: access.useFactories }));
vi.mock("@/components/SetupGuard", () => ({ SetupGuard: ({ children }: { children: ReactNode }) => children }));
vi.mock("@/components/GlobalCommandPalette", () => ({ GlobalCommandPalette: () => null }));
vi.mock("@/components/SupportSessionBanner", () => ({ SupportSessionBanner: () => null }));
vi.mock("@/components/LicenseExpiryBanner", () => ({ default: () => null }));
vi.mock("@/hooks/usePageObservability", () => ({ usePageObservability: () => undefined }));
vi.mock("@/hooks/useIdentifyOrganizationGroup", () => ({ useIdentifyOrganizationGroup: () => undefined }));
vi.mock("@/hooks/usePersistOrganizationLastLocation", () => ({ usePersistOrganizationLastLocation: () => undefined }));
vi.mock("@/hooks/useRedirectIntegrationSetupReturn", () => ({ useRedirectIntegrationSetupReturn: () => undefined }));
vi.mock("@/hooks/useConsumeIntegrationSetupReturnOnArrival", () => ({
  useConsumeIntegrationSetupReturnOnArrival: () => undefined,
}));
vi.mock("@/hooks/useUserNotificationsWebsocket", () => ({ UserNotificationsListener: () => null }));

vi.mock("@/pages/home", () => ({ HomePage: () => <div>Organization home</div> }));
vi.mock("@/pages/auth/Login", () => ({ Login: () => <div>Login page</div> }));
vi.mock("@/pages/organization/settings/General", () => ({ General: () => <div>General content</div> }));
vi.mock("@/pages/organization/settings/Members", () => ({ Members: () => <div>Members content</div> }));
vi.mock("@/pages/organization/settings/Groups", () => ({ Groups: () => <div>Groups content</div> }));
vi.mock("@/pages/organization/settings/ApiKeys", () => ({ APIKeys: () => <div>API keys content</div> }));
vi.mock("@/pages/organization/settings/APIKeyDetail", () => ({
  APIKeyDetail: () => <div>API key detail content</div>,
}));
vi.mock("@/pages/organization/settings/components/IntegrationSetup", () => ({
  IntegrationSetup: () => <div>Integration setup content</div>,
}));
vi.mock("@/pages/factories/pages/settings/FactorySettingsLayout", () => ({ FactorySettingsLayout: () => <Outlet /> }));
vi.mock("@/pages/factories/pages/organizationSettings/spending-redesign/OrganizationSpendingExplorer", () => ({
  OrganizationSpendingExplorer: () => <div>Workspace spending content</div>,
}));

function visit(path: string) {
  window.history.replaceState(null, "", path);
  return render(<App />);
}

const legacySettingsCases = [
  ["", "General content"],
  ["/general", "General content"],
  ["/members", "Members content"],
  ["/api-keys", "API keys content"],
  ["/groups", "Groups content"],
  ["/api-keys/key-456", "API key detail content"],
  ["/integrations/github/setup", "Integration setup content"],
] as const;

describe("App legacy settings routes", () => {
  beforeEach(() => {
    access.factoriesEnabled = false;
    access.registryLoading = false;
    access.authenticated = true;
    access.deniedResource = "";
    access.useFactories.mockClear();
    window.localStorage.clear();
  });

  for (const [suffix, content] of legacySettingsCases) {
    it(`redirects settings${suffix} home when factories are off`, async () => {
      visit(`/org-123/settings${suffix}`);

      expect(await screen.findByText("Organization home")).toBeInTheDocument();
      expect(window.location.pathname).toBe("/org-123");
      expect(screen.queryByText(content)).not.toBeInTheDocument();
    });

    it(`keeps settings${suffix} available when factories are on`, async () => {
      access.factoriesEnabled = true;
      visit(`/org-123/settings${suffix}`);

      expect(await screen.findByText(content)).toBeInTheDocument();
      expect(window.location.pathname).toBe(`/org-123/settings${suffix || "/general"}`);
    });
  }

  it("blocks spending before workspace selection when factories are off", async () => {
    visit("/org-123/settings/llm-spend");

    expect(await screen.findByText("Organization home")).toBeInTheDocument();
    expect(window.location.pathname).toBe("/org-123");
    expect(access.useFactories).not.toHaveBeenCalled();
  });

  for (const suffix of ["/general", "/llm-spend"]) {
    it(`waits for feature access before mounting or redirecting settings${suffix}`, async () => {
      access.registryLoading = true;
      const view = visit(`/org-123/settings${suffix}`);

      expect(screen.getByText("Loading...")).toBeInTheDocument();
      expect(screen.queryByText("General content")).not.toBeInTheDocument();
      expect(access.useFactories).not.toHaveBeenCalled();
      expect(window.location.pathname).toBe(`/org-123/settings${suffix}`);

      access.registryLoading = false;
      view.rerender(<App />);

      expect(await screen.findByText("Organization home")).toBeInTheDocument();
      expect(window.location.pathname).toBe("/org-123");
    });
  }

  it("replaces the blocked settings history entry", async () => {
    window.history.replaceState(null, "", "/org-123?previous=1");
    window.history.pushState(null, "", "/org-123/settings/members");
    render(<App />);
    expect(await screen.findByText("Organization home")).toBeInTheDocument();

    await act(async () => window.history.back());
    await waitFor(() => expect(window.location.search).toBe("?previous=1"));
    expect(window.location.pathname).toBe("/org-123");
  });

  it("retains section permissions without requiring factories permission for legacy settings", async () => {
    access.factoriesEnabled = true;
    access.deniedResource = "factories";
    const view = visit("/org-123/settings/members");
    expect(await screen.findByText("Members content")).toBeInTheDocument();

    access.deniedResource = "members";
    view.rerender(<App />);
    expect(screen.queryByText("Members content")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Permission denied" })).toBeInTheDocument();
  });

  it("retains authentication before checking factories access", async () => {
    access.authenticated = false;
    visit("/org-123/settings/general");

    expect(await screen.findByText("Login page")).toBeInTheDocument();
    expect(window.location.pathname).toBe("/login");
    expect(new URLSearchParams(window.location.search).get("redirect")).toBe("/org-123/settings/general");
  });

  for (const path of [
    "/org-123/settings/llm-spend?range=month",
    "/org-123/workspaces/demo/settings/organization/spending?range=month",
  ]) {
    it(`retains the enabled workspace spending destination from ${path}`, async () => {
      access.factoriesEnabled = true;
      visit(path);

      expect(await screen.findByText("Workspace spending content")).toBeInTheDocument();
      expect(window.location.pathname).toBe("/org-123/workspaces/demo/settings/organization/spending");
      expect(window.location.search).toBe("?range=month");
    });
  }
});
