import { afterEach, describe, expect, it, vi } from "vitest";

import { peekIntegrationSetupReturn } from "@/lib/integrationSetupReturn";

import { useHomeIntegrationConnectActions } from "./useHomeIntegrationConnectActions";

describe("useHomeIntegrationConnectActions", () => {
  afterEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it("returns capability setup to the onboarding repository flow", () => {
    const open = vi.spyOn(window, "open").mockImplementation(() => null);
    const returnTo = "/org-1/workspaces/APP/setup?step=vcs&pick=newest";
    const actions = useHomeIntegrationConnectActions({
      organizationId: "org-1",
      returnTo,
      availableIntegrations: [{ name: "github", legacySetupOnly: false }],
      connected: [],
      pendingConnectKeyRef: { current: null },
      setDialogMode: vi.fn(),
      setDialogIntegrationName: vi.fn(),
      setConfigureIntegrationId: vi.fn(),
    });

    actions.openConnectDialog("github");

    expect(peekIntegrationSetupReturn("org-1")).toBe(returnTo);
    expect(open).toHaveBeenCalledWith("/org-1/settings/integrations/github/setup", "_blank", "noopener,noreferrer");
  });
});
