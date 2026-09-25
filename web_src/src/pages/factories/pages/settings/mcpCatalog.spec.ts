import { describe, expect, it } from "bun:test";

import { filterMCPCatalog, groupMCPCatalog, MCP_CATALOG } from "./mcpCatalog";

describe("MCP_CATALOG", () => {
  it("includes a HTTPS URL and sign-in auth on every entry", () => {
    expect(MCP_CATALOG.map((entry) => entry.id)).toEqual([
      "github",
      "jira",
      "linear",
      "circleci",
      "semaphore",
      "sentry",
    ]);
    for (const entry of MCP_CATALOG) {
      expect(entry.url.startsWith("https://")).toBe(true);
      expect(entry.auth).toBe("AUTH_OAUTH");
    }
  });
});

describe("filterMCPCatalog", () => {
  it("returns every entry when the query is empty", () => {
    expect(filterMCPCatalog(MCP_CATALOG, "  ")).toEqual(MCP_CATALOG);
  });

  it("matches label, name, id, or category", () => {
    expect(filterMCPCatalog(MCP_CATALOG, "GitHub").map((entry) => entry.id)).toEqual(["github"]);
    expect(filterMCPCatalog(MCP_CATALOG, "linear").map((entry) => entry.id)).toEqual(["linear"]);
    expect(filterMCPCatalog(MCP_CATALOG, "circle").map((entry) => entry.id)).toEqual(["circleci"]);
    expect(
      filterMCPCatalog(MCP_CATALOG, "observability")
        .map((entry) => entry.id)
        .sort(),
    ).toEqual(["sentry"]);
  });
});

describe("groupMCPCatalog", () => {
  it("groups by category and sorts servers by name", () => {
    const groups = groupMCPCatalog(MCP_CATALOG);
    expect(groups.map((group) => group.id)).toEqual(["code", "issues", "cicd", "observability"]);
    expect(groups.find((group) => group.id === "code")?.entries.map((entry) => entry.label)).toEqual(["GitHub"]);
    expect(groups.find((group) => group.id === "issues")?.entries.map((entry) => entry.label)).toEqual([
      "Jira",
      "Linear",
    ]);
    expect(groups.find((group) => group.id === "cicd")?.entries.map((entry) => entry.label)).toEqual([
      "CircleCI",
      "Semaphore",
    ]);
    expect(groups.find((group) => group.id === "observability")?.entries.map((entry) => entry.label)).toEqual([
      "Sentry",
    ]);
  });

  it("omits categories that have no matching servers", () => {
    const github = MCP_CATALOG.find((entry) => entry.id === "github");
    expect(github).toBeDefined();
    expect(groupMCPCatalog([github!]).map((group) => group.id)).toEqual(["code"]);
  });
});
