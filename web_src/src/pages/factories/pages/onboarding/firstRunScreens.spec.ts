import { describe, expect, it } from "bun:test";

import { screenWithVcsConnection, type GitHubScreenState } from "./firstRunScreens";

const github: GitHubScreenState = {
  pending: false,
  identity: false,
  ready: false,
  stayOnConnect: false,
  appConfigured: false,
  accountConnectionRequired: true,
};

describe("screenWithVcsConnection GitHub Enterprise", () => {
  it("keeps the hostname screen until the GitHub App exists", () => {
    expect(screenWithVcsConnection("ghe-host", "github", github, true)).toBe("ghe-host");
    expect(screenWithVcsConnection("choose", "github", github, true)).toBe("ghe-host");
  });

  it("opens repository choice after the GitHub App is created", () => {
    const connected = { ...github, ready: true, appConfigured: true, accountConnectionRequired: false };
    expect(screenWithVcsConnection("welcome", "github", connected, true)).toBe("choose");
    expect(screenWithVcsConnection("connect", "github", connected, true)).toBe("choose");
  });

  it("keeps an explicit hostname screen so Back can correct the host", () => {
    const connected = { ...github, ready: true, appConfigured: true, accountConnectionRequired: false };
    expect(screenWithVcsConnection("ghe-host", "github", connected, true)).toBe("ghe-host");
  });
});
