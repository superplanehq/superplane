import { beforeEach, describe, expect, it, vi } from "vitest";

import { peekIntegrationSetupReturn } from "./integrationSetupReturn";
import {
  githubPrivateAppSetupPath,
  privateGitHubAppCreateConfiguration,
  startPrivateGitHubAppSetup,
} from "./privateGitHubApp";

describe("privateGitHubAppCreateConfiguration", () => {
  it("sends privateApp for GitHub setup create", () => {
    expect(privateGitHubAppCreateConfiguration("github")).toEqual({ privateApp: true });
  });

  it("does not send privateApp for other integrations", () => {
    expect(privateGitHubAppCreateConfiguration("slack")).toBeUndefined();
  });
});

describe("githubPrivateAppSetupPath", () => {
  it("opens the GitHub setup wizard", () => {
    expect(githubPrivateAppSetupPath("org-1")).toBe("/org-1/settings/integrations/github/setup");
  });
});

describe("startPrivateGitHubAppSetup", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stores the return path and opens the setup wizard", () => {
    const goTo = vi.fn();

    startPrivateGitHubAppSetup({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/ws/setup?step=vcs&pick=newest",
      goTo,
    });

    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/github/setup");
    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/ws/setup?step=vcs&pick=newest");
  });
});
