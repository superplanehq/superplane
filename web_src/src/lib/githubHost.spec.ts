import { describe, expect, it } from "bun:test";

import {
  githubHostFromIntegration,
  githubRepoWebUrl,
  githubWebOrigin,
  isGitHubEnterpriseServerHost,
  isGitHubWebHostname,
} from "./githubHost";

describe("githubHostFromIntegration", () => {
  it("returns empty for github.com", () => {
    expect(
      githubHostFromIntegration({
        metadata: { integrationName: "github" },
        spec: { configuration: { host: "" } },
      }),
    ).toBe("");
  });

  it("prefers the wizard host property", () => {
    expect(
      githubHostFromIntegration({
        metadata: { integrationName: "github" },
        spec: { configuration: { host: "old.ghe.com" } },
        status: { properties: [{ name: "host", value: "acme.ghe.com" }] },
      }),
    ).toBe("acme.ghe.com");
  });

  it("ignores other integrations", () => {
    expect(
      githubHostFromIntegration({
        metadata: { integrationName: "bitbucket" },
        spec: { configuration: { host: "acme.ghe.com" } },
      }),
    ).toBe("");
  });
});

describe("github web URLs", () => {
  it("uses github.com when the host is empty", () => {
    expect(githubWebOrigin()).toBe("https://github.com");
    expect(githubRepoWebUrl("acme/api")).toBe("https://github.com/acme/api");
  });

  it("uses the tenant host", () => {
    expect(githubWebOrigin("acme.ghe.com")).toBe("https://acme.ghe.com");
    expect(githubRepoWebUrl("acme/api", "acme.ghe.com")).toBe("https://acme.ghe.com/acme/api");
  });
});

describe("GitHub host kind", () => {
  it("treats a ghe.com hostname as GitHub Enterprise Cloud", () => {
    expect(isGitHubEnterpriseServerHost("acme.ghe.com")).toBe(false);
    expect(isGitHubWebHostname("acme.ghe.com")).toBe(true);
  });

  it("treats a self-hosted hostname as GitHub Enterprise Server", () => {
    expect(isGitHubEnterpriseServerHost("ghes.corp.example")).toBe(true);
    expect(isGitHubWebHostname("ghes.corp.example", "ghes.corp.example")).toBe(true);
    expect(isGitHubWebHostname("ghes.corp.example")).toBe(false);
  });
});
