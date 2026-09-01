import { afterEach, describe, expect, it, vi } from "vitest";

import { requestGitHubInstallationConfigure } from "./useOnboardingPageModel";

describe("requestGitHubInstallationConfigure", () => {
  it("remembers the return path and navigates to the GitHub installation URL in the same tab", () => {
    const remember = vi.fn();
    const navigate = vi.fn();

    requestGitHubInstallationConfigure({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/PAY/setup?step=repo",
      integration: {
        spec: { configuration: { organization: "acme" } },
        status: { metadata: { installationId: "42" } },
      },
      remember,
      navigate,
    });

    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/workspaces/PAY/setup?step=repo");
    expect(navigate).toHaveBeenCalledWith("https://github.com/organizations/acme/settings/installations/42");
  });

  it("falls back to the installations list when the integration has no installation yet", () => {
    const remember = vi.fn();
    const navigate = vi.fn();

    requestGitHubInstallationConfigure({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/PAY/setup?step=repo",
      integration: null,
      remember,
      navigate,
    });

    expect(navigate).toHaveBeenCalledWith("https://github.com/settings/installations");
  });

  it("defaults to window.location.assign, not a new tab, so GitHub's redirect returns to this window", () => {
    const remember = vi.fn();
    const assign = vi.fn();
    const openSpy = vi.spyOn(window, "open");
    vi.stubGlobal("location", { assign });

    requestGitHubInstallationConfigure({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/PAY/setup?step=repo",
      integration: { status: { metadata: { installationId: "42" } } },
      remember,
    });

    expect(assign).toHaveBeenCalledWith("https://github.com/settings/installations/42");
    expect(openSpy).not.toHaveBeenCalled();

    vi.unstubAllGlobals();
    openSpy.mockRestore();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });
});
