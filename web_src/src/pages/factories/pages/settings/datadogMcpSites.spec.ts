import { describe, expect, it } from "bun:test";

import { datadogMCPSiteForURL, datadogMCPURL, DATADOG_MCP_SITES, isDatadogMCPURL } from "./datadogMcpSites";

describe("datadog MCP sites", () => {
  it("maps each commercial site to its MCP host", () => {
    expect(DATADOG_MCP_SITES.map((site) => [site.id, site.url])).toEqual([
      ["datadoghq.com", "https://mcp.datadoghq.com/v1/mcp"],
      ["us3.datadoghq.com", "https://mcp.us3.datadoghq.com/v1/mcp"],
      ["us5.datadoghq.com", "https://mcp.us5.datadoghq.com/v1/mcp"],
      ["datadoghq.eu", "https://mcp.datadoghq.eu/v1/mcp"],
      ["ap1.datadoghq.com", "https://mcp.ap1.datadoghq.com/v1/mcp"],
      ["ap2.datadoghq.com", "https://mcp.ap2.datadoghq.com/v1/mcp"],
      ["uk1.datadoghq.com", "https://mcp.uk1.datadoghq.com/v1/mcp"],
    ]);
    expect(datadogMCPURL("datadoghq.com")).toBe("https://mcp.datadoghq.com/v1/mcp");
  });

  it("matches a saved Datadog MCP URL to its site", () => {
    expect(datadogMCPSiteForURL("https://mcp.datadoghq.eu/v1/mcp/")?.id).toBe("datadoghq.eu");
    expect(datadogMCPSiteForURL("https://MCP.US5.DATADOGHQ.COM/v1/mcp")?.id).toBe("us5.datadoghq.com");
    expect(isDatadogMCPURL("https://mcp.ap1.datadoghq.com/v1/mcp")).toBe(true);
    expect(isDatadogMCPURL("https://mcp.datadoghq.com/v1/mcp?subdomain=acme")).toBe(true);
    expect(isDatadogMCPURL("https://mcp.example.com/v1/mcp")).toBe(false);
    expect(isDatadogMCPURL("https://mcp.datadoghq.com/mcp")).toBe(false);
    expect(datadogMCPSiteForURL(undefined)).toBeUndefined();
  });
});
