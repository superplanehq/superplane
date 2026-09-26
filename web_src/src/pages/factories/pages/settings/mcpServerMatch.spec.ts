import { describe, expect, it } from "bun:test";

import {
  HEADER_MCP_RESOURCE,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
} from "../../__fixtures__/agentResourceFixtures";
import { canonicalMCPServerURL, connectedMCPResourceForURL } from "./mcpServerMatch";

describe("canonicalMCPServerURL", () => {
  it("treats a trailing slash as the same server", () => {
    expect(canonicalMCPServerURL("https://mcp.sentry.dev/mcp/")).toBe(
      canonicalMCPServerURL("https://mcp.sentry.dev/mcp"),
    );
  });

  it("returns empty for a blank value", () => {
    expect(canonicalMCPServerURL("  ")).toBe("");
  });
});

describe("connectedMCPResourceForURL", () => {
  it("finds a connected server at the same URL", () => {
    const sentry = { ...OAUTH_CONNECTED_RESOURCE, url: "https://mcp.sentry.dev/mcp" };
    expect(connectedMCPResourceForURL([sentry, OAUTH_NOT_CONNECTED_RESOURCE], "https://mcp.sentry.dev/mcp/")).toEqual(
      sentry,
    );
  });

  it("ignores a server that is not connected", () => {
    const sentry = { ...OAUTH_NOT_CONNECTED_RESOURCE, url: "https://mcp.sentry.dev/mcp" };
    expect(connectedMCPResourceForURL([sentry], "https://mcp.sentry.dev/mcp")).toBeUndefined();
  });

  it("ignores the server that is being edited", () => {
    expect(
      connectedMCPResourceForURL([HEADER_MCP_RESOURCE], HEADER_MCP_RESOURCE.url ?? "", HEADER_MCP_RESOURCE.id),
    ).toBeUndefined();
  });
});
