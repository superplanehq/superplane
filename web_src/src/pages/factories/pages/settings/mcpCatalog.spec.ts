import { describe, expect, it } from "bun:test";

import { AGENT_RESOURCES_COPY, GITHUB_PERSONAL_ACCESS_TOKEN_URL } from "./agentResourceCopy";
import { catalogConnectionDefaults, filterMCPCatalog, groupMCPCatalog, MCP_CATALOG } from "./mcpCatalog";

describe("MCP_CATALOG", () => {
  it("includes a HTTPS URL and a SuperPlane-ready auth method on every entry", () => {
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
      expect(entry.instruction).toBeDefined();
    }
    const github = MCP_CATALOG.find((entry) => entry.id === "github");
    expect(github?.auth).toBe("AUTH_HEADERS");
    expect(github?.headerName).toBe("Authorization");
    expect(MCP_CATALOG.filter((entry) => entry.id !== "github").every((entry) => entry.auth === "AUTH_OAUTH")).toBe(
      true,
    );
  });

  it("prefills GitHub header auth and a token instruction", () => {
    const github = MCP_CATALOG.find((entry) => entry.id === "github");
    expect(catalogConnectionDefaults(github)).toEqual({
      name: "github",
      url: "https://api.githubcopilot.com/mcp/",
      auth: "AUTH_HEADERS",
      headers: [{ name: "Authorization" }],
      instruction: AGENT_RESOURCES_COPY.githubInstruction,
    });
    expect(AGENT_RESOURCES_COPY.githubInstruction).toEqual({
      before: "Create a ",
      href: GITHUB_PERSONAL_ACCESS_TOKEN_URL,
      label: "GitHub personal access token",
      after: " and paste it here.",
    });
  });

  it("omits defaults for a custom MCP server", () => {
    expect(catalogConnectionDefaults(undefined)).toBeUndefined();
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
