import { describe, expect, it, vi } from "bun:test";

import type { OrganizationsIntegration } from "@/api-client";

import { pendingGitHubAccountPicker, startDirectGitHubConnect } from "./startDirectGitHubConnect";

const follow = vi.hoisted(() => vi.fn(() => true));
vi.mock("@/lib/browserAction", () => ({ followBrowserAction: follow }));
vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: vi.fn(),
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
  isOnboardingSetupReturnPath: (path?: string) => path?.split("?")[0] === "/onboarding",
}));

const activeDiscovery: OrganizationsIntegration = {
  metadata: { id: "int-1", integrationName: "github" },
  status: {
    state: "pending",
    metadata: {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      installationDiscovery: { active: true, complete: false },
    },
  },
};

describe("active GitHub installation discovery", () => {
  it("opens the account picker before the first account is available", () => {
    expect(pendingGitHubAccountPicker([activeDiscovery], "user-1")).toEqual({
      id: "int-1",
      state: "csrf",
      appSlug: "superplane",
      githubLogin: "",
      discoveringAccounts: true,
      installations: [],
    });
  });

  it("keeps the picker loading while a callback installation becomes visible", () => {
    const pendingCallback: OrganizationsIntegration = {
      ...activeDiscovery,
      status: {
        ...activeDiscovery.status,
        metadata: {
          ...activeDiscovery.status?.metadata,
          installationDiscovery: { active: false, complete: true },
          setupInstallationId: "22",
        },
      },
    };

    expect(pendingGitHubAccountPicker([pendingCallback], "user-1")).toMatchObject({
      id: "int-1",
      discoveringAccounts: true,
      installations: [],
    });
  });

  it("offers manual installation after a bounded scan cannot finish", () => {
    const incompleteDiscovery: OrganizationsIntegration = {
      ...activeDiscovery,
      status: {
        ...activeDiscovery.status,
        metadata: {
          ...activeDiscovery.status?.metadata,
          installationDiscovery: { active: true, complete: false, installAvailable: true },
        },
      },
    };

    expect(pendingGitHubAccountPicker([incompleteDiscovery], "user-1")).toMatchObject({
      id: "int-1",
      discoveringAccounts: true,
      installAvailable: true,
    });
  });

  it("stays inside onboarding instead of opening GitHub", async () => {
    const create = vi.fn();
    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?step=vcs",
      existingNames: new Set(),
      connected: [activeDiscovery],
      currentUserId: "user-1",
      create,
    });

    expect(started).toBe(false);
    expect(create).not.toHaveBeenCalled();
    expect(follow).not.toHaveBeenCalled();
  });
});
