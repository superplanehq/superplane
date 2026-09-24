import { describe, expect, it } from "bun:test";

import { filterMCPCatalog, groupMCPCatalog, MCP_CATALOG } from "./mcpCatalog";

describe("filterMCPCatalog", () => {
  it("returns every entry when the query is empty", () => {
    expect(filterMCPCatalog(MCP_CATALOG, "  ")).toEqual(MCP_CATALOG);
  });

  it("matches label, name, id, or category", () => {
    expect(filterMCPCatalog(MCP_CATALOG, "GitHub").map((entry) => entry.id)).toEqual(["github"]);
    expect(filterMCPCatalog(MCP_CATALOG, "linear").map((entry) => entry.id)).toEqual(["linear"]);
    expect(filterMCPCatalog(MCP_CATALOG, "pager").map((entry) => entry.id)).toEqual(["pagerduty"]);
    expect(filterMCPCatalog(MCP_CATALOG, "observability").map((entry) => entry.id).sort()).toEqual([
      "datadog",
      "grafana",
      "honeycomb",
      "logfire",
      "newrelic",
      "sentry",
    ]);
  });
});

describe("groupMCPCatalog", () => {
  it("groups by category and sorts servers by name", () => {
    const groups = groupMCPCatalog(MCP_CATALOG);
    expect(groups.map((group) => group.id)).toEqual([
      "code",
      "issues",
      "chat",
      "cicd",
      "observability",
      "incident",
      "infrastructure",
    ]);
    expect(groups.find((group) => group.id === "code")?.entries.map((entry) => entry.label)).toEqual([
      "Bitbucket",
      "GitHub",
      "GitLab",
    ]);
    expect(groups.find((group) => group.id === "incident")?.entries.map((entry) => entry.label)).toEqual([
      "FireHydrant",
      "incident.io",
      "PagerDuty",
      "Rootly",
    ]);
  });

  it("omits categories that have no matching servers", () => {
    const github = MCP_CATALOG.find((entry) => entry.id === "github");
    expect(github).toBeDefined();
    expect(groupMCPCatalog([github!]).map((group) => group.id)).toEqual(["code"]);
  });
});
