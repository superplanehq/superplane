import { describe, expect, it } from "bun:test";

import {
  AGENT_RESOURCES_COPY,
  CIRCLECI_PERSONAL_API_TOKEN_URL,
  GITHUB_PERSONAL_ACCESS_TOKEN_URL,
  SEMAPHORE_API_TOKEN_URL,
} from "./agentResourceCopy";
import {
  catalogConnectionDefaults,
  catalogEntryForResource,
  catalogEntryIsConnected,
  catalogOAuthResourceForEntry,
  filterMCPCatalog,
  groupMCPCatalog,
  MCP_CATALOG,
} from "./mcpCatalog";

describe("MCP_CATALOG", () => {
  it("includes a HTTPS URL and a SuperPlane-ready auth method on every entry", () => {
    expect(MCP_CATALOG.map((entry) => entry.id)).toEqual([
      "github",
      "gitlab",
      "postman",
      "jira",
      "linear",
      "notion",
      "figma",
      "slack",
      "circleci",
      "semaphore",
      "vercel",
      "render",
      "railway",
      "sentry",
      "datadog",
      "grafana",
      "cloudflare",
      "stripe",
      "supabase",
      "mongodb-atlas",
      "neon",
      "prisma",
    ]);
    for (const entry of MCP_CATALOG) {
      expect(entry.url.startsWith("https://")).toBe(true);
      expect(entry.instruction).toBeDefined();
    }
    const github = MCP_CATALOG.find((entry) => entry.id === "github");
    const circleci = MCP_CATALOG.find((entry) => entry.id === "circleci");
    const semaphore = MCP_CATALOG.find((entry) => entry.id === "semaphore");
    expect(github?.auth).toBe("AUTH_HEADERS");
    expect(github?.headerName).toBe("Authorization");
    expect(circleci?.auth).toBe("AUTH_HEADERS");
    expect(circleci?.headerName).toBe("Authorization");
    expect(semaphore?.auth).toBe("AUTH_HEADERS");
    expect(semaphore?.headerName).toBe("Authorization");
    expect(
      MCP_CATALOG.filter((entry) => entry.id !== "github" && entry.id !== "circleci" && entry.id !== "semaphore").every(
        (entry) => entry.auth === "AUTH_OAUTH",
      ),
    ).toBe(true);
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

  it("prefills CircleCI header auth and a token instruction", () => {
    const circleci = MCP_CATALOG.find((entry) => entry.id === "circleci");
    expect(catalogConnectionDefaults(circleci)).toEqual({
      name: "circleci",
      url: "https://mcp.circleci.com/v1/mcp",
      auth: "AUTH_HEADERS",
      headers: [{ name: "Authorization" }],
      instruction: AGENT_RESOURCES_COPY.circleciInstruction,
    });
    expect(AGENT_RESOURCES_COPY.circleciInstruction).toEqual({
      before: "Create a ",
      href: CIRCLECI_PERSONAL_API_TOKEN_URL,
      label: "CircleCI personal API token",
      after: " and paste it here.",
    });
  });

  it("prefills Semaphore header auth and a token instruction", () => {
    const semaphore = MCP_CATALOG.find((entry) => entry.id === "semaphore");
    expect(catalogConnectionDefaults(semaphore)).toEqual({
      name: "semaphore",
      url: "https://mcp.semaphoreci.com/mcp",
      auth: "AUTH_HEADERS",
      headers: [{ name: "Authorization" }],
      instruction: AGENT_RESOURCES_COPY.semaphoreInstruction,
    });
    expect(AGENT_RESOURCES_COPY.semaphoreInstruction).toEqual({
      before: "Ask Semaphore support to enable MCP. Reset your ",
      href: SEMAPHORE_API_TOKEN_URL,
      label: "Semaphore API token",
      after: " and paste it here.",
    });
  });

  it("omits defaults for a custom MCP server", () => {
    expect(catalogConnectionDefaults(undefined)).toBeUndefined();
  });

  it("matches a saved server to a catalog entry by URL and auth", () => {
    expect(catalogEntryForResource({ url: "https://mcp.sentry.dev/mcp", auth: "AUTH_OAUTH" })?.id).toBe("sentry");
    expect(catalogEntryForResource({ url: "https://mcp.datadoghq.com/v1/mcp", auth: "AUTH_OAUTH" })?.id).toBe(
      "datadog",
    );
    expect(catalogEntryForResource({ url: "https://mcp.datadoghq.eu/v1/mcp", auth: "AUTH_OAUTH" })?.id).toBe("datadog");
    expect(catalogEntryForResource({ url: "https://mcp.datadoghq.eu/v1/mcp", auth: "AUTH_HEADERS" })).toBeUndefined();
    expect(catalogEntryForResource({ url: "https://mcp.linear.app/mcp", auth: "AUTH_OAUTH" })?.id).toBe("linear");
    expect(catalogEntryForResource({ url: "https://api.githubcopilot.com/mcp/", auth: "AUTH_HEADERS" })?.id).toBe(
      "github",
    );
    expect(catalogEntryForResource({ url: "https://mcp.circleci.com/v1/mcp", auth: "AUTH_HEADERS" })?.id).toBe(
      "circleci",
    );
    expect(catalogEntryForResource({ url: "https://mcp.semaphoreci.com/mcp", auth: "AUTH_HEADERS" })?.id).toBe(
      "semaphore",
    );
    expect(catalogEntryForResource({ url: "https://mcp.semaphoreci.com/mcp", auth: "AUTH_OAUTH" })).toBeUndefined();
    expect(catalogEntryForResource({ url: "https://mcp.sentry.dev/mcp", auth: "AUTH_HEADERS" })).toBeUndefined();
    expect(catalogEntryForResource({ url: "https://mcp.example.com/mcp", auth: "AUTH_OAUTH" })).toBeUndefined();
    expect(catalogEntryForResource(undefined)).toBeUndefined();
  });
});

describe("catalogOAuthResourceForEntry", () => {
  const sentry = MCP_CATALOG.find((entry) => entry.id === "sentry");
  const github = MCP_CATALOG.find((entry) => entry.id === "github");
  const circleci = MCP_CATALOG.find((entry) => entry.id === "circleci");
  const semaphore = MCP_CATALOG.find((entry) => entry.id === "semaphore");

  it("finds an OAuth server that already uses the catalog URL", () => {
    expect(sentry).toBeDefined();
    expect(
      catalogOAuthResourceForEntry(
        [
          { url: "https://mcp.example.com/mcp", auth: "AUTH_OAUTH" },
          { url: "https://mcp.sentry.dev/mcp", auth: "AUTH_OAUTH" },
        ],
        sentry!,
      ),
    ).toEqual({ url: "https://mcp.sentry.dev/mcp", auth: "AUTH_OAUTH" });
  });

  it("ignores a header server on the same catalog URL", () => {
    expect(sentry).toBeDefined();
    expect(
      catalogOAuthResourceForEntry([{ url: "https://mcp.sentry.dev/mcp", auth: "AUTH_HEADERS" }], sentry!),
    ).toBeUndefined();
  });

  it("resumes a Datadog server saved on another site", () => {
    const datadog = MCP_CATALOG.find((entry) => entry.id === "datadog");
    expect(datadog).toBeDefined();
    expect(
      catalogOAuthResourceForEntry(
        [{ name: "datadog", url: "https://mcp.us5.datadoghq.com/v1/mcp", auth: "AUTH_OAUTH" }],
        datadog!,
      ),
    ).toEqual({ name: "datadog", url: "https://mcp.us5.datadoghq.com/v1/mcp", auth: "AUTH_OAUTH" });
    expect(
      catalogOAuthResourceForEntry(
        [{ name: "custom", url: "https://mcp.us5.datadoghq.com/v1/mcp", auth: "AUTH_OAUTH" }],
        datadog!,
      ),
    ).toBeUndefined();
  });

  it("treats any connected Datadog site as the catalog server", () => {
    const datadog = MCP_CATALOG.find((entry) => entry.id === "datadog");
    expect(datadog).toBeDefined();
    expect(
      catalogEntryIsConnected(
        [{ url: "https://mcp.datadoghq.eu/v1/mcp", auth: "AUTH_OAUTH", oauthStatus: "OAUTH_STATUS_CONNECTED" }],
        datadog!,
      ),
    ).toBe(true);
    expect(
      catalogEntryIsConnected(
        [{ url: "https://mcp.datadoghq.eu/v1/mcp", auth: "AUTH_OAUTH", oauthStatus: "OAUTH_STATUS_NOT_CONNECTED" }],
        datadog!,
      ),
    ).toBe(false);
  });

  it("does not resume header catalog entries", () => {
    expect(github).toBeDefined();
    expect(
      catalogOAuthResourceForEntry([{ url: "https://api.githubcopilot.com/mcp/", auth: "AUTH_HEADERS" }], github!),
    ).toBeUndefined();
    expect(circleci).toBeDefined();
    expect(
      catalogOAuthResourceForEntry([{ url: "https://mcp.circleci.com/v1/mcp", auth: "AUTH_HEADERS" }], circleci!),
    ).toBeUndefined();
    expect(semaphore).toBeDefined();
    expect(
      catalogOAuthResourceForEntry([{ url: "https://mcp.semaphoreci.com/mcp", auth: "AUTH_HEADERS" }], semaphore!),
    ).toBeUndefined();
  });
});

describe("filterMCPCatalog", () => {
  it("returns every entry when the query is empty", () => {
    expect(filterMCPCatalog(MCP_CATALOG, "  ")).toEqual(MCP_CATALOG);
  });

  it("matches label, name, id, or category", () => {
    expect(filterMCPCatalog(MCP_CATALOG, "GitHub").map((entry) => entry.id)).toEqual(["github"]);
    expect(filterMCPCatalog(MCP_CATALOG, "linear").map((entry) => entry.id)).toEqual(["linear"]);
    expect(filterMCPCatalog(MCP_CATALOG, "sema").map((entry) => entry.id)).toEqual(["semaphore"]);
    expect(filterMCPCatalog(MCP_CATALOG, "circle").map((entry) => entry.id)).toEqual(["circleci"]);
    expect(
      filterMCPCatalog(MCP_CATALOG, "observability")
        .map((entry) => entry.id)
        .sort(),
    ).toEqual(["datadog", "grafana", "sentry"]);
  });
});

describe("groupMCPCatalog", () => {
  it("groups by category and sorts servers by name", () => {
    const groups = groupMCPCatalog(MCP_CATALOG);
    expect(groups.map((group) => group.id)).toEqual([
      "code",
      "issues",
      "docs",
      "design",
      "chat",
      "cicd",
      "observability",
      "infrastructure",
      "database",
    ]);
    expect(groups.find((group) => group.id === "code")?.entries.map((entry) => entry.label)).toEqual([
      "GitHub",
      "GitLab",
      "Postman",
    ]);
    expect(groups.find((group) => group.id === "issues")?.entries.map((entry) => entry.label)).toEqual([
      "Jira",
      "Linear",
    ]);
    expect(groups.find((group) => group.id === "docs")?.entries.map((entry) => entry.label)).toEqual(["Notion"]);
    expect(groups.find((group) => group.id === "design")?.entries.map((entry) => entry.label)).toEqual(["Figma"]);
    expect(groups.find((group) => group.id === "chat")?.entries.map((entry) => entry.label)).toEqual(["Slack"]);
    expect(groups.find((group) => group.id === "cicd")?.entries.map((entry) => entry.label)).toEqual([
      "CircleCI",
      "Railway",
      "Render",
      "Semaphore",
      "Vercel",
    ]);
    expect(groups.find((group) => group.id === "observability")?.entries.map((entry) => entry.label)).toEqual([
      "Datadog",
      "Grafana",
      "Sentry",
    ]);
    expect(groups.find((group) => group.id === "infrastructure")?.entries.map((entry) => entry.label)).toEqual([
      "Cloudflare",
      "Stripe",
    ]);
    expect(groups.find((group) => group.id === "database")?.entries.map((entry) => entry.label)).toEqual([
      "MongoDB Atlas",
      "Neon",
      "Prisma",
      "Supabase",
    ]);
  });

  it("omits categories that have no matching servers", () => {
    const github = MCP_CATALOG.find((entry) => entry.id === "github");
    expect(github).toBeDefined();
    expect(groupMCPCatalog([github!]).map((group) => group.id)).toEqual(["code"]);
  });
});
