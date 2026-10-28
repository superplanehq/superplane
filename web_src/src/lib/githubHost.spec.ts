import { describe, expect, it } from "bun:test";

import {
  githubApiOrigin,
  githubHostFromHint,
  githubHostFromIntegration,
  githubProfileUrl,
  githubRepoWebUrl,
  githubWebHostFromUrl,
  githubWebOrigin,
  isGitHubEnterpriseServerHost,
  isGitHubWebHostname,
  isKnownGitHubHostname,
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

  it("treats an empty host property as github.com even when configuration has a tenant", () => {
    expect(
      githubHostFromIntegration({
        metadata: { integrationName: "github" },
        spec: { configuration: { host: "acme.ghe.com" } },
        status: { properties: [{ name: "host", value: "" }] },
      }),
    ).toBe("");
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
    expect(githubApiOrigin()).toBe("https://api.github.com");
    expect(githubRepoWebUrl("acme/api")).toBe("https://github.com/acme/api");
    expect(githubProfileUrl("alex")).toBe("https://github.com/alex");
    expect(githubWebHostFromUrl("https://github.com/acme/api/pull/12")).toBe("");
    expect(
      githubHostFromHint({
        html_url: "https://github.com/acme/app",
        url: "https://api.github.com/repos/acme/app",
      }),
    ).toBe("");
  });

  it("uses the tenant host", () => {
    expect(githubWebOrigin("acme.ghe.com")).toBe("https://acme.ghe.com");
    expect(githubApiOrigin("acme.ghe.com")).toBe("https://api.acme.ghe.com");
    expect(githubRepoWebUrl("acme/api", "acme.ghe.com")).toBe("https://acme.ghe.com/acme/api");
    expect(githubProfileUrl("alex", "acme.ghe.com")).toBe("https://acme.ghe.com/alex");
    expect(githubWebHostFromUrl("https://acme.ghe.com/acme/api/pull/12")).toBe("acme.ghe.com");
    expect(githubHostFromHint("https://acme.ghe.com/acme/app")).toBe("acme.ghe.com");
    expect(githubHostFromHint({ html_url: "https://acme.ghe.com/acme/app" })).toBe("acme.ghe.com");
    expect(githubHostFromHint("github.com")).toBe("");
  });
});

describe("GitHub host kind", () => {
  it("treats a ghe.com hostname as GitHub Enterprise Cloud", () => {
    expect(isGitHubEnterpriseServerHost("acme.ghe.com")).toBe(false);
    expect(isGitHubWebHostname("acme.ghe.com")).toBe(true);
    expect(isGitHubWebHostname("www.github.com")).toBe(true);
    expect(isGitHubWebHostname("api.acme.ghe.com")).toBe(false);
    expect(isKnownGitHubHostname("api.github.com")).toBe(true);
    expect(isKnownGitHubHostname("api.acme.ghe.com")).toBe(true);
    expect(isKnownGitHubHostname("gitlab.com")).toBe(false);
  });

  it("treats a self-hosted hostname as GitHub Enterprise Server", () => {
    expect(isGitHubEnterpriseServerHost("ghes.corp.example")).toBe(true);
    expect(isGitHubWebHostname("ghes.corp.example", "ghes.corp.example")).toBe(true);
    expect(isGitHubWebHostname("ghes.corp.example")).toBe(false);
    expect(githubApiOrigin("ghes.corp.example")).toBe("https://ghes.corp.example/api/v3");
  });
});
